"""Single-use authorized compensation; no application image/config deployment."""
import copy,csv,fcntl,hashlib,io,json,os,re,signal,subprocess,sys,tarfile,time,urllib.request,urllib.error,urllib.parse
from pathlib import Path
import controls
os.umask(0o077)
SUFFIX=os.environ.get('BACKFILL_OPERATION_SUFFIX','');assert SUFFIX in ['', '-retry1']
S=Path.home()/('.local/state/new-api-reward-backfill-20261102'+SUFFIX)
SUPPORT=Path.home()/'.local/state/new-api-invitation-production-20261102-r2'
PREFIX='new-api-reward-backfill-20261102'+SUFFIX
LABEL='redemption-backfill-20261102'+SUFFIX

def initialize():
 global w,C,APP,ROOT,A,J
 C=json.loads((S/'context.json').read_text());APP=C['initial']['app'];ROOT=Path(C['backup_dir']);A=json.loads((S/'approved.json').read_text())
 sys.path.insert(0,str(SUPPORT));import release_worker
 w=release_worker;w.S=S;w.B={'app_id':APP['Id'],'manifest':{'Caddyfile':{'sha256':C['caddy_file_sha256']}}};w.ORIGINAL=APP;w.INITIAL=C['initial'];w.CADDY=C['caddy'];w.VERSION='codex/invite+b77dd3008+production'
 w.RULE=['-p','tcp','-d',C['subnet'],'-m','conntrack','--ctstate','NEW','--ctorigdstport','3000','-m','comment','--comment',LABEL,'-j','REJECT','--reject-with','tcp-reset']
 J=json.loads((S/'journal.json').read_text()) if (S/'journal.json').exists() else {'phase':'prepared','started':time.time(),'worker_pid':os.getpid(),'apply_started':False,'gate_active':False}
 w.J=J

def phase(name,**kw):w.phase(name,**kw)
def save():w.save()
def inspect(name):return w.inspect(name)
def sql(q):return w.sql(q)
def run(args,*,stdin=None,stdout=None,timeout=300):
 errfile=S/'last-operation-stderr.txt'
 try:errhandle=errfile.open('wb')
 except OSError:errhandle=open(os.devnull,'wb')
 with errhandle as err:
  p=subprocess.Popen(args,stdin=stdin,stdout=stdout if stdout is not None else subprocess.PIPE,stderr=err)
  until=time.time()+timeout
  while True:
   try:out,_=p.communicate(timeout=min(15,max(0.1,until-time.time())));break
   except subprocess.TimeoutExpired:
    J['heartbeat']=time.time();w.safe_save()
    if time.time()>=until:p.kill();p.communicate();raise RuntimeError('operation command timed out; private stderr retained')
 if p.returncode:raise RuntimeError('operation command failed; private stderr retained')
 return out.decode().strip() if out is not None else ''

def tool(mode,plan_name):
 name=PREFIX+'-tool-'+mode;assert inspect(name) is None
 args=['docker','run','--rm','--name',name,'--label','codex.operation='+LABEL,'--network','new-api_default','--memory=256m','--cpus=0.5','--pids-limit=96','--read-only','--tmpfs','/tmp:rw,noexec,nosuid,size=16m','--user',str(os.getuid())+':'+str(os.getgid()),'--env-file',str(S/'tool.env'),'-v',str(S)+':/work','--entrypoint','/work/tool',APP['Image'],'--driver','postgres']
 if mode=='preview':args+=['--before',str(A['before']),'--out','/work/'+plan_name]
 else:args+=['--apply','/work/'+plan_name,'--sha256',J['plan_hash']]
 result=json.loads(run(args,timeout=180));(S/('tool-'+mode+'-result.json')).write_text(json.dumps(result,indent=2));return result

def current_tables():return sql("SELECT tablename FROM pg_tables WHERE schemaname='public' ORDER BY tablename").splitlines()
def fingerprint_query(tables):
 parts=[]
 for table in tables:
  quoted='"'+table.replace('"','""')+'"';literal="'"+table.replace("'","''")+"'"
  parts.append('SELECT '+literal+",count(*)::bigint,md5(coalesce(string_agg(h,'' ORDER BY h),'')) FROM (SELECT md5(row_to_json(t)::text) h FROM public."+quoted+' t) x')
 return 'COPY ('+' UNION ALL '.join(parts)+') TO STDOUT WITH (FORMAT CSV)'
