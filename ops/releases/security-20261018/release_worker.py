"""Guarded application-only production cutover with automatic image rollback."""
import copy,datetime,fcntl,hashlib,http.client,ipaddress,json,os,re,signal,socket,subprocess,sys,time,urllib.parse,urllib.request
from pathlib import Path
import probe_remote
os.umask(0o077)
S=Path.home()/'.local/state/new-api-security-release-20261018';B=json.loads((S/'backup-state.json').read_text());ROOT=Path(B['backup_dir'])
OLD=B['old_image'];NEW='sha256:d07e5682b016202338524f832d8206f2d99ac4b916a1391db92100cdfb9405e1'
VERSION='codex/sec-v0.13.2-p1+ba844d202+production';NAME='new-api-local';RETAIN='new-api-local-rollback-security-20261018'
J={'phase':'preflight','worker_pid':os.getpid(),'old_id':B['app_id'],'new_image':NEW,'version':VERSION,'started':time.time()}
INITIAL=json.loads((S/'initial-inspect.json').read_text());ORIGINAL=INITIAL['app'];CADDY=json.loads((S/'caddy-before.json').read_text())
RULE=['-p','tcp','-d','172.18.0.0/16','-m','conntrack','--ctstate','NEW','--ctorigdstport','3000','-m','comment','--comment','new-api-security-release-20261018','-j','REJECT','--reject-with','tcp-reset']
def save():
 f=S/'journal.tmp';f.write_text(json.dumps(J,indent=2));f.replace(S/'journal.json')
def phase(value,**kw):
 J.update(kw,phase=value,heartbeat=time.time());save();print(json.dumps({'phase':value,'beijing':datetime.datetime.now(datetime.timezone(datetime.timedelta(hours=8))).isoformat(),**kw}),flush=True)
def run(*args,timeout=120):
 p=subprocess.run(args,capture_output=True,text=True,timeout=timeout)
 if p.returncode:
  (S/'last-command-error.txt').write_text(p.stderr)
  raise RuntimeError('command failed: '+' '.join(args[:4]))
 return p.stdout.strip()
def inspect(n):
 p=subprocess.run(['docker','inspect',n],capture_output=True,text=True,timeout=15)
 return json.loads(p.stdout)[0] if p.returncode==0 else None
def sql(q):return probe_remote.sql(q)
def canonical(x):return json.dumps(x,sort_keys=True,separators=(',',':'))
def digest(x):return hashlib.sha256(canonical(x).encode()).hexdigest()
def guard_other_services():
 for k in ['postgres','redis','caddy','old_test']:
  old=INITIAL[k];now=inspect(old['Id']);assert now and now['Id']==old['Id'] and now['Image']==old['Image'] and now['State']['StartedAt']==old['State']['StartedAt'] and now['RestartCount']==old['RestartCount'] and now['State']['Running'],k
 assert hashlib.sha256(Path('/opt/snowlight/Caddyfile').read_bytes()).hexdigest()==B['manifest']['Caddyfile']['sha256']
def caddy_get(path='/config/'):
 return json.loads(run('docker','exec','snowlight-caddy','wget','-qO-','http://127.0.0.1:2019'+path))
def caddy_load(config):
 p=subprocess.run(['docker','exec','-i','snowlight-caddy','wget','-qO-','--header','Content-Type: application/json','--post-file=/dev/stdin','http://127.0.0.1:2019/load'],input=json.dumps(config),capture_output=True,text=True,timeout=30)
 if p.returncode:(S/'caddy-load-error.txt').write_text(p.stderr);raise RuntimeError('Caddy configuration load failed')
