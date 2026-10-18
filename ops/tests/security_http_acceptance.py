#!/usr/bin/env python3
"""Synthetic acceptance; can only use containers bearing this task's label."""
import argparse
import concurrent.futures
import http.cookiejar
import json
from pathlib import Path
import secrets
import subprocess
import time
import urllib.error
import urllib.request

p = argparse.ArgumentParser()
p.add_argument('--prefix', required=True)
p.add_argument('--port', type=int, default=13018)
p.add_argument('--version', required=True)
a = p.parse_args()
assert a.prefix.startswith('new-api-security-') and a.port >= 1024
for kind in ['app', 'postgres', 'redis', 'fixture']:
    c = json.loads(subprocess.check_output(['docker', 'inspect', a.prefix+'-'+kind]))[0]
    assert c['Config']['Labels'].get('codex.task') == 'new-api-security-20261018'
base = 'http://127.0.0.1:'+str(a.port)
opener = urllib.request.build_opener(urllib.request.HTTPCookieProcessor(http.cookiejar.CookieJar()))
user_id = None
checks = []

def request(path, body=None, method=None, bearer=None, raw=None):
    headers = {'Content-Type': 'application/json'}
    if user_id: headers['New-Api-User'] = str(user_id)
    if bearer: headers['Authorization'] = 'Bearer sk-'+bearer
    data = raw if raw is not None else (json.dumps(body).encode() if body is not None else None)
    req = urllib.request.Request(base+path, data=data, headers=headers, method=method)
    try:
        r = opener.open(req, timeout=20)
    except urllib.error.HTTPError as e:
        r = e
    text = r.read().decode()
    try: value = json.loads(text)
    except ValueError: value = text
    return r.code, value

def ok(path, body=None, method=None):
    code, value = request(path, body, method)
    assert code == 200 and value.get('success'), (path, code, value)
    return value.get('data')

def sql(statement):
    return subprocess.check_output(['docker', 'exec', '-i', a.prefix+'-postgres', 'psql', '-U', 'security', '-d', 'security_test', '-At', '-v', 'ON_ERROR_STOP=1', '-c', statement], text=True).strip()

def quota():
    return int(sql('SELECT quota FROM users WHERE id='+str(user_id)))

def check(name, fn):
    fn()
    checks.append({'check': name, 'passed': True})

assert ok('/api/status')['version'] == a.version
password = secrets.token_hex(12)
root = Path.home()/'.local/state/new-api-security-tests'/a.prefix.removeprefix('new-api-security-')
credential = root/'http-credentials.json'
if not credential.exists():
    credential.write_text(json.dumps({'username':'secadmin','password':password}))
    credential.chmod(0o600)
login = json.loads(credential.read_text())
if not ok('/api/setup')['status']:
    ok('/api/setup', dict(login,confirmPassword=login['password']))
user_id = ok('/api/user/login', login)['id']
ok('/api/subscription/self/preference',{'billing_preference':'wallet_only'},'PUT')
for key, value in [('ModelRatio', {'security-chat':1,'security-image':1}), ('ModelPrice', {'security-image':0.01}), ('CompletionRatio', {'security-chat':2}), ('GroupRatio', {'default':1,'denied':1})]:
    ok('/api/option/', {'key':key,'value':json.dumps(value)}, 'PUT')
if int(sql("SELECT count(*) FROM channels WHERE name='synthetic-security'")) == 0:
    ok('/api/channel/', {'mode':'single','channel':{'name':'synthetic-security','type':1,'key':'synthetic-upstream','base_url':'http://'+a.prefix+'-fixture:8080','models':'security-chat,security-image','group':'default','status':1}})
# Baseline channel additions are propagated by a periodic memory-cache sync.
# Restart only the owned test app to load the persisted fixture configuration.
subprocess.run(['docker','restart',a.prefix+'-app'],check=True,stdout=subprocess.DEVNULL)
for _ in range(40):
    try:
        if ok('/api/status')['version'] == a.version: break
    except Exception: pass
    time.sleep(1)
else: raise AssertionError('test configuration reload failed')
token = secrets.token_hex(24)
sql('INSERT INTO tokens (user_id,key,status,name,expired_time,remain_quota,unlimited_quota,"group") VALUES ('+str(user_id)+",'"+token+"',1,'acceptance',-1,10000000,false,'default')")
chat = {'model':'security-chat','messages':[{'role':'user','content':'synthetic'}], 'temperature':0, 'top_p':0, 'presence_penalty':0, 'frequency_penalty':0}

def normal_chat():
    before = quota()
    code, value = request('/v1/chat/completions', chat, bearer=token)
    assert code == 200 and value['choices'][0]['message']['content'] == 'fixture-ok', (code,value)
    time.sleep(0.4)
    assert quota() < before
check('chat relay and wallet debit', normal_chat)

def stream():
    code, value = request('/v1/chat/completions', dict(chat,stream=True), bearer=token)
    assert code == 200 and 'fixture-ok' in value and '[DONE]' in value, (code,value)
check('SSE streaming', stream)