def fingerprints(tables,restore=False):
 args=['docker','exec','-e','PGTZ=UTC',PREFIX+'-postgres' if restore else 'new-api-postgres','psql','-XqAt','-U','restore_admin' if restore else 'newapi','-d','restore_data' if restore else 'new-api','-v','ON_ERROR_STOP=1','-c',fingerprint_query(tables)]
 return {r[0]:r[1:] for r in csv.reader(io.StringIO(run(args,timeout=180)))}
def digest_query(expression,table,condition='true'):
 return sql("SELECT count(*)::text||'|'||md5(coalesce(string_agg(h,'' ORDER BY h),'')) FROM (SELECT md5("+expression+") h FROM "+table+' t WHERE '+condition+') x')
def capture(logmax=None):
 initial=logmax is None
 if initial:logmax=int(sql('SELECT coalesce(max(id),0) FROM logs'))
 tables=current_tables();fp=fingerprints(tables if initial else [t for t in tables if t!='logs'])
 if initial:old='|'.join(fp['logs'])
 else:
  old=digest_query('row_to_json(t)::text','logs','id<='+str(logmax));new=digest_query('row_to_json(t)::text','logs','id>'+str(logmax));fp['logs']=controls.log_fingerprint(old,new)
 selected=json.loads(sql("SELECT json_object_agg(id,row_to_json(t)) FROM users t WHERE id IN (1,21)"))
 return {'fingerprints':fp,'users_normalized':digest_query("CASE WHEN id IN (1,21) THEN (to_jsonb(t)-'aff_quota'-'aff_history')::text ELSE to_jsonb(t)::text END",'users'),'old_logs':old,'logmax':logmax,'selected_users':selected,'fhy131':w.fhy()}
def rows(table,condition='true'):
 assert table in ['redemption_reward_backfills','logs']
 return json.loads(sql('SELECT coalesce(json_agg(row_to_json(t)),\'[]\'::json) FROM '+table+' t WHERE '+condition))
def audit(partial=False,persist=True,after=None):
 before=json.loads((S/'before.json').read_text());after=after if after is not None else capture(before['logmax']);ledger=rows('redemption_reward_backfills') if inspect_ledger() else [];logs=rows('logs','id>'+str(before['logmax']))
 result=controls.audit(before,after,A,ledger,logs,J['plan_hash'],partial)
 if persist:
  (S/'after-private.json').write_text(json.dumps(after));(S/'ledger-private.json').write_text(json.dumps(ledger));(S/'logs-private.json').write_text(json.dumps(logs))
 return result
def inspect_ledger():return sql("SELECT to_regclass('public.redemption_reward_backfills') IS NOT NULL")=='t'
def no_clients():
 n=int(sql("SELECT count(*) FROM pg_stat_activity WHERE datname='new-api' AND backend_type='client backend' AND pid<>pg_backend_pid()"));assert n==0,'unexpected production database client during quiescent window'

def maintenance_config(original):
 config=copy.deepcopy(original);count=0
 def walk(x):
  nonlocal count
  if isinstance(x,dict):
   if x.get('handler')=='reverse_proxy' and any(u.get('dial')=='172.17.0.1:3000' for u in x.get('upstreams',[])):
    assert all(u.get('dial')=='172.17.0.1:3000' for u in x['upstreams']);proxy=copy.deepcopy(x);x.clear();x.update({'handler':'subroute','routes':[{'handle':[{'handler':'static_response','status_code':503,'headers':{'Retry-After':['60'],'Content-Type':['application/json']},'body':json.dumps({'maintenance':LABEL})}],'terminal':True},{'handle':[proxy]}]});count+=1;return
   for val in x.values():walk(val)
  elif isinstance(x,list):
   for val in x:walk(val)
 walk(config);assert count==2;return config

def gate():
 assert w.canonical(w.caddy_get())==w.canonical(C['caddy']);assert subprocess.run(['sudo','-n','iptables','-C','DOCKER-USER',*w.RULE],capture_output=True).returncode!=0
 phase('draining',maintenance_started=time.time());w.caddy_load(maintenance_config(C['caddy']));run(['sudo','-n','iptables','-I','DOCKER-USER','1',*w.RULE]);J['gate_active']=True;save()

