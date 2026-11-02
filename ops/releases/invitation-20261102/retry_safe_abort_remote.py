"""One operator-initiated retry, only after a proven pre-cutover drain abort."""
import pathlib,json,subprocess,hashlib,time,shutil,os
import release_worker as m
os.umask(0o077);S=m.S;j=json.loads((S/'journal.json').read_text());current=m.inspect(m.NAME)
assert not (S/'retry-authorized.json').exists(),'bounded retry already consumed'
rule_absent=subprocess.run(['sudo','-n','iptables','-C','DOCKER-USER',*m.RULE],capture_output=True).returncode==1
routing_original=m.canonical(m.caddy_get())==m.canonical(m.CADDY)
assert m.can_retry_safe_abort(j,current,m.ORIGINAL,routing_original,rule_absent,bool(m.inspect(m.RETAIN))),'previous release is not a safely retryable pre-cutover abort'
m.guard_other_services();m.require_known_ingress(current['State']['Pid']);assert m.settlement_seconds()>=13
for file in ['release_worker.py.pid','guardian.py.pid']:
 if (S/file).exists():
  pid=(S/file).read_text().strip();p=pathlib.Path('/proc')/pid/'cmdline'
  if p.exists():assert not p.read_bytes(),'previous worker/watchdog still alive'
archive=S/'attempts'/('safe-drain-abort-'+str(int(j['started'])));archive.mkdir(parents=True)
record={'retry_number':1,'previous_journal_sha256':hashlib.sha256((S/'journal.json').read_bytes()).hexdigest(),'original_app_unchanged':True,'original_routing_restored':True,'own_rule_absent':True,'previous_new_id':None,'at':time.time(),'reason':'idle upstream sockets were incorrectly treated as live requests'}
for name in ['release_worker.py.log','guardian.py.log','release-error.txt','initial-inspect.json']:
 if (S/name).exists():shutil.copyfile(S/name,archive/name)
(S/'journal.json').rename(archive/'journal.json');(S/'retry-authorized.json').write_text(json.dumps(record,indent=2))
for name in ['guardian.py','release_worker.py']:
 out=open(S/(name+'.log'),'a');p=subprocess.Popen(['python3',str(S/name)],stdin=subprocess.DEVNULL,stdout=out,stderr=subprocess.STDOUT,start_new_session=True,cwd=S);(S/(name+'.pid')).write_text(str(p.pid));print(json.dumps({'retry_launched':name,'pid':p.pid}))