def bounds():
    time.sleep(0.4)
    before = quota()
    for path, payload in [('/v1/images/generations',{'model':'security-image','prompt':'test','n':18446744073686646784}),('/v1/chat/completions',dict(chat,max_tokens=2147483647)),('/v1/billing/charge',{'model':'security-image','quantity':9223372036854775807})]:
        code, value = request(path, payload, bearer=token)
        assert code == 400, (path,code,value)
    assert quota() == before
check('overflow inputs reject without quota mutation', bounds)

def body_limit():
    for path in ['/api/stripe/webhook','/api/creem/webhook','/api/user/epay/notify','/api/subscription/epay/notify']:
        code, _ = request(path,raw=b'x'*(512*1024+1))
        assert code == 413, (path,code)
check('anonymous webhook body limits', body_limit)

def private_charge():
    before = quota()
    code, value = request('/v1/billing/charge',{'model':'security-image','quantity':3},bearer=token)
    assert code == 200 and value['quantity'] == 3 and value['quota'] == 15000, (code,value)
    time.sleep(0.4)
    assert before-quota() == 15000
check('private billing endpoint and quantity multiplier', private_charge)

def concurrent_settings():
    before = quota()
    def work(i):
        if i % 2:
            return request('/v1/billing/charge',{'model':'security-image'},bearer=token)
        return request('/api/user/self',{'language':'en'},'PUT')
    with concurrent.futures.ThreadPoolExecutor(max_workers=4) as pool:
        results = list(pool.map(work, range(8)))
    assert all(code == 200 for code,_ in results), results
    assert all(v.get('success',v.get('object')=='billing_charge') for _,v in results), results
    time.sleep(0.8)
    assert before-quota() == 20000, (before,quota())
check('concurrent self settings cannot restore consumed wallet quota', concurrent_settings)

def token_not_serialized():
    sentinel = secrets.token_hex(16)
    sql("UPDATE users SET access_token='"+sentinel+"' WHERE id="+str(user_id))
    _, users = request('/api/user/')
    assert sentinel not in json.dumps(users) and 'access_token' not in json.dumps(users)
check('admin user list excludes access token', token_not_serialized)

def explicit_zero():
    text = subprocess.check_output(['docker','exec',a.prefix+'-app','wget','-qO-','http://'+a.prefix+'-fixture:8080/stats'],text=True)
    records = json.loads(text)
    forwarded = next(r['body'] for r in records if r['path']=='/v1/chat/completions' and not r['body'].get('stream'))
    for key in ['temperature','top_p','presence_penalty','frequency_penalty']: assert forwarded[key] == 0
check('explicit zero parameters reach upstream', explicit_zero)

def midnight_subscription():
    midnight = int(sql("SELECT extract(epoch FROM date_trunc('day',now() AT TIME ZONE 'Asia/Shanghai') AT TIME ZONE 'Asia/Shanghai')::bigint"))
    sql("INSERT INTO subscription_plans (id,title,enabled,duration_unit,duration_value,total_amount,effective_groups,quota_reset_period,quota_reset_custom_seconds) VALUES (991318,'synthetic-midnight',true,'month',1,100000,'[\"default\"]','custom',86400) ON CONFLICT (id) DO NOTHING")
    sql("UPDATE user_subscriptions SET status='expired' WHERE user_id="+str(user_id)+" AND plan_id=991318")
    sid = sql("INSERT INTO user_subscriptions (user_id,plan_id,amount_total,amount_used,start_time,end_time,status,last_reset_time,next_reset_time) VALUES ("+str(user_id)+',991318,100000,99,'+str(midnight-172800)+','+str(midnight+172800)+",'active',"+str(midnight-86400)+','+str(midnight)+') RETURNING id').splitlines()[0]
    ok('/api/subscription/self/preference',{'billing_preference':'subscription_only'},'PUT')
    before = quota()
    code, value = request('/v1/billing/charge',{'model':'security-image','quantity':2},bearer=token)
    assert code == 200 and value['billing_source']=='subscription', (code,value)
    time.sleep(0.4)
    row = sql('SELECT amount_used,last_reset_time,next_reset_time FROM user_subscriptions WHERE id='+sid)
    assert row == '10000|'+str(midnight)+'|'+str(midnight+86400), row
    assert quota() == before
    ok('/api/subscription/self/preference',{'billing_preference':'wallet_only'},'PUT')
check('private subscription resets at Beijing midnight and bills subscription only', midnight_subscription)

def restart():
    before = quota()
    subprocess.run(['docker','restart',a.prefix+'-app'],check=True,stdout=subprocess.DEVNULL)
    for _ in range(40):
        try:
            if ok('/api/status')['version']==a.version: break
        except Exception: pass
        time.sleep(1)
    else: raise AssertionError('test app restart failed')
    normal_chat()
    assert quota() < before
check('test container restart preserves wallet and configuration', restart)

result = {'version':a.version,'test_prefix':a.prefix,'synthetic_upstream':True,'checks':checks,'passed':len(checks)}
(root/'http-acceptance.json').write_text(json.dumps(result,indent=2))
print(json.dumps(result,indent=2))
