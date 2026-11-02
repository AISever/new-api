"""Recover the cutover dump only in the owned restore PostgreSQL, never live data."""
import pathlib,json,subprocess,ast,hashlib
S=pathlib.Path.home()/'.local/state/new-api-invitation-production-20261102-r2';J=json.loads((S/'journal.json').read_text());B=json.loads((S/'backup-state.json').read_text());root=pathlib.Path(B['backup_dir']);prefix='new-api-release-restore-20261102-r2';pg=prefix+'-postgres';database='invitation_cutover_check'
assert J['phase'] in ['observing','verified']
def run(*a,timeout=600):return subprocess.check_output(a,text=True,stderr=subprocess.PIPE,timeout=timeout).strip()
info=json.loads(run('docker','inspect',pg))[0];assert info['Config']['Labels']['codex.release']=='new-api-production-release-20261102-r2' and not info['State']['Running']
network=json.loads(run('docker','network','inspect',prefix))[0];assert network['Internal'] and not info['HostConfig']['PortBindings']
dump=root/('database-precutover-'+str(int(J['started']))+'.dump');assert dump.exists()
run('docker','start',pg)
try:
 import time
 for _ in range(30):
  if subprocess.run(['docker','exec',pg,'pg_isready','-U','restore_admin'],capture_output=True).returncode==0:break
  time.sleep(1)
 else:raise RuntimeError('owned cutover restore DB unavailable')
 assert run('docker','exec',pg,'psql','-XqAt','-U','restore_admin','-d','postgres','-c',"SELECT count(*) FROM pg_database WHERE datname='"+database+"'")=='0'
 run('docker','exec',pg,'psql','-XqAt','-U','restore_admin','-d','postgres','-c','CREATE DATABASE '+database)
 with dump.open('rb') as inp,open(S/'cutover-restore.stderr','wb') as err:
  subprocess.run(['docker','exec','-i',pg,'pg_restore','-U','restore_admin','-d',database,'--no-owner','--no-privileges','--exit-on-error'],stdin=inp,stdout=subprocess.DEVNULL,stderr=err,check=True,timeout=900)
 def sql(q):return run('docker','exec','-e','PGTZ=UTC',pg,'psql','-XqAt','-U','restore_admin','-d',database,'-v','ON_ERROR_STOP=1','-c',q)
 code=ast.parse((S/'release_worker.py').read_text());env={'sql':sql,'json':json}
 for name in ['snapshot','fhy']:
  fn=next(n for n in code.body if isinstance(n,ast.FunctionDef) and n.name==name);exec(compile(ast.Module(body=[fn],type_ignores=[]),'cutover-verify','exec'),env)
 before=json.loads((S/'cutover-before.json').read_text());assert env['snapshot']()==before['snapshot'];assert env['fhy']()==before['fhy131']
 result={'cutover_dump_restore_verified':True,'financial_and_configuration_fingerprints_match':True,'fhy131_full_row_matches':True,'dump_sha256':hashlib.sha256(dump.read_bytes()).hexdigest(),'dump_bytes':dump.stat().st_size,'production_database_restored':False}
 (S/'cutover-backup-verification.json').write_text(json.dumps(result,indent=2));print(json.dumps(result,indent=2))
finally:run('docker','stop','-t','15',pg)