def backup_and_restore():
 phase('backup_and_restore')
 before=capture();(S/'before.json').write_text(json.dumps(before,indent=2));(ROOT/'before-private.json').write_text(json.dumps(before))
 tables=current_tables();(ROOT/'tables.json').write_text(json.dumps(tables))
 with (ROOT/'database.dump').open('wb') as out:run(['docker','exec','new-api-postgres','pg_dump','-U','newapi','-d','new-api','-Fc','--compress=1','--no-owner','--no-privileges'],stdout=out,timeout=180)
 with (ROOT/'globals.sql').open('wb') as out:run(['docker','exec','new-api-postgres','pg_dumpall','-U','newapi','--globals-only'],stdout=out,timeout=60)
 # All application writers are stopped; any new data writer invalidates the proof.
 no_clients();assert capture(before['logmax'])==before,'production data changed during backup'
 with (ROOT/'globals.sql').open('rb') as inp:run(['docker','exec','-i',PREFIX+'-postgres','psql','-XqAt','-U','restore_admin','-d','postgres','-v','ON_ERROR_STOP=1'],stdin=inp)
 run(['docker','exec',PREFIX+'-postgres','psql','-XqAt','-U','restore_admin','-d','postgres','-v','ON_ERROR_STOP=1','-c','CREATE DATABASE restore_data'])
 run(['docker','exec',PREFIX+'-postgres','pg_restore','-U','restore_admin','-d','restore_data','--no-owner','--no-privileges','--exit-on-error','--jobs=2','/backup/database.dump'],timeout=420)
 restored=fingerprints(tables,True);assert restored==before['fingerprints'],'restored table data differs'
 (ROOT/'restored-fingerprints.json').write_text(json.dumps(restored,indent=2))
 archive=ROOT/'application-files.tar.gz'
 run(['sudo','-n','tar','--acls','--xattrs','--numeric-owner','-I','gzip -1','-cf',str(archive),'-C','/opt/new-api','data','logs'],timeout=180);run(['sudo','-n','chown',str(os.getuid())+':'+str(os.getgid()),str(archive)]);archive.chmod(0o600)
 files=0;file_bytes=0;dest=ROOT/'restored-files';dest.mkdir(mode=0o700)
 with tarfile.open(archive,'r:gz') as tar:
  tar.extractall(dest,filter='data')
  for member in tar.getmembers():
   if member.isfile():
    h=hashlib.sha256();f=tar.extractfile(member)
    for block in iter(lambda:f.read(1024*1024),b''):h.update(block)
    assert hash_file(dest/member.name)==h.hexdigest();files+=1;file_bytes+=member.size
 # Image already exported while service was online; content is immutable.
 image=json.loads(run(['docker','image','inspect',APP['Image']]))[0]
 normalized_id=controls.verify_image(ROOT/'current-image.tar',image)
 # Docker may normalize export config serialization. Load only an untagged
 # stream, verify its content, and never overwrite the original runtime tag.
 with tarfile.open(ROOT/'current-image.tar') as src,tarfile.open(ROOT/'untagged-restore-image.tar','w') as dest:
  manifest=json.load(src.extractfile('manifest.json'));manifest[0]['RepoTags']=[];data=json.dumps(manifest).encode();member=tarfile.TarInfo('manifest.json');member.size=len(data);dest.addfile(member,io.BytesIO(data))
  for name in [manifest[0]['Config'],*manifest[0]['Layers']]:dest.addfile(src.getmember(name),src.extractfile(name))
 with (ROOT/'untagged-restore-image.tar').open('rb') as inp:loaded_output=run(['docker','image','load'],stdin=inp,timeout=90)
 (ROOT/'image-load-output.txt').write_text(loaded_output);normalized_id=controls.loaded_image_id(loaded_output)
 loaded=json.loads(run(['docker','image','inspect',normalized_id]))[0];assert loaded['RootFS']['Layers']==image['RootFS']['Layers'] and loaded['Config']==image['Config'];assert inspect('new-api-local')['Image']==APP['Image']
 redis_restore()
 proof={'database_restored':True,'all_tables_match':True,'files_restored':True,'image_verified':True,'redis_restored':True,'app_stopped':not inspect('new-api-local')['State']['Running'],'table_count':len(tables),'row_count':sum(int(v[0]) for v in restored.values()),'file_count':files,'file_bytes':file_bytes}
 controls.ready(proof);(S/'restore-proof.json').write_text(json.dumps(proof,indent=2));(ROOT/'restore-proof.json').write_text(json.dumps(proof,indent=2))
 manifest={}
 for name in ['database.dump','globals.sql','application-files.tar.gz','current-image.tar','app-inspect.json','caddy-runtime.json','Caddyfile','redis.rdb','restore-proof.json']:
  path=ROOT/name;manifest[name]={'sha256':hash_file(path),'bytes':path.stat().st_size}
 (ROOT/'manifest.json').write_text(json.dumps(manifest,indent=2));phase('recovery_verified',backup_dir=str(ROOT),table_count=len(tables))