def gate():
 config=copy.deepcopy(CADDY);count=0
 def walk(x):
  nonlocal count
  if isinstance(x,dict):
   if x.get('handler')=='reverse_proxy' and any(u.get('dial')=='172.17.0.1:3000' for u in x.get('upstreams',[])):
    assert all(u.get('dial')=='172.17.0.1:3000' for u in x['upstreams']);x.clear();x.update({'handler':'static_response','status_code':503,'headers':{'Retry-After':['60'],'Content-Type':['application/json']},'body':'{"maintenance":"security-release-20261018"}'});count+=1;return
   for v in x.values():walk(v)
  elif isinstance(x,list):
   for v in x:walk(v)
 walk(config);assert count==2;assert canonical(caddy_get())==canonical(CADDY)
 phase('draining',maintenance_started=time.time());caddy_load(config)
 if subprocess.run(['sudo','-n','iptables','-C','DOCKER-USER',*RULE],capture_output=True).returncode:
  run('sudo','-n','iptables','-I','DOCKER-USER','1',*RULE)
 J['gate_active']=True;save()
def ungate():
 if canonical(caddy_get())!=canonical(CADDY):caddy_load(CADDY)
 while subprocess.run(['sudo','-n','iptables','-C','DOCKER-USER',*RULE],capture_output=True).returncode==0:
  run('sudo','-n','iptables','-D','DOCKER-USER',*RULE)
 J['gate_active']=False;save()
class DockerConnection(http.client.HTTPConnection):
 def connect(self):
  self.sock=socket.socket(socket.AF_UNIX,socket.SOCK_STREAM);self.sock.settimeout(self.timeout);self.sock.connect('/var/run/docker.sock')
def create_container():
 body=copy.deepcopy(ORIGINAL['Config']);body['Image']=NEW;body['Env']=[x for x in body['Env'] if not x.startswith('VERSION=')]+['VERSION='+VERSION];body['HostConfig']=copy.deepcopy(ORIGINAL['HostConfig'])
 assert body['HostConfig']['NetworkMode']=='new-api_default'
 body['NetworkingConfig']={'EndpointsConfig':{'new-api_default':{}}}
 conn=DockerConnection('localhost',timeout=60);conn.request('POST','/v'+run('docker','version','--format','{{.Server.APIVersion}}')+'/containers/create?name='+NAME,body=json.dumps(body),headers={'Content-Type':'application/json'});r=conn.getresponse();data=r.read();conn.close()
 if r.status!=201:(S/'docker-create-error.json').write_bytes(data);raise RuntimeError('candidate container creation failed')
 return json.loads(data)['Id']
def snapshot(exclude_probe=False):
 probe=json.loads((S/'probe-private.json').read_text())['id'] if exclude_probe else 0
 result={}
 for table,condition in [('users','id<>'+str(probe)),('tokens','user_id<>'+str(probe)),('user_subscriptions','true'),('subscription_plans','true'),('options','true'),('redemptions','true'),('top_ups','true'),('subscription_orders','true')]:
  result[table]=sql('SELECT count(*)::text||\'|\'||md5(coalesce(string_agg(h,\'\' ORDER BY h),\'\')) FROM (SELECT md5(row_to_json(t)::text) h FROM "'+table+'" t WHERE '+condition+') x')
 result['channel_config']=sql("SELECT md5(coalesce(string_agg(h,'' ORDER BY h),'')) FROM (SELECT md5((to_jsonb(t)-'used_quota'-'response_time'-'test_time')::text) h FROM channels t) x")
 return result
def fhy():
 return json.loads(sql("SELECT row_to_json(x) FROM (SELECT s.*,u.username,p.quota_reset_period,p.quota_reset_custom_seconds FROM user_subscriptions s JOIN users u ON u.id=s.user_id JOIN subscription_plans p ON p.id=s.plan_id WHERE s.id=131) x"))
def check_fhy(row):
 assert row['id']==131 and row['username']=='fhy' and row['user_id']==1267
 assert row['quota_reset_period']=='custom' and row['quota_reset_custom_seconds']==86400
 for key in ['last_reset_time','next_reset_time']:
  local=datetime.datetime.fromtimestamp(row[key],datetime.timezone(datetime.timedelta(hours=8)));assert local.hour==local.minute==local.second==0
 assert row['next_reset_time']-row['last_reset_time']==86400

