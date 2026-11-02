import ast,copy,json,types,unittest
from pathlib import Path
code=ast.parse(Path(__file__).with_name('release_worker.py').read_text())

def function(name,env):
 node=next(n for n in code.body if isinstance(n,ast.FunctionDef) and n.name==name)
 exec(compile(ast.Module(body=[node],type_ignores=[]),'release-controls','exec'),env)
 return env[name]

class Controls(unittest.TestCase):
 def test_maintenance_changes_only_two_production_backends(self):
  before={'routes':[{'handler':'reverse_proxy','upstreams':[{'dial':'172.17.0.1:3000'}]},{'handler':'reverse_proxy','upstreams':[{'dial':'172.17.0.1:3000'}]},{'handler':'reverse_proxy','upstreams':[{'dial':'127.0.0.1:8080'}]}]}
  loaded=[];commands=[]
  e={'copy':copy,'CADDY':before,'caddy_get':lambda:before,'canonical':lambda x:json.dumps(x,sort_keys=True),'phase':lambda *a,**kw:None,'time':types.SimpleNamespace(time=lambda:1),'caddy_load':lambda x:loaded.append(x),'subprocess':types.SimpleNamespace(run=lambda *a,**k:types.SimpleNamespace(returncode=1)),'RULE':['scoped-rule'],'run':lambda *a:commands.append(a),'J':{},'save':lambda:None}
  function('gate',e)();self.assertEqual(loaded[0]['routes'][2],before['routes'][2]);self.assertEqual(before['routes'][0]['handler'],'reverse_proxy');
  for route in loaded[0]['routes'][:2]:
   self.assertEqual(route['handler'],'subroute');self.assertTrue(route['routes'][0]['terminal']);self.assertEqual(route['routes'][0]['handle'][0]['status_code'],503);self.assertEqual(route['routes'][1]['handle'][0],before['routes'][0])
  self.assertIn('DOCKER-USER',commands[0])
 def test_clone_changes_only_image_and_version(self):
  original={'Config':{'Env':['SQL_DSN=preserved','SESSION_SECRET=preserved','VERSION=old'],'Image':'old','Cmd':['--log-dir','/app/logs'],'Labels':{'custom':'keep','org.opencontainers.image.revision':'old'}},'HostConfig':{'NetworkMode':'new-api_default','RestartPolicy':{'Name':'always'},'Binds':['/opt/new-api/data:/data']}}
  bodies=[]
  class Connection:
   def __init__(self,*a,**k):pass
   def request(self,*a,**k):bodies.append(json.loads(k['body']))
   def getresponse(self):return types.SimpleNamespace(status=201,read=lambda:b'{"Id":"candidate-id"}')
   def close(self):pass
  e={'copy':copy,'ORIGINAL':original,'NEW':'verified-image','VERSION':'release-version','NAME':'new-api-local','DockerConnection':Connection,'json':json,'run':lambda *a:'1.52','inspect':lambda n:{'Config':{'Labels':{'org.opencontainers.image.revision':'new'}}}}
  self.assertEqual(function('create_container',e)(),'candidate-id');b=bodies[0];self.assertEqual(b['Labels'],{'custom':'keep','org.opencontainers.image.revision':'new'});self.assertEqual(b['Image'],'verified-image');self.assertEqual(b['Env'],['SQL_DSN=preserved','SESSION_SECRET=preserved','VERSION=release-version']);self.assertEqual(b['HostConfig'],original['HostConfig']);self.assertEqual(b['Cmd'],original['Config']['Cmd'])
 def test_unrelated_reverse_proxy_count_aborts(self):
  e={'copy':copy,'CADDY':{'handler':'reverse_proxy','upstreams':[{'dial':'172.17.0.1:3000'}]}}
  with self.assertRaises(AssertionError):function('gate',e)()
 def test_existing_journal_prevents_replay(self):
  import tempfile
  with tempfile.TemporaryDirectory() as d:
   state=Path(d);(state/'journal.json').write_text('{"phase":"verified"}')
   with self.assertRaises(AssertionError):function('ensure_new_run',{'S':state})()
   self.assertEqual((state/'journal.json').read_text(),'{"phase":"verified"}')
 def test_preflight_failure_does_not_mutate_production(self):
  import tempfile
  with tempfile.TemporaryDirectory() as d:
   calls=[];j={'phase':'preflight'}
   e={'S':Path(d),'J':j,'save':lambda:None,'phase':lambda value,**kw:calls.append(value),'rollback':lambda:calls.append('rollback'),'safe_note':lambda *a:None,'safe_save':lambda:None,'emergency_phase':lambda value,**kw:calls.append(value)}
   function('handle_release_failure',e)(AssertionError('wrong baseline'))
   self.assertEqual(calls,['aborted_before_cutover'])
 def test_failure_after_maintenance_rolls_back(self):
  import tempfile
  with tempfile.TemporaryDirectory() as d:
   calls=[]
   e={'S':Path(d),'J':{'phase':'draining'},'save':lambda:None,'phase':lambda value,**kw:calls.append(value),'rollback':lambda:calls.append('rollback'),'safe_note':lambda *a:None,'safe_save':lambda:None,'emergency_phase':lambda value,**kw:calls.append(value)}
   function('handle_release_failure',e)(RuntimeError('startup failed'))
   self.assertEqual(calls,['rollback'])
