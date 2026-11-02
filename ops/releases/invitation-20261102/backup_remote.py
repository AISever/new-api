"""Production-consistent snapshot backup. Never stops or mutates the app."""
import csv, datetime, hashlib, io, json, os, re, shutil, subprocess, tarfile, time
from pathlib import Path
os.umask(0o077)
STATE=Path.home()/'.local/state/new-api-invitation-production-20261102-r2'
OLD='sha256:d07e5682b016202338524f832d8206f2d99ac4b916a1391db92100cdfb9405e1'
def run(*args,timeout=600):
 p=subprocess.run(args,capture_output=True,text=True,timeout=timeout)
 if p.returncode:raise RuntimeError('command failed: '+' '.join(args[:4])+'; details retained in private backup')
 return p.stdout.strip()
def inspect(name):return json.loads(run('docker','inspect',name))[0]
def psql(q):return run('docker','exec','-e','PGTZ=UTC','new-api-postgres','psql','-XqAt','-U','newapi','-d','new-api','-v','ON_ERROR_STOP=1','-c',q)
def fingerprint_sql(tables):
 rows=[]
 for t in tables:
  quoted='"'+t.replace('"','""')+'"';literal="'"+t.replace("'","''")+"'"
  rows.append('SELECT '+literal+" AS table_name,count(*)::bigint AS rows,md5(coalesce(string_agg(row_hash,'' ORDER BY row_hash),'')) AS content_md5 FROM (SELECT md5(row_to_json(t)::text) AS row_hash FROM public."+quoted+' t) x')
 return ' UNION ALL '.join(rows)
app=inspect('new-api-local');assert app['Image']==OLD and app['State']['Running']
assert json.loads(run('docker','image','inspect','new-api:security-ba844d202'))[0]['Id']==OLD
stamp=datetime.datetime.now(datetime.timezone(datetime.timedelta(hours=8))).strftime('%Y%m%d-%H%M%S')
root=Path('/opt/new-api-backups')/('invitation-'+stamp)
run('sudo','-n','mkdir','-p',str(root));run('sudo','-n','chown',str(os.getuid())+':'+str(os.getgid()),str(root));root.chmod(0o700)
state={'backup_dir':str(root),'app_id':app['Id'],'old_image':OLD,'created_beijing':stamp}
(STATE/'backup-state.json').write_text(json.dumps(state));(root/'owner.json').write_text(json.dumps({'release':'invitation-20261102-r2','purpose':'production-backup'}))
for name in ['new-api-local','new-api-postgres','new-api-redis','snowlight-caddy','new-api-test']:
 (root/(name+'.inspect.json')).write_text(json.dumps(inspect(name)))
shutil.copyfile(STATE/'caddy-before.json',root/'caddy-runtime.json')
run('sudo','-n','cp','--preserve=all','/opt/snowlight/Caddyfile',str(root/'Caddyfile'));run('sudo','-n','chown',str(os.getuid())+':'+str(os.getgid()),str(root/'Caddyfile'));(root/'Caddyfile').chmod(0o600)
with open(root/'globals.sql','wb') as out:
 subprocess.run(['docker','exec','new-api-postgres','pg_dumpall','-U','newapi','--globals-only'],stdout=out,stderr=open(root/'globals.stderr','wb'),check=True,timeout=120)
tables=psql("SELECT tablename FROM pg_tables WHERE schemaname='public' ORDER BY tablename").splitlines()
(root/'tables.json').write_text(json.dumps(tables))
proc=subprocess.Popen(['docker','exec','-i','-e','PGTZ=UTC','new-api-postgres','psql','-XqAt','-U','newapi','-d','new-api','-v','ON_ERROR_STOP=1'],stdin=subprocess.PIPE,stdout=subprocess.PIPE,stderr=open(root/'snapshot.stderr','wb'),text=True,bufsize=1)
try:
 proc.stdin.write('BEGIN ISOLATION LEVEL REPEATABLE READ READ ONLY; SELECT pg_export_snapshot();\n');proc.stdin.flush()
 snapshot=proc.stdout.readline().strip();assert re.fullmatch(r'[0-9A-F-]+',snapshot), 'failed to export a consistent snapshot'
 print(json.dumps({'phase':'dumping_consistent_database','tables':len(tables),'backup_dir':str(root)}),flush=True)
 with open(root/'database.dump','wb') as out,open(root/'dump.stderr','wb') as err:
  subprocess.run(['docker','exec','new-api-postgres','pg_dump','-U','newapi','-d','new-api','--format=custom','--compress=1','--snapshot='+snapshot,'--no-owner','--no-privileges'],stdout=out,stderr=err,check=True,timeout=900)
 q='BEGIN ISOLATION LEVEL REPEATABLE READ READ ONLY; SET TRANSACTION SNAPSHOT '+"'"+snapshot+"'; COPY ("+fingerprint_sql(tables)+") TO STDOUT WITH (FORMAT CSV); COMMIT;"
 with open(root/'database-fingerprints.csv','w') as out,open(root/'fingerprint.stderr','w') as err:
  subprocess.run(['docker','exec','-e','PGTZ=UTC','new-api-postgres','psql','-XqAt','-U','newapi','-d','new-api','-v','ON_ERROR_STOP=1','-c',q],stdout=out,stderr=err,check=True,timeout=900)
finally:
 if proc.poll() is None:
  proc.stdin.write('ROLLBACK;\n\\q\n');proc.stdin.flush();proc.wait(timeout=20)
print(json.dumps({'phase':'archiving_application_files_and_baseline_image'}),flush=True)
with open(root/'files.stderr','w') as err:
 p=subprocess.run(['sudo','-n','tar','--acls','--xattrs','--numeric-owner','-czf',str(root/'application-files.tar.gz'),'-C','/opt/new-api','data','logs'],stderr=err,timeout=900)
 if p.returncode not in [0,1]:raise RuntimeError('application archive failed')
 state['online_log_archive_changed_while_read']=p.returncode==1
run('sudo','-n','chown',str(os.getuid())+':'+str(os.getgid()),str(root/'application-files.tar.gz'));(root/'application-files.tar.gz').chmod(0o600)
with open(root/'baseline-image.tar','wb') as out,open(root/'image.stderr','wb') as err:
 subprocess.run(['docker','image','save','new-api:security-ba844d202'],stdout=out,stderr=err,check=True,timeout=600)
manifest={}
for name in ['database.dump','database-fingerprints.csv','globals.sql','application-files.tar.gz','baseline-image.tar','new-api-local.inspect.json','caddy-runtime.json','Caddyfile']:
 p=root/name;h=hashlib.sha256()
 with open(p,'rb') as f:
  for chunk in iter(lambda:f.read(1024*1024),b''):h.update(chunk)
 manifest[name]={'bytes':p.stat().st_size,'sha256':h.hexdigest()}
(root/'manifest.json').write_text(json.dumps(manifest,indent=2));state.update({'manifest':manifest,'tables':len(tables),'backup_complete':True});(STATE/'backup-state.json').write_text(json.dumps(state));(root/'backup-state.json').write_text(json.dumps(state,indent=2))
assert inspect('new-api-local')['Id']==app['Id'] and inspect('new-api-local')['State']['Running']
print(json.dumps({'backup_complete':True,'backup_dir':str(root),'tables':len(tables),'manifest':manifest,'online_log_archive_changed_while_read':state['online_log_archive_changed_while_read']},indent=2))