def clear_quota_caches():
 env=dict(x.split('=',1) for x in ORIGINAL['Config']['Env'] if '='in x);url=urllib.parse.urlparse(env['REDIS_CONN_STRING']);addr=inspect('new-api-redis')['NetworkSettings']['Networks']['new-api_default']['IPAddress']
 sock=socket.create_connection((addr,url.port or 6379),timeout=10);f=sock.makefile('rb')
 def read():
  line=f.readline();kind=line[:1];v=line[1:-2]
  if kind==b'-':raise RuntimeError('Redis command rejected')
  if kind==b'+':return v
  if kind==b':':return int(v)
  if kind==b'$':
   n=int(v)
   if n<0:return None
   out=f.read(n);assert f.read(2)==b'\r\n';return out
  if kind==b'*':return [read() for _ in range(int(v))]
  raise RuntimeError('invalid Redis reply')
 def command(*items):
  parts=[x if isinstance(x,bytes) else str(x).encode() for x in items];sock.sendall(b'*'+str(len(parts)).encode()+b'\r\n'+b''.join(b'$'+str(len(p)).encode()+b'\r\n'+p+b'\r\n' for p in parts));return read()
 if url.password:
  if url.username:command('AUTH',urllib.parse.unquote(url.username),urllib.parse.unquote(url.password))
  else:command('AUTH',urllib.parse.unquote(url.password))
 command('SELECT',url.path.lstrip('/') or '0');counts={}
 for pattern in ['user:*','token:*']:
  cursor=b'0';keys=[]
  while True:
   cursor,batch=command('SCAN',cursor,'MATCH',pattern,'COUNT',500);keys+=batch
   if cursor==b'0':break
  keys=[k for k in keys if command('TYPE',k)==b'hash'];counts[pattern]=sum(command('DEL',k) for k in keys)
 f.close();sock.close();return counts

def health(version,public=False):
 urls=['http://127.0.0.1:3000/api/status']
 if public:urls+=['https://api.aisever.cn/api/status','https://newapi.aisever.cn/api/status']
 for url in urls:
  with urllib.request.urlopen(url,timeout=10) as r:
   d=json.load(r);assert r.status==200 and d.get('success') and d.get('data',{}).get('version')==version,url

def wait_health(version):
 for _ in range(60):
  try:health(version);return
  except Exception:time.sleep(1)
 raise RuntimeError('application health/version did not become ready')

def rollback():
 phase('rolling_back',rollback_started=time.time())
 current=inspect(NAME)
 if current and current['Id']!=B['app_id']:
  assert current['Image']==NEW and (not J.get('new_id') or current['Id']==J['new_id'])
  # Route new requests to maintenance before returning to the old image.
  try:
   if not J.get('gate_active'):gate()
  except Exception:pass
  if current['State']['Running']:
   deadline=time.time()+120
   while sum(x.get('num_requests',0) for x in caddy_get('/reverse_proxy/upstreams')) and time.time()<deadline:time.sleep(2)
   time.sleep(13)
  run('docker','update','--restart=no',current['Id']);run('docker','stop','-t','15',current['Id']);run('docker','rename',current['Id'],'new-api-local-failed-security-'+current['Id'][:9])
 old=inspect(B['app_id']);assert old and old['Image']==OLD
 if old['Name']!='/'+NAME:run('docker','rename',old['Id'],NAME)
 policy=ORIGINAL['HostConfig']['RestartPolicy'];run('docker','update','--restart='+policy['Name'],old['Id'])
 if not inspect(old['Id'])['State']['Running']:run('docker','start',old['Id'])
 old_version=next(x.split('=',1)[1] for x in ORIGINAL['Config']['Env'] if x.startswith('VERSION='));wait_health(old_version);ungate();health(old_version,True);guard_other_services();phase('rolled_back',rollback_verified=True)

