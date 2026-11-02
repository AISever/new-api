"""Read-only production discovery and owned isolated restore preparation; no gate."""
import copy,datetime,hashlib,json,os,secrets,subprocess,urllib.parse,time
from pathlib import Path
os.umask(0o077)
S=Path.home()/'.local/state/new-api-reward-backfill-20261102';assert json.loads((S/'owner.json').read_text())['approved'];assert not (S/'context.json').exists() and not (S/'journal.json').exists()
expected=json.loads((S/'expectations.json').read_text())
def run(*args):return subprocess.check_output(args,stderr=subprocess.PIPE,text=True).strip()
def inspect(name):return json.loads(run('docker','inspect',name))[0]
app=inspect('new-api-local');assert app['Id']==expected['app_id'] and app['Image']==expected['image'] and app['State']['Running'] and app['RestartCount']==0
assert hashlib.sha256((S/'tool').read_bytes()).hexdigest()==expected['tool_sha256']
support=Path.home()/'.local/state/new-api-invitation-production-20261102-r2'
for name,digest in expected['support_hashes'].items():assert hashlib.sha256((support/name).read_bytes()).hexdigest()==digest
initial={'app':app,'postgres':inspect('new-api-postgres'),'redis':inspect('new-api-redis'),'caddy':inspect('snowlight-caddy'),'old_test':inspect('new-api-test')}
all_items=json.loads(run('docker','inspect',*run('docker','ps','-aq').split()));others={}
production_dsn=dict(x.split('=',1) for x in app['Config']['Env'] if '='in x)['SQL_DSN'];writers=[]
for item in all_items:
 env=dict(x.split('=',1) for x in item['Config'].get('Env',[]) if '='in x)
 if item['State']['Running']:
  dsn=env.get('SQL_DSN','');parsed=urllib.parse.urlparse(dsn)
  if parsed.path=='/new-api':assert dsn==production_dsn;writers.append(item['Id'])
  if item['Id']!=app['Id']:
   assert not any(m.get('Source') in ['/opt/new-api/data','/opt/new-api/logs'] for m in item.get('Mounts',[])),'shared production filesystem writer'
 if item['Id']!=app['Id']:others[item['Name']]={'id':item['Id'],'image':item['Image'],'started':item['State']['StartedAt'],'restarts':item['RestartCount'],'running':item['State']['Running']}
assert writers==[app['Id']]
env=dict(x.split('=',1) for x in app['Config']['Env'] if '='in x);assert not env.get('LOG_SQL_DSN') and '\n' not in production_dsn
(S/'tool.env').write_text('BACKFILL_DSN='+production_dsn+'\nLOG_SQL_DSN=\n')
caddy=json.loads(run('docker','exec','snowlight-caddy','wget','-qO-','http://127.0.0.1:2019/config/'))
network=json.loads(run('docker','network','inspect','new-api_default'))[0];assert network['IPAM']['Config'][0]['Subnet']=='172.18.0.0/16'
# Docker inspect network is a different API object; kept separately from containers.
stamp=datetime.datetime.now(datetime.timezone(datetime.timedelta(hours=9))).strftime('%Y%m%d-%H%M%S')
root=Path('/opt/new-api-backups')/('redemption-backfill-'+stamp);run('sudo','-n','mkdir',str(root));run('sudo','-n','chown',str(os.getuid())+':'+str(os.getgid()),str(root));root.chmod(0o700)
(root/'owner.json').write_text(json.dumps({'operation':'redemption-backfill-20261102','client_timezone':'Asia/Tokyo'}))
(root/'app-inspect.json').write_text(json.dumps(app));(root/'caddy-runtime.json').write_text(json.dumps(caddy));run('sudo','-n','cp','--preserve=all','/opt/snowlight/Caddyfile',str(root/'Caddyfile'));run('sudo','-n','chown',str(os.getuid())+':'+str(os.getgid()),str(root/'Caddyfile'));(root/'Caddyfile').chmod(0o600)
context={'initial':initial,'caddy':caddy,'subnet':'172.18.0.0/16','caddy_file_sha256':hashlib.sha256((root/'Caddyfile').read_bytes()).hexdigest(),'backup_dir':str(root)}
(S/'context.json').write_text(json.dumps(context));(S/'other-services.json').write_text(json.dumps(others))
with (root/'current-image.tar').open('wb') as out,open(S/'image-save-stderr.txt','wb') as err:subprocess.run(['docker','image','save',app['Image']],stdout=out,stderr=err,check=True,timeout=120)
name='new-api-reward-backfill-20261102-postgres';assert subprocess.run(['docker','inspect',name],capture_output=True).returncode!=0
pg_env=S/'restore-postgres.env';pg_env.write_text('POSTGRES_USER=restore_admin\nPOSTGRES_PASSWORD='+secrets.token_hex(24)+'\nPOSTGRES_DB=postgres\n')
run('docker','run','-d','--name',name,'--label','codex.operation=redemption-backfill-20261102','--network','none','--memory=512m','--cpus=0.75','--pids-limit=128','--restart=no','--env-file',str(pg_env),initial['postgres']['Image'],'postgres','-c','shared_buffers=64MB','-c','max_connections=20')
for _ in range(30):
 if subprocess.run(['docker','exec',name,'pg_isready','-U','restore_admin'],capture_output=True).returncode==0:break
 time.sleep(1)
else:raise RuntimeError('isolated restore database unavailable')
# Validate the exact maintenance JSON without loading it into production.
import operation
operation.initialize();maintenance=operation.maintenance_config(caddy)
r=subprocess.run(['docker','exec','-i','snowlight-caddy','caddy','validate','--config','/dev/stdin'],input=json.dumps(maintenance),text=True,capture_output=True,timeout=30);(S/'maintenance-validation.txt').write_text(r.stdout+r.stderr);assert r.returncode==0
(S/'prepared.json').write_text(json.dumps({'prepared':True,'production_service_unchanged':True,'backup_dir':str(root),'other_container_count':len(others),'restore_container_network':'none'}))
print((S/'prepared.json').read_text())
