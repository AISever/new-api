"""Rehearse the exact cutover dump while the published app is monitored."""
import hashlib,json,os,subprocess,time
from pathlib import Path
os.umask(0o077);S=Path.home()/'.local/state/new-api-security-release-20261018';j=json.loads((S/'journal.json').read_text());assert j['phase']=='observing';b=json.loads((S/'backup-state.json').read_text());root=Path(b['backup_dir']);dump=root/('database-precutover-'+str(int(j['started']))+'.dump');expected=json.loads((root/('cutover-before-'+str(int(j['started']))+'.json')).read_text())
name='new-api-release-restore-20261018-postgres'
def run(*args,timeout=120):
 p=subprocess.run(args,capture_output=True,text=True,timeout=timeout)
 if p.returncode:
  (S/'cutover-restore-error.txt').write_text(p.stderr);raise RuntimeError('cutover backup rehearsal failed; details kept privately')
 return p.stdout.strip()
c=json.loads(run('docker','inspect',name))[0];assert c['Config']['Labels'].get('codex.release')=='new-api-production-release-20261018';assert not c['HostConfig']['PortBindings']
net=json.loads(run('docker','network','inspect',next(iter(c['NetworkSettings']['Networks']))))[0];assert net['Internal']
run('docker','start',name)
for _ in range(30):
 if subprocess.run(['docker','exec',name,'pg_isready','-U','restore_admin'],capture_output=True).returncode==0:break
 time.sleep(1)
else:raise RuntimeError('isolated restore DB startup failed')
run('docker','exec',name,'psql','-Xq','-U','restore_admin','-d','postgres','-v','ON_ERROR_STOP=1','-c','CREATE DATABASE "new-api-cutover" OWNER restore_admin')
with open(dump,'rb') as f,open(root/'cutover-restore.stderr','wb') as err:
 subprocess.run(['docker','exec','-i',name,'pg_restore','-U','restore_admin','-d','new-api-cutover','--exit-on-error','--no-owner','--no-privileges'],stdin=f,stdout=subprocess.DEVNULL,stderr=err,check=True,timeout=900)
def sql(q):return run('docker','exec','-e','PGTZ=UTC',name,'psql','-XqAt','-U','restore_admin','-d','new-api-cutover','-v','ON_ERROR_STOP=1','-c',q)
actual={}
for table in ['users','tokens','user_subscriptions','subscription_plans','options','redemptions','top_ups','subscription_orders']:
 actual[table]=sql('SELECT count(*)::text||\'|\'||md5(coalesce(string_agg(h,\'\' ORDER BY h),\'\')) FROM (SELECT md5(row_to_json(t)::text) h FROM "'+table+'" t) x')
actual['channel_config']=sql("SELECT md5(coalesce(string_agg(h,'' ORDER BY h),'')) FROM (SELECT md5((to_jsonb(t)-'used_quota'-'response_time'-'test_time')::text) h FROM channels t) x")
assert actual==expected['snapshot'],'restored cutover financial/configuration tables differ'
fhy=json.loads(sql("SELECT row_to_json(x) FROM (SELECT s.*,u.username,p.quota_reset_period,p.quota_reset_custom_seconds FROM user_subscriptions s JOIN users u ON u.id=s.user_id JOIN subscription_plans p ON p.id=s.plan_id WHERE s.id=131) x"));assert fhy==expected['fhy131']
h=hashlib.sha256()
with open(dump,'rb') as f:
 for data in iter(lambda:f.read(1024*1024),b''):h.update(data)
meta={'path':str(dump),'bytes':dump.stat().st_size,'sha256':h.hexdigest()};(S/'cutover-dump-export.json').write_text(json.dumps(meta));run('docker','stop',name)
result={'exact_cutover_backup_restored':True,'financial_and_configuration_fingerprints_match':True,'fhy131_exact_match':True,'restore_network_internal':True,'backup':meta};(S/'cutover-restore-verification.json').write_text(json.dumps(result,indent=2));(root/'cutover-restore-verification.json').write_text(json.dumps(result,indent=2));print(json.dumps(result,indent=2))
