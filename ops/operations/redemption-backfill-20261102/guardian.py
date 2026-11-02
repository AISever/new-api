"""Recover the same application if the operation worker is lost; never issue credits."""
import json,os,signal,subprocess,time
from pathlib import Path
S=Path.home()/'.local/state/new-api-reward-backfill-20261102'
while True:
 try:j=json.loads((S/'journal.json').read_text())
 except (FileNotFoundError,ValueError):time.sleep(2);continue
 if j['phase'] in ['verified','recovered_service','recovery_requires_attention']:break
 pid=j['worker_pid']
 try:alive=str(S/'operation.py').encode() in Path('/proc/'+str(pid)+'/cmdline').read_bytes().split(b'\0')
 except OSError:alive=False
 if alive and time.time()-j.get('heartbeat',j['started'])<240:time.sleep(5);continue
 if alive:
  try:os.kill(pid,signal.SIGTERM)
  except OSError:pass
  for _ in range(12):
   time.sleep(5)
   try:
    now=json.loads((S/'journal.json').read_text())
    if now['phase'] in ['verified','recovered_service','recovery_requires_attention']:break
   except (OSError,ValueError):pass
  else:
   try:
    if str(S/'operation.py').encode() in Path('/proc/'+str(pid)+'/cmdline').read_bytes().split(b'\0'):os.kill(pid,signal.SIGKILL)
   except OSError:pass
 try:now=json.loads((S/'journal.json').read_text())
 except (OSError,ValueError):now={}
 if now.get('phase') not in ['verified','recovered_service','recovery_requires_attention']:
  for attempt in range(3):
   try:out=open(S/'guardian-recovery.log','a')
   except OSError:out=open(os.devnull,'w')
   try:
    with out:result=subprocess.run(['python3',str(S/'operation.py'),'--recover'],stdout=out,stderr=subprocess.STDOUT,timeout=300)
    if result.returncode==0:break
   except (OSError,subprocess.TimeoutExpired):pass
   time.sleep(3)
  else:
   try:
    j=json.loads((S/'journal.json').read_text());j.update(phase='recovery_requires_attention',guardian_recovery_attempts=3,heartbeat=time.time());(S/'journal.json').write_text(json.dumps(j))
   except OSError:pass
 break
