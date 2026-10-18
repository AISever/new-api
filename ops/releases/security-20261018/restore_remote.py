"""Restore the production backup into an internal network; no production writes."""
import csv, hashlib, io, json, os, re, secrets, subprocess, tarfile, time
from pathlib import Path
os.umask(0o077)
S=Path.home()/'.local/state/new-api-security-release-20261018';state=json.loads((S/'backup-state.json').read_text());assert state['backup_complete'];root=Path(state['backup_dir']);label='new-api-production-release-20261018';prefix='new-api-release-restore-20261018'
def run(*args,timeout=600):
 p=subprocess.run(args,capture_output=True,text=True,timeout=timeout)
 if p.returncode:
  (root/'restore-command-error.txt').write_text(p.stderr)
  raise RuntimeError('isolated restore command failed; details retained privately: '+' '.join(args[:4]))
 return p.stdout.strip()
def inspect(n):return json.loads(run('docker','inspect',n))[0]
def sha(p):
 h=hashlib.sha256()
 with open(p,'rb') as f:
  for b in iter(lambda:f.read(1024*1024),b''):h.update(b)
 return h.hexdigest()
for name,meta in state['manifest'].items():assert sha(root/name)==meta['sha256'],name
for kind in ['postgres','redis','app']:
 assert subprocess.run(['docker','inspect',prefix+'-'+kind],capture_output=True).returncode!=0, 'refusing to overwrite existing restore container'
assert inspect('new-api-local')['Id']==state['app_id'] and inspect('new-api-local')['State']['Running']
# Docker 29 may normalize a legacy image config on export, changing its ID.
# Verify filesystem/config equivalence, then load an UNTAGGED legacy stream so
# the production rollback tag cannot be overwritten by that normalization.
original_image=json.loads(run('docker','image','inspect',state['old_image']))[0]
with tarfile.open(root/'baseline-image.tar') as archive:
 manifest=json.load(archive.extractfile('manifest.json'))
 assert len(manifest)==1 and manifest[0]['RepoTags']==['new-api:kkidc-production-bc32807fc']
 config_bytes=archive.extractfile(manifest[0]['Config']).read();config=json.loads(config_bytes)
 assert config['rootfs']['diff_ids']==original_image['RootFS']['Layers']
 for key in ['Env','Cmd','Entrypoint','WorkingDir','User','ExposedPorts','Volumes']:
  assert config['config'].get(key)==original_image['Config'].get(key),key
 normalized_image='sha256:'+hashlib.sha256(config_bytes).hexdigest()
 manifest[0]['RepoTags']=[]
 with open(root/'image-restore.stdout','wb') as out,open(root/'image-restore.stderr','wb') as err:
  loader=subprocess.Popen(['docker','image','load'],stdin=subprocess.PIPE,stdout=out,stderr=err)
  with tarfile.open(fileobj=loader.stdin,mode='w|') as stream:
   payload=json.dumps(manifest).encode();member=tarfile.TarInfo('manifest.json');member.size=len(payload);stream.addfile(member,io.BytesIO(payload))
   for name in [manifest[0]['Config'],*manifest[0]['Layers']]:
    member=archive.getmember(name);stream.addfile(member,archive.extractfile(member))
  loader.stdin.close();assert loader.wait(timeout=180)==0
assert json.loads(run('docker','image','inspect','new-api:kkidc-production-bc32807fc'))[0]['Id']==state['old_image']
normalized_image=re.search(r'Loaded image ID: (sha256:[0-9a-f]{64})',(root/'image-restore.stdout').read_text()).group(1)
assert json.loads(run('docker','image','inspect',normalized_image))[0]['RootFS']['Layers']==original_image['RootFS']['Layers']
restore_files=root/'restored-files';restore_files.mkdir(mode=0o700)
files=0;total=0
with tarfile.open(root/'application-files.tar.gz','r:gz') as t:
 t.extractall(restore_files,filter='data')
 for m in t.getmembers():
  if m.isfile():
   h=hashlib.sha256()
   with t.extractfile(m) as f:
    for b in iter(lambda:f.read(1024*1024),b''):h.update(b)
   assert sha(restore_files/m.name)==h.hexdigest(),m.name
   files+=1;total+=m.size
print(json.dumps({'phase':'filesystem_and_image_restore_verified','files':files,'bytes':total}),flush=True)
run('docker','network','create','--internal','--label','codex.release='+label,prefix)
secret=secrets.token_hex(24);(root/'restore-postgres.env').write_text('POSTGRES_USER=restore_admin\nPOSTGRES_DB=postgres\nPOSTGRES_PASSWORD='+secret+'\n')
pg_image=inspect('new-api-postgres')['Image'];redis_image=inspect('new-api-redis')['Image']
def start(kind,memory,image,opts=(),cmd=()):
 run('docker','run','-d','--name',prefix+'-'+kind,'--label','codex.release='+label,'--network',prefix,'--memory='+memory,'--cpus=0.75','--pids-limit=128','--restart=no',*opts,image,*cmd)
start('postgres','512m',pg_image,['--env-file',str(root/'restore-postgres.env')],['postgres','-c','max_connections=20','-c','shared_buffers=64MB'])
for _ in range(50):
 if subprocess.run(['docker','exec',prefix+'-postgres','pg_isready','-U','restore_admin'],capture_output=True).returncode==0:break
 time.sleep(1)