def hash_file(path):
 h=hashlib.sha256()
 with path.open('rb') as f:
  for b in iter(lambda:f.read(1024*1024),b''):h.update(b)
 return h.hexdigest()
def redis_restore():
 env=dict(x.split('=',1) for x in APP['Config']['Env'] if '='in x);conn=urllib.parse.urlparse(env['REDIS_CONN_STRING']);args=['docker','exec']
 if conn.password:args+=['-e','REDISCLI_AUTH='+urllib.parse.unquote(conn.password)]
 args+=['new-api-redis','redis-cli']
 if conn.username:args+=['--user',urllib.parse.unquote(conn.username)]
 with (ROOT/'redis.rdb').open('wb') as out:run(args+['--rdb','-'],stdout=out,timeout=60)
 assert (ROOT/'redis.rdb').read_bytes()[:5]==b'REDIS';directory=ROOT/'restored-redis';directory.mkdir(mode=0o700);(directory/'dump.rdb').write_bytes((ROOT/'redis.rdb').read_bytes());run(['sudo','-n','chown','-R','999:999',str(directory)])
 name=PREFIX+'-redis';assert inspect(name) is None
 run(['docker','run','-d','--name',name,'--label','codex.operation='+LABEL,'--network','none','--memory=96m','--cpus=0.5','--pids-limit=96','--restart=no','-v',str(directory)+':/data',C['initial']['redis']['Image'],'redis-server','--save','','--appendonly','no'])
 for _ in range(30):
  if subprocess.run(['docker','exec',name,'redis-cli','PING'],capture_output=True).stdout.strip()==b'PONG':break
  time.sleep(1)
 else:raise RuntimeError('isolated Redis restore failed')
 run(['docker','exec',name,'redis-check-rdb','/data/dump.rdb']);run(['docker','stop','-t','10',name])

def verify_runtime(check_other=True):
 now=inspect('new-api-local');assert now['Id']==APP['Id'] and now['Image']==APP['Image'] and now['Config']==APP['Config'] and now['HostConfig']==APP['HostConfig'] and now['RestartCount']==APP['RestartCount'] and not now['State']['OOMKilled']
 if check_other:w.guard_other_services()

def stop_tools():
 for mode in ['preview','apply','repeat']:
  name=PREFIX+'-tool-'+mode;item=inspect(name)
  if item:
   assert item['Config']['Labels'].get('codex.operation')==LABEL
   if item['State']['Running']:run(['docker','stop','-t','10',name],timeout=30)

def recover():
 stop_tools();verify_runtime(check_other=False)
 if J.get('apply_started') and not (S/'financial-verified.json').exists():
  no_clients();result=audit(partial=True,persist=False)
  try:(S/'recovered-financial-state.json').write_text(json.dumps(result,indent=2))
  except OSError:pass
 if not inspect('new-api-local')['State']['Running']:run(['docker','start','new-api-local'])
 w.wait_health(w.VERSION);w.ungate();w.health(w.VERSION,True);verify_runtime(check_other=False);w.emergency_phase('recovered_service',finished_at=time.time())

