"""Back up Redis replication RDB and verify an isolated, networkless restore."""
import hashlib,json,os,secrets,shutil,subprocess,time,urllib.parse
from pathlib import Path
os.umask(0o077)
S=Path.home()/'.local/state/new-api-invitation-production-20261102-r2';state=json.loads((S/'backup-state.json').read_text());root=Path(state['backup_dir'])
def run(*args,timeout=180):
 p=subprocess.run(args,capture_output=True,text=True,timeout=timeout)
 if p.returncode:
  (root/'redis-restore-error.txt').write_text(p.stderr)
  raise RuntimeError('Redis backup/restore command failed; details kept privately')
 return p.stdout.strip()
def inspect(n):return json.loads(run('docker','inspect',n))[0]
app=inspect('new-api-local');assert app['Id']==state['app_id']
env=dict(x.split('=',1) for x in app['Config']['Env'] if '=' in x);conn=urllib.parse.urlparse(env['REDIS_CONN_STRING'])
args=['docker','exec']
if conn.password:args+=['-e','REDISCLI_AUTH='+urllib.parse.unquote(conn.password)]
args+=['new-api-redis','redis-cli']
if conn.username:args+=['--user',urllib.parse.unquote(conn.username)]
with open(root/'redis.rdb','wb') as out,open(root/'redis-backup.stderr','wb') as err:
 subprocess.run(args+['--rdb','-'],stdout=out,stderr=err,check=True,timeout=180)
assert (root/'redis.rdb').read_bytes()[:5]==b'REDIS'
hash_value=hashlib.sha256((root/'redis.rdb').read_bytes()).hexdigest()
name='new-api-release-redis-restore-20261102-r2';assert subprocess.run(['docker','inspect',name],capture_output=True).returncode!=0
copy=root/'restored-redis';copy.mkdir(mode=0o700);shutil.copyfile(root/'redis.rdb',copy/'dump.rdb');run('sudo','-n','chown','-R','999:999',str(copy))
run('docker','run','-d','--name',name,'--label','codex.release=new-api-production-release-20261102-r2','--network','none','--memory=256m','--cpus=0.5','--pids-limit=128','--restart=no','-v',str(copy)+':/data',inspect('new-api-redis')['Image'],'redis-server','--save','','--appendonly','no')
for _ in range(40):
 p=subprocess.run(['docker','exec',name,'redis-cli','PING'],capture_output=True,text=True)
 if p.returncode==0 and 'PONG' in p.stdout:break
 time.sleep(1)
else:raise RuntimeError('restored Redis did not become healthy')
check=run('docker','exec',name,'redis-check-rdb','/data/dump.rdb')
result={'redis_backup_restore_verified':True,'bytes':(root/'redis.rdb').stat().st_size,'sha256':hash_value,'check_rdb_success':True,'network':'none','ports_published':bool(inspect(name)['HostConfig']['PortBindings']),'source_selected_database':conn.path.lstrip('/') or '0','restored_selected_db_keys':int(run('docker','exec',name,'redis-cli','-n',conn.path.lstrip('/') or '0','DBSIZE'))}
assert not result['ports_published'];assert inspect(name)['Config']['Labels']['codex.release']=='new-api-production-release-20261102-r2';run('docker','stop',name)
(root/'redis-restore-verification.json').write_text(json.dumps(result,indent=2));(S/'redis-restore-verification.json').write_text(json.dumps(result,indent=2));manifest=json.loads((root/'manifest.json').read_text());manifest['redis.rdb']={'bytes':result['bytes'],'sha256':hash_value};(root/'manifest.json').write_text(json.dumps(manifest,indent=2))
print(json.dumps(result,indent=2))
