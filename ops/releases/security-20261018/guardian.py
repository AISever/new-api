"""Independent watchdog: roll back if the detached release worker disappears."""
import json,os,signal,subprocess,time
from pathlib import Path
S=Path.home()/'.local/state/new-api-security-release-20261018'
while True:
 try:j=json.loads((S/'journal.json').read_text())
 except (FileNotFoundError,ValueError):time.sleep(2);continue
 if j['phase'] in ['verified','rolled_back','rollback_failed','aborted_before_cutover']:break
 pid=j['worker_pid'];alive=Path('/proc/'+str(pid)+'/cmdline').exists()
 if alive:
  try:alive=b'release_worker.py' in Path('/proc/'+str(pid)+'/cmdline').read_bytes()
  except OSError:alive=False
 stale=time.time()-j.get('heartbeat',j['started'])>240
 if alive and not stale:time.sleep(5);continue
 (S/'guardian-event.json').write_text(json.dumps({'worker_alive':alive,'heartbeat_stale':stale,'at':time.time()}))
 if alive:
  os.kill(pid,signal.SIGTERM)
  for _ in range(12):
   time.sleep(5)
   now=json.loads((S/'journal.json').read_text())
   if now['phase'] in ['rolled_back','verified','rollback_failed']:break
  else:
   if Path('/proc/'+str(pid)+'/cmdline').exists() and b'release_worker.py' in Path('/proc/'+str(pid)+'/cmdline').read_bytes():os.kill(pid,signal.SIGKILL)
 if json.loads((S/'journal.json').read_text())['phase'] not in ['rolled_back','verified','rollback_failed']:
  with open(S/'guardian-rollback.log','a') as out:
   subprocess.run(['python3',str(S/'release_worker.py'),'--rollback'],stdout=out,stderr=subprocess.STDOUT,timeout=240)
 break