def execute():
 verify_runtime();assert inspect('new-api-local')['State']['Running'];assert not inspect_ledger();assert hashlib.sha256((S/'tool').read_bytes()).hexdigest()=='85996477f88aa21d9687f75941251ffb2ed586d6b194e7905aeed9c99d9d6782'
 assert not (S/'financial-verified.json').exists();pre=tool('preview','preflight-plan.json');controls.scope(json.loads((S/'preflight-plan.json').read_text()),A)
 phase('waiting_for_quiet')
 until=time.time()+180
 while w.active_requests():
  assert time.time()<until,'production traffic did not become idle';J['heartbeat']=time.time();save();time.sleep(1)
 gate();until=time.time()+120
 while w.active_requests():
  assert time.time()<until,'requests did not drain';J['heartbeat']=time.time();save();time.sleep(1)
 w.require_known_ingress(APP['State']['Pid']);run(['sudo','-n','nsenter','-t',str(APP['State']['Pid']),'-n','ss','-K','-t','state','established','sport','=',':3000'])
 until=time.time()+w.settlement_seconds()
 while time.time()<until:J['heartbeat']=time.time();save();time.sleep(min(2,max(0,until-time.time())))
 stable=w.snapshot();time.sleep(2);assert w.snapshot()==stable,'writes not settled';run(['docker','stop','-t','30','new-api-local'],timeout=45);phase('stopped',app_stopped_at=time.time());no_clients()
 backup_and_restore();proof=json.loads((S/'restore-proof.json').read_text());proof['app_stopped']=not inspect('new-api-local')['State']['Running'];controls.ready(proof)
 result=tool('preview','production-plan.json');plan=json.loads((S/'production-plan.json').read_text());controls.scope(plan,A);assert hash_file(S/'production-plan.json')==result['sha256'];J['plan_hash']=result['sha256'];save()
 no_clients();assert capture(json.loads((S/'before.json').read_text())['logmax'])==json.loads((S/'before.json').read_text()),'data changed before apply'
 phase('applying',apply_started=True)
 applied=tool('apply','production-plan.json');assert applied['success'] and applied['result']['credited']==34 and applied['result']['already_completed']==0
 result=audit();(S/'initial-financial-audit.json').write_text(json.dumps(result,indent=2));phase('financial_verified',credited_codes=result['credited_codes'],total_quota=result['total_quota'])
 before_repeat=capture(json.loads((S/'before.json').read_text())['logmax']);repeat=tool('repeat','production-plan.json');assert repeat['success'] and repeat['result']['credited']==0 and repeat['result']['already_completed']==34;after_repeat=capture(before_repeat['logmax']);assert after_repeat==before_repeat;assert audit(after=after_repeat)==result
 (S/'financial-verified.json').write_text(json.dumps(result,indent=2))
 run(['docker','stop','-t','10',PREFIX+'-postgres'])
 run(['docker','start','new-api-local']);w.wait_health(w.VERSION);verify_runtime();w.ungate();w.health(w.VERSION,True);phase('observing',service_restored_at=time.time(),repeat_created_zero_rewards=True)
 for _ in range(20):
  w.health(w.VERSION,True);verify_runtime();w.check_fhy_transition(json.loads((S/'before.json').read_text())['fhy131'],w.fhy(),time.time());J['heartbeat']=time.time();save();time.sleep(15)
 phase('verified',finished_at=time.time(),observation_seconds=300,financial_summary=result)

def main():
 initialize();lock=acquire_lock('--recover' in sys.argv)
 if '--recover' in sys.argv:
  J['worker_pid']=os.getpid();recover();return
 assert not (S/'journal.json').exists(),'single-use operation refuses replay';J['worker_pid']=os.getpid();save()
 signal.signal(signal.SIGTERM,lambda *_:(_ for _ in ()).throw(RuntimeError('worker interrupted')))
 guarded_execute()

def guarded_execute():
 try:execute()
 except BaseException as err:
  J['error']=str(err);w.safe_save()
  try:recover()
  except BaseException as recovery_error:J.update(phase='recovery_requires_attention',recovery_error=type(recovery_error).__name__,heartbeat=time.time());w.safe_save()
  raise

def acquire_lock(recovery):
 lock=open(S/'operation.lock','rb' if recovery else 'ab');fcntl.flock(lock,fcntl.LOCK_EX|fcntl.LOCK_NB);return lock
if __name__=='__main__':main()