def deploy():
 assert json.loads((S/'restore-verification.json').read_text())['recoverability_verified'];assert json.loads((S/'redis-restore-verification.json').read_text())['redis_backup_restore_verified'];assert json.loads((S/'probe-baseline.json').read_text())['chat_success']
 assert inspect(NAME)['Id']==B['app_id'] and inspect(NAME)['Image']==OLD and inspect(NAME)['State']['Running'];assert inspect(NEW)['Id']==NEW;guard_other_services()
 net=json.loads(run('docker','network','inspect','new-api_default'))[0];assert any(x.get('Subnet')=='172.18.0.0/16' for x in net['IPAM']['Config'])
 assert not inspect(RETAIN);row=fhy();check_fhy(row)
 # Do not interrupt the already configured natural midnight reset.
 midnight_before=json.loads((S/'restore-verification.json').read_text())['fhy131_restored']
 while fhy()['next_reset_time']-time.time()<150:
  assert fhy()['next_reset_time']>time.time()-120,'old service midnight reset overdue'
  phase('waiting_for_midnight_reset');time.sleep(20)
 J['old_service_midnight_reset_observed']=fhy()['last_reset_time']==midnight_before['next_reset_time'];save()
 row=fhy();check_fhy(row);assert row['next_reset_time']>time.time()+150
 scheduled_deadline=time.time()+180
 while int(sql("SELECT count(*) FROM user_subscriptions WHERE status='active' AND ((next_reset_time>0 AND next_reset_time<"+str(int(time.time())+90)+") OR (end_time>0 AND end_time<"+str(int(time.time())+90)+"))")):
  assert time.time()<scheduled_deadline,'scheduled subscription activity did not settle'
  phase('waiting_for_scheduled_subscription_task');time.sleep(20)
 gate();pid=inspect(NAME)['State']['Pid'];deadline=time.time()+120
 while True:
  active=sum(x.get('num_requests',0) for x in caddy_get('/reverse_proxy/upstreams') if x.get('address')=='172.17.0.1:3000')
  sockets=run('sudo','-n','nsenter','-t',str(pid),'-n','ss','-Htn','state','established');external=[]
  for line in sockets.splitlines():
   fields=line.split();local,peer=fields[-2:]
   if local.rsplit(':',1)[-1]!='3000' and peer.rsplit(':',1)[-1] not in ['5432','6379']:external.append(line)
  if not active and not external:break
  if time.time()>deadline:raise RuntimeError('inflight requests did not drain safely')
  time.sleep(2)
 run('sudo','-n','nsenter','-t',str(pid),'-n','ss','-K','-t','state','established','sport','=',':3000')
 time.sleep(13)
 stable=snapshot();time.sleep(2);assert snapshot()==stable,'financial writes still active after drain'
 phase('cutover',drained=True);run('docker','update','--restart=no',B['app_id']);run('docker','stop','-t','30',B['app_id'])
 before=snapshot();other_before=snapshot(True);row=fhy();check_fhy(row)
 before_record={'snapshot':before,'non_probe_snapshot':other_before,'fhy131':row}
 (S/'cutover-before.json').write_text(json.dumps(before_record,indent=2));(ROOT/('cutover-before-'+str(int(J['started']))+'.json')).write_text(json.dumps(before_record,indent=2))
 with open(ROOT/('database-precutover-'+str(int(J['started']))+'.dump'),'wb') as out,open(ROOT/('precutover-'+str(int(J['started']))+'.stderr'),'wb') as err:
  subprocess.run(['docker','exec','new-api-postgres','pg_dump','-U','newapi','-d','new-api','-Fc','--compress=1','--no-owner','--no-privileges'],stdout=out,stderr=err,check=True,timeout=120)
 with open(ROOT/('database-precutover-'+str(int(J['started']))+'.dump'),'rb') as source,open(ROOT/('precutover-'+str(int(J['started']))+'-list.txt'),'wb') as out:
  subprocess.run(['docker','exec','-i','new-api-postgres','pg_restore','--list'],stdin=source,stdout=out,stderr=subprocess.PIPE,check=True,timeout=20)
 J['cleared_quota_caches']=clear_quota_caches();save()
 run('docker','rename',B['app_id'],RETAIN);new_id=create_container();phase('starting_candidate',new_id=new_id)
 run('docker','start',new_id);wait_health(VERSION)
 now=inspect(new_id);assert now['Image']==NEW
 before_env=dict(x.split('=',1) for x in ORIGINAL['Config']['Env']);after_env=dict(x.split('=',1) for x in now['Config']['Env']);before_env['VERSION']=VERSION;assert before_env==after_env
 for key in ['Cmd','Entrypoint','WorkingDir','User','ExposedPorts','Volumes']:
  assert now['Config'].get(key)==ORIGINAL['Config'].get(key),key
 expected_labels=dict(inspect(NEW)['Config'].get('Labels') or {});expected_labels.update(ORIGINAL['Config'].get('Labels') or {})
 assert (now['Config'].get('Labels') or {})==expected_labels,'unexpected container labels'
 for key in ['Binds','PortBindings','RestartPolicy','LogConfig','NetworkMode','Memory','NanoCpus','PidsLimit','Privileged','ReadonlyRootfs','CapAdd','CapDrop','SecurityOpt','Dns','ExtraHosts','Ulimits','ShmSize']:
  assert now['HostConfig'].get(key)==ORIGINAL['HostConfig'].get(key),key
 assert snapshot()==before,'startup changed persistent quota/configuration';assert fhy()==row,'startup changed fhy subscription';guard_other_services()
 phase('candidate_smoke');smoke=probe_remote.probe('candidate');assert snapshot(True)==other_before,'verification changed an existing user or configuration'
 ungate();health(VERSION,True);guard_other_services();phase('observing',published_at=time.time(),candidate_smoke=smoke,configuration_preserved=True,all_existing_quota_rows_preserved=True,fhy131_unchanged_at_cutover=True)
 failures=0;until=time.time()+1800;samples=0;midnight_seen=False
 while time.time()<until:
  try:
   health(VERSION,True);c=inspect(NAME);assert c['Id']==new_id and c['State']['Running'] and not c['State']['OOMKilled'] and c['RestartCount']==0;guard_other_services();r=fhy();check_fhy(r)
   if time.time()>=row['next_reset_time']+90:
    assert r['last_reset_time']>=row['next_reset_time'] and r['next_reset_time']>time.time(),'midnight reset overdue';midnight_seen=True
   failures=0;samples+=1;J.update(heartbeat=time.time(),health_samples=samples,midnight_reset_observed=midnight_seen,fhy131_latest=r);save()
  except Exception as error:
   failures+=1;(S/'monitor-last-error.txt').write_text(repr(error))
   if failures>=2:raise RuntimeError('production monitoring detected repeated regression')
  time.sleep(15 if not failures else 3)
 probe_remote.disable_probe()
 health(VERSION,True);guard_other_services();phase('verified',health_samples=samples,observation_seconds=1800,midnight_reset_observed=midnight_seen,probe_disabled=True)

