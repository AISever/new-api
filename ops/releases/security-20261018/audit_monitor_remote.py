"""Independent read-only ledger/error audit; interrupt worker to trigger rollback."""
import json,os,re,signal,subprocess,time
from pathlib import Path
S=Path.home()/'.local/state/new-api-security-release-20261018';samples=0
while True:
 j=json.loads((S/'journal.json').read_text())
 if j['phase'] in ['verified','rolled_back','rollback_failed','aborted_before_cutover']:break
 if j['phase']!='observing':time.sleep(3);continue
 errors=[]
 try:
  p=subprocess.run(['docker','logs','--since',str(int(j['published_at'])),'--tail','1500',j['new_id']],capture_output=True,text=True,timeout=10)
  assert p.returncode==0
  text=p.stdout+'\n'+p.stderr
  if re.search(r'(?m)^(panic:|fatal error:)|failed to batch update (user|token) quota|\[Recovery\].*panic recovered',text):errors.append('critical application or quota persistence error')
  q="SELECT count(*) FROM logs WHERE created_at>="+str(int(j['published_at']))+" AND type=2 AND quota<0"
  p=subprocess.run(['docker','exec','new-api-postgres','psql','-XqAt','-U','newapi','-d','new-api','-v','ON_ERROR_STOP=1','-c',q],capture_output=True,text=True,timeout=10)
  assert p.returncode==0
  if int(p.stdout.strip()):errors.append('negative consumption record after publication')
  samples+=1;(S/'ledger-audit.json').write_text(json.dumps({'audit_samples':samples,'critical_errors':errors,'checked_at':time.time()}))
 except Exception as e:
  (S/'ledger-audit-read-error.txt').write_text(repr(e))
 if errors:
  (S/'ledger-audit-trigger.json').write_text(json.dumps({'errors':errors,'at':time.time()}))
  pid=j['worker_pid'];assert b'release_worker.py' in Path('/proc/'+str(pid)+'/cmdline').read_bytes();os.kill(pid,signal.SIGTERM);break
 time.sleep(30)