else:raise RuntimeError('restore PostgreSQL startup failed')
with open(root/'globals.sql','rb') as f,open(root/'restore-globals.stderr','wb') as e:
 subprocess.run(['docker','exec','-i',prefix+'-postgres','psql','-Xq','-U','restore_admin','-d','postgres','-v','ON_ERROR_STOP=1'],stdin=f,stdout=subprocess.DEVNULL,stderr=e,check=True,timeout=120)
run('docker','exec',prefix+'-postgres','psql','-Xq','-U','restore_admin','-d','postgres','-v','ON_ERROR_STOP=1','-c','CREATE DATABASE "new-api" OWNER newapi')
print(json.dumps({'phase':'restoring_all_production_tables_into_isolated_database'}),flush=True)
with open(root/'database.dump','rb') as f,open(root/'restore-database.stderr','wb') as e:
 subprocess.run(['docker','exec','-i',prefix+'-postgres','pg_restore','-U','restore_admin','-d','new-api','--exit-on-error','--no-owner','--no-privileges'],stdin=f,stdout=subprocess.DEVNULL,stderr=e,check=True,timeout=900)
tables=json.loads((root/'tables.json').read_text());parts=[]
for t in tables:
 quoted='"'+t.replace('"','""')+'"';literal="'"+t.replace("'","''")+"'"
 parts.append('SELECT '+literal+",count(*)::bigint,md5(coalesce(string_agg(row_hash,'' ORDER BY row_hash),'')) FROM (SELECT md5(row_to_json(t)::text) row_hash FROM public."+quoted+' t) x')
q='COPY ('+' UNION ALL '.join(parts)+') TO STDOUT WITH (FORMAT CSV)'
actual=run('docker','exec','-e','PGTZ=UTC',prefix+'-postgres','psql','-XqAt','-U','restore_admin','-d','new-api','-v','ON_ERROR_STOP=1','-c',q,timeout=900)
source={r[0]:r[1:] for r in csv.reader(io.StringIO((root/'database-fingerprints.csv').read_text()))};restored={r[0]:r[1:] for r in csv.reader(io.StringIO(actual))};assert source==restored,'restored table rows differ from the exported consistent snapshot'
(root/'restored-database-fingerprints.csv').write_text(actual+'\n')
print(json.dumps({'phase':'all_tables_exact_restore_match','tables':len(source),'rows':sum(int(r[0]) for r in source.values())}),flush=True)
start('redis','96m',redis_image,[],['redis-server','--maxmemory','64mb','--maxmemory-policy','noeviction'])
app_env=dict(x.split('=',1) for x in inspect('new-api-local')['Config']['Env'] if '=' in x)
app_env.update({'SQL_DSN':'postgresql://restore_admin:'+secret+'@'+prefix+'-postgres:5432/new-api?sslmode=disable','REDIS_CONN_STRING':'redis://'+prefix+'-redis','SESSION_SECRET':secrets.token_hex(24),'NODE_TYPE':'slave','VERSION':'ba844d202-restore-check'})
assert not app_env.get('LOG_SQL_DSN')
(root/'restore-app.env').write_text('\n'.join(k+'='+v for k,v in app_env.items())+'\n')
start('app','512m','sha256:d07e5682b016202338524f832d8206f2d99ac4b916a1391db92100cdfb9405e1',['--env-file',str(root/'restore-app.env'),'-v',str(restore_files/'data')+':/data:ro'],['--log-dir','/tmp/restore-logs'])
health=None
for _ in range(60):
 try:
  health=json.loads(run('docker','exec',prefix+'-app','wget','-qO-','http://127.0.0.1:3000/api/status'))
  if health.get('success'):break
 except Exception:pass
 time.sleep(1)
else:raise RuntimeError('candidate failed startup against restored production data')
fhy=run('docker','exec',prefix+'-postgres','psql','-XqAt','-U','restore_admin','-d','new-api','-c',"SELECT row_to_json(x) FROM (SELECT s.id,s.user_id,u.username,s.amount_total,s.amount_used,s.last_reset_time,s.next_reset_time FROM user_subscriptions s JOIN users u ON u.id=s.user_id WHERE s.id=131) x")
result={'recoverability_verified':True,'backup_dir':str(root),'tables_verified':len(source),'total_rows_verified':sum(int(r[0]) for r in source.values()),'file_count_verified':files,'file_bytes_verified':total,'baseline_image_restore_verified':True,'baseline_archive_loaded_image_id':normalized_image,'baseline_archive_rootfs_identical':True,'candidate_boots_restored_production_data':True,'candidate_image_id':inspect(prefix+'-app')['Image'],'restore_network_internal':True,'restore_has_no_published_ports':all(not inspect(prefix+'-'+k)['HostConfig']['PortBindings'] for k in ['app','postgres','redis']),'fhy131_restored':json.loads(fhy)}
assert inspect('new-api-local')['Id']==state['app_id'] and inspect('new-api-local')['State']['Running']
for kind in ['app','redis','postgres']:
 assert inspect(prefix+'-'+kind)['Config']['Labels'].get('codex.release')==label
 run('docker','stop','-t','10',prefix+'-'+kind)
(root/'restore-verification.json').write_text(json.dumps(result,indent=2));(S/'restore-verification.json').write_text(json.dumps(result,indent=2));print(json.dumps(result,indent=2))