class EmergencyControls(unittest.TestCase):
 def test_state_storage_failure_cannot_prevent_rollback(self):
  class FailedState:
   def __truediv__(self,name):return self
   def write_text(self,*a,**k):raise OSError('state volume full')
  calls=[];e={'S':FailedState(),'J':{'phase':'starting_candidate','new_id':'new'},'save':lambda:(_ for _ in ()).throw(OSError('disk full')),'time':types.SimpleNamespace(time=lambda:1),'json':json,'print':lambda *a,**k:None,'rollback':lambda:calls.append('rollback'),'safe_note':lambda *a:None,'safe_save':lambda:None,'emergency_phase':lambda value,**kw:calls.append(value)}
  for name in ['safe_note','safe_save','emergency_phase']:
   if any(isinstance(n,ast.FunctionDef) and n.name==name for n in code.body):function(name,e)
  function('handle_release_failure',e)(OSError('disk full'))
  self.assertEqual(calls,['rollback'])
 def test_caddy_telemetry_failure_still_restores_old_application(self):
  calls=[]
  old={'Id':'old','Image':'old-image','Name':'/backup','State':{'Running':False}}
  candidate={'Id':'new','Image':'new-image','Name':'/new-api-local','State':{'Running':True}}
  def inspect(name):return old if name=='old' else candidate
  def run(*a,**k):
   calls.append(a)
   if a[:3]==('docker','rename','old'):old['Name']='/new-api-local'
   if a[:3]==('docker','start','old'):old['State']['Running']=True
  e={'B':{'app_id':'old'},'OLD':'old-image','NEW':'new-image','NAME':'new-api-local','J':{'new_id':'new','gate_active':True},'ORIGINAL':{'Config':{'Env':['VERSION=old-version']},'HostConfig':{'RestartPolicy':{'Name':'always'}}},'phase':lambda *a,**k:None,'emergency_phase':lambda *a,**k:None,'safe_note':lambda *a:None,'safe_save':lambda:None,'inspect':inspect,'run':run,'time':types.SimpleNamespace(time=lambda:0,sleep=lambda t:None),'caddy_get':lambda *a:(_ for _ in ()).throw(RuntimeError('Caddy admin down')),'ungate':lambda:None,'wait_health':lambda *a:None,'health':lambda *a:None,'guard_other_services':lambda:None}
  function('rollback',e)()
  self.assertIn(('docker','start','old'),calls)
 def test_valid_natural_midnight_advancement_is_accepted(self):
  import datetime
  row={'id':131,'username':'fhy','user_id':1267,'quota_reset_period':'custom','quota_reset_custom_seconds':86400,'last_reset_time':1793548800,'next_reset_time':1793635200,'amount_total':50000000}
  after={**row,'last_reset_time':row['next_reset_time'],'next_reset_time':row['next_reset_time']+86400}
  e={'datetime':datetime};function('check_fhy',e)
  function('check_fhy_transition',e)(row,after,row['next_reset_time']+1)
  with self.assertRaises(AssertionError):e['check_fhy_transition'](row,after,row['next_reset_time']-1)
