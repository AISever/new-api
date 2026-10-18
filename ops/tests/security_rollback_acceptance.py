#!/usr/bin/env python3
"""Probe the deployed baseline image against only this task's synthetic DB."""
import hashlib
import http.cookiejar
import json
from pathlib import Path
import subprocess
import time
import urllib.request

prefix = 'new-api-security-20261018'
label = 'new-api-security-20261018'
root = Path.home()/'.local/state/new-api-security-tests/20261018'
name = prefix+'-rollback'

def run(*args):
    return subprocess.check_output(args,text=True,stderr=subprocess.PIPE).strip()

def inspect(name):
    return json.loads(run('docker','inspect',name))[0]

def production():
    return {n:{k:inspect(n)[k] for k in ['Id','Image','RestartCount','State']} for n in ['new-api-local','new-api-postgres','new-api-redis','snowlight-caddy','new-api-test']}

def sql(statement):
    return run('docker','exec',prefix+'-postgres','psql','-U','security','-d','security_test','-At','-v','ON_ERROR_STOP=1','-c',statement)

def state():
    return sql("SELECT id,quota,used_quota,setting FROM users WHERE username='secadmin'")+sql('SELECT id,amount_used,last_reset_time,next_reset_time FROM user_subscriptions ORDER BY id')

def schema():
    return hashlib.sha256(sql("SELECT table_name,column_name,data_type,coalesce(column_default,'') FROM information_schema.columns WHERE table_schema='public' ORDER BY table_name,ordinal_position").encode()).hexdigest()

for kind in ['app','postgres','redis','fixture']:
    assert inspect(prefix+'-'+kind)['Config']['Labels'].get('codex.task')==label
assert subprocess.run(['docker','inspect',name],capture_output=True).returncode != 0
baseline = inspect('new-api-local')
assert baseline['Image']=='sha256:13a6bcdccd8424b3e6c962d3f072f2cdc14158e07b256f34dcd4650f790455d7'
before = production()
data_before, schema_before = state(), schema()
lines = (root/'app.env').read_text().splitlines()
assert any('@'+prefix+'-postgres:5432/security_test' in line for line in lines)
assert 'REDIS_CONN_STRING=redis://'+prefix+'-redis' in lines
lines = [line for line in lines if not line.startswith('VERSION=')]
lines.append('VERSION=codex/release-v0.13.2-p1+bc32807fc+kkidc-production')
env = root/'rollback.env';env.write_text('\n'.join(lines)+'\n');env.chmod(0o600)
opener = urllib.request.build_opener(urllib.request.HTTPCookieProcessor(http.cookiejar.CookieJar()))
def request(base,path,body=None,headers=None):
    req=urllib.request.Request(base+path,data=json.dumps(body).encode() if body is not None else None,headers=headers or {'Content-Type':'application/json'})
    with opener.open(req,timeout=15) as r: return json.load(r)

subprocess.run(['docker','stop',prefix+'-app'],check=True,stdout=subprocess.DEVNULL)
try:
    run('docker','run','-d','--name',name,'--label','codex.task='+label,'--network',prefix,'--memory=512m','--cpus=0.5','--pids-limit=128','--restart=no','--env-file',str(env),baseline['Image'])
    c=inspect(name)
    assert c['Image']==baseline['Image']
    base='http://'+c['NetworkSettings']['Networks'][prefix]['IPAddress']+':3000'
    for _ in range(60):
        try:
            status=request(base,'/api/status')
            if status.get('success'):break
        except Exception:pass
        time.sleep(1)
    else:raise RuntimeError('baseline failed to read migrated synthetic DB')
    login=request(base,'/api/user/login',json.loads((root/'http-credentials.json').read_text()))
    assert login.get('success')
    user_id=login['data']['id']
    profile=request(base,'/api/user/self',headers={'New-Api-User':str(user_id)})
    assert profile.get('success')
    assert state()==data_before, 'startup/login must preserve wallet, setting, subscription quota and anchors'
    assert schema()==schema_before, 'rollback startup changed the schema'
    token=sql('SELECT key FROM tokens ORDER BY id DESC LIMIT 1')
    quota_before=int(sql('SELECT quota FROM users WHERE id='+str(user_id)))
    chat=request(base,'/v1/chat/completions',{'model':'security-chat','messages':[{'role':'user','content':'rollback synthetic'}]},headers={'Content-Type':'application/json','Authorization':'Bearer sk-'+token})
    assert chat['choices'][0]['message']['content']=='fixture-ok'
    time.sleep(1)
    quota_after=int(sql('SELECT quota FROM users WHERE id='+str(user_id)))
    assert quota_after < quota_before
    result={'baseline_image_id':c['Image'],'old_image_reads_candidate_schema':True,'schema_unchanged':True,'startup_preserves_wallet_subscription_and_anchors':True,'login_and_self_api':True,'synthetic_chat_and_wallet_debit':True,'debit':quota_before-quota_after}
finally:
    p=subprocess.run(['docker','inspect',name],capture_output=True,text=True)
    if p.returncode==0:
        assert json.loads(p.stdout)[0]['Config']['Labels'].get('codex.task')==label
        subprocess.run(['docker','stop',name],check=True,stdout=subprocess.DEVNULL)
    subprocess.run(['docker','start',prefix+'-app'],check=True,stdout=subprocess.DEVNULL)
assert production()==before
result['production_unchanged']=True
(root/'rollback-acceptance.json').write_text(json.dumps(result,indent=2))
print(json.dumps(result,indent=2))