def ensure_new_run():
 # A completed or active release must never be replayed as a new deployment.
 assert not (S/'journal.json').exists(),'release journal exists; inspect it instead of replaying this single-use release'

def handle_release_failure(error):
 (S/'release-error.txt').write_text(repr(error));J['error']=str(error);save()
 if J.get('phase')=='preflight' and not J.get('new_id') and not J.get('gate_active'):
  phase('aborted_before_cutover');return
 try:rollback()
 except BaseException as rollback_error:
  (S/'rollback-error.txt').write_text(repr(rollback_error));phase('rollback_failed',error='release and automatic rollback require immediate intervention')

def on_term(signum,frame):raise RuntimeError('release worker interrupted')
if __name__=='__main__':
 lock=open(S/'release.lock','a');fcntl.flock(lock,fcntl.LOCK_EX|fcntl.LOCK_NB)
 signal.signal(signal.SIGTERM,on_term)
 if '--check-only' in sys.argv:
  guard_other_services();assert canonical(caddy_get())==canonical(CADDY);caddy_load(CADDY);assert canonical(caddy_get())==canonical(CADDY);snapshot();check_fhy(fhy());print('Production release preflight and no-op configuration load passed',flush=True)
 elif '--rollback' in sys.argv:
  if (S/'journal.json').exists():J.update(json.loads((S/'journal.json').read_text()))
  J['worker_pid']=os.getpid();rollback()
 else:
  ensure_new_run();save()
  try:deploy()
  except BaseException as error:
   handle_release_failure(error);raise