class DrainControls(unittest.TestCase):
 def test_missing_or_invalid_telemetry_is_not_zero(self):
  for rows in [[],[{'address':'172.17.0.1:3000','num_requests':-1}],[{'address':'172.17.0.1:3000','num_requests':None}]]:
   with self.assertRaises(AssertionError):function('active_requests',{'caddy_get':lambda *a:rows})()
  self.assertEqual(function('active_requests',{'caddy_get':lambda *a:[{'address':'172.17.0.1:3000','num_requests':0}]})(),0)
 def test_settlement_covers_effective_batch_interval(self):
  e={'ORIGINAL':{'Config':{'Env':['BATCH_UPDATE_ENABLED=true','BATCH_UPDATE_INTERVAL=30']}}};self.assertEqual(function('settlement_seconds',e)(),63)
  e['ORIGINAL']['Config']['Env']=['BATCH_UPDATE_ENABLED=true'];self.assertEqual(e['settlement_seconds'](),13)
 def test_direct_existing_ingress_prevents_disconnect(self):
  import ipaddress
  e={'INITIAL':{'caddy':{'NetworkSettings':{'Networks':{}},'State':{'Pid':77}}},'json':json,'ipaddress':ipaddress,'run':lambda *a,**k:json.dumps([{'IPAM':{'Config':[{'Gateway':'172.18.0.1'}]}}]) if a[:3]==('docker','network','inspect') else ('0 0 172.17.0.2:4567 172.17.0.1:3000' if 'dport' in a else '0 0 172.18.0.6:3000 203.0.113.9:4567')}
  with self.assertRaises(AssertionError):function('require_known_ingress',e)(123)
  e['run']=lambda *a,**k:json.dumps([{'IPAM':{'Config':[{'Gateway':'172.18.0.1'}]}}]) if a[:3]==('docker','network','inspect') else ('0 0 172.17.0.2:4567 172.17.0.1:3000' if 'dport' in a else '0 0 172.18.0.6:3000 172.18.0.1:4567')
  e['require_known_ingress'](123)
  e['run']=lambda *a,**k:json.dumps([{'IPAM':{'Config':[{'Gateway':'172.18.0.1'}]}}]) if a[:3]==('docker','network','inspect') else ('0 0 172.17.0.2:4567 172.17.0.1:3000' if 'dport' in a else '0 0 172.18.0.6:3000 172.18.0.1:9999')
  with self.assertRaises(AssertionError):e['require_known_ingress'](123)
 def test_safe_abort_retry_refuses_any_completed_or_uncertain_cutover(self):
  original={'Id':'old','Image':'image','State':{'Running':True,'StartedAt':'start'},'RestartCount':0,'Config':{'Env':['VERSION=old']},'HostConfig':{'RestartPolicy':{'Name':'always'}}}
  journal={'phase':'rolled_back','rollback_verified':True,'error':'inflight requests did not drain safely'}
  f=function('can_retry_safe_abort',{})
  self.assertTrue(f(journal,original,original,True,True,False))
  for extra in [{'new_id':'new'},{'published_at':123},{'retry_number':1}]:self.assertFalse(f({**journal,**extra},original,original,True,True,False))
  self.assertFalse(f(journal,original,original,False,True,False));self.assertFalse(f(journal,original,original,True,False,False));self.assertFalse(f(journal,original,original,True,True,True))
if __name__=='__main__':unittest.main()
