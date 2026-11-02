import copy,unittest,tempfile,sys,json,hashlib,tarfile,io,fcntl,builtins
from pathlib import Path
from unittest.mock import patch
import controls
import operation

class Controls(unittest.TestCase):
 def setUp(self):
  self.a={'entries':[{'redemption_id':10,'inviter_id':1,'inviter_username':'owner','invitee_id':2,'invitee_username':'friend','reward_quota':5,'fact_sha256':'fact','already_backfilled':False}],'total_quota':5,'expected_beneficiaries':{'1':{'username':'owner','quota':5,'codes':1}}}
  self.plan={'entries':copy.deepcopy(self.a['entries']),'pending_reward_quota':'5'}
  self.before={'fingerprints':{'users':'old','logs':'old','options':'same','user_subscriptions':'same'},'users_normalized':'same','old_logs':'same','selected_users':{'1':{'id':1,'quota':9876,'aff_quota':7,'aff_history':11}},'fhy131':{'id':131,'amount_used':42,'next_reset_time':86400}}
  self.after=copy.deepcopy(self.before);self.after['fingerprints']['redemption_reward_backfills']='new';self.after['selected_users']['1'].update(aff_quota=12,aff_history=16)
  self.ledger=[{'redemption_id':10,'inviter_id':1,'invitee_id':2,'reward_quota':5,'fact_hash':'fact','plan_hash':'plan','log_id':50}]
  self.logs=[{'id':50,'user_id':1,'type':4,'request_id':'redemption-backfill:10','content':'历史兑换返奖补发 $0.01，兑换码ID 10，受邀用户 friend，奖励额度 5，原兑换时间 x'}]
 def test_exact_scope(self):controls.scope(self.plan,self.a)
 def test_scope_rejects_changed_amount_or_extra_recipient(self):
  for field,val in [('reward_quota',6),('inviter_id',3),('fact_sha256','changed')]:
   p=copy.deepcopy(self.plan);p['entries'][0][field]=val
   with self.assertRaises(AssertionError):controls.scope(p,self.a)
 def test_scope_rejects_completed_or_duplicate(self):
  for p in [dict(self.plan,entries=self.plan['entries']*2),dict(self.plan,entries=[dict(self.plan['entries'][0],already_backfilled=True)])]:
   with self.assertRaises(AssertionError):controls.scope(p,self.a)
 def test_success(self):self.assertEqual(controls.audit(self.before,self.after,self.a,self.ledger,self.logs,'plan')['credited_codes'],1)
 def test_preserve_everything_else(self):
  for part in ['fhy131','old_logs','users_normalized']:
   after=copy.deepcopy(self.after);after[part]='changed'
   with self.assertRaises(AssertionError):controls.audit(self.before,after,self.a,self.ledger,self.logs,'plan')
  after=copy.deepcopy(self.after);after['fingerprints']['options']='changed'
  with self.assertRaises(AssertionError):controls.audit(self.before,after,self.a,self.ledger,self.logs,'plan')
 def test_wrong_delta_log_owner_and_duplicate_ledger_rejected(self):
  after=copy.deepcopy(self.after);after['selected_users']['1']['quota']=9877
  with self.assertRaises(AssertionError):controls.audit(self.before,after,self.a,self.ledger,self.logs,'plan')
  logs=copy.deepcopy(self.logs);logs[0]['user_id']=99
  with self.assertRaises(AssertionError):controls.audit(self.before,self.after,self.a,self.ledger,logs,'plan')
  with self.assertRaises(AssertionError):controls.audit(self.before,self.after,self.a,self.ledger*2,self.logs,'plan')
 def test_partial_recovery_is_audited_not_double_paid(self):
  r=controls.audit(self.before,self.before,self.a,[],[],'plan',partial=True);self.assertEqual(r['credited_codes'],0)
  with self.assertRaises(AssertionError):controls.audit(self.before,self.before,self.a,[],[],'plan')
 def test_credit_requires_all_recovery_proofs(self):
  p=dict(database_restored=True,all_tables_match=True,files_restored=True,image_verified=True,redis_restored=True,app_stopped=True)
  controls.ready(p)
  for key in p:
   x=dict(p);x[key]=False
   with self.assertRaises(AssertionError):controls.ready(x)
 def test_recovery_command_runs_without_writable_log_storage(self):
  operation.S=Path('/not-writable');operation.J={}
  with patch.object(Path,'open',side_effect=PermissionError('readonly')):
   self.assertEqual(operation.run([sys.executable,'-c',"print('service-restored')"]),'service-restored')
 def test_error_journal_failure_does_not_skip_recovery(self):
  state={'service_up':False}
  operation.J={}
  class W:
   def safe_save(self):pass
  operation.w=W()
  with patch.object(operation,'execute',side_effect=OSError('full disk')),patch.object(operation,'save',side_effect=OSError('readonly')),patch.object(operation,'recover',side_effect=lambda:state.update(service_up=True)):
   with self.assertRaises(OSError):operation.guarded_execute()
  self.assertTrue(state['service_up'])
 def test_corrupt_image_layer_cannot_pass_recovery_proof(self):
  layer=b'original-layer-payload';diff='sha256:'+hashlib.sha256(layer).hexdigest();config=json.dumps({'rootfs':{'diff_ids':[diff]},'config':{'Env':['A=B']}}).encode();cid=hashlib.sha256(config).hexdigest()
  image={'Id':'sha256:'+cid,'RootFS':{'Layers':[diff]},'Config':{'Env':['A=B']}}
  for corrupt in [False,True]:
   with tempfile.TemporaryDirectory() as d:
    path=Path(d)/'image.tar'
    with tarfile.open(path,'w') as t:
     for name,data in [('manifest.json',json.dumps([{'Config':cid+'.json','Layers':['layer.tar']}]).encode()),(cid+'.json',config),('layer.tar',b'corrupt' if corrupt else layer)]:
      info=tarfile.TarInfo(name);info.size=len(data);t.addfile(info,io.BytesIO(data))
    if corrupt:
     with self.assertRaises(AssertionError):controls.verify_image(path,image)
    else:controls.verify_image(path,image)
 def test_guardian_can_acquire_existing_lock_on_readonly_storage(self):
  with tempfile.TemporaryDirectory() as d:
   operation.S=Path(d);(operation.S/'operation.lock').write_bytes(b'');real_open=builtins.open
   def readonly(path,mode='r',*a,**kw):
    if any(x in mode for x in 'wa+'):raise PermissionError('read-only filesystem')
    return real_open(path,mode,*a,**kw)
   with patch('builtins.open',side_effect=readonly):
    lock=operation.acquire_lock(True)
    with real_open(operation.S/'operation.lock','rb') as second:
     with self.assertRaises(BlockingIOError):fcntl.flock(second,fcntl.LOCK_EX|fcntl.LOCK_NB)
    lock.close()
 def test_loaded_id_is_taken_from_actual_restore_not_export_config(self):
  actual='sha256:'+'a'*64
  self.assertEqual(controls.loaded_image_id('Loaded image ID: '+actual+'\n'),actual)
  for text in ['', 'Loaded image: wrong',('Loaded image ID: '+actual+'\n')*2]:
   with self.assertRaises(AssertionError):controls.loaded_image_id(text)
 def test_log_digest_preserves_empty_delta_and_detects_either_change(self):
  self.assertEqual(controls.log_fingerprint('10|old','0|empty'),['10','old'])
  digest=controls.log_fingerprint('10|old','2|new');self.assertEqual(digest[0],'12')
  self.assertNotEqual(digest,controls.log_fingerprint('10|tampered','2|new'))
  self.assertNotEqual(digest,controls.log_fingerprint('10|old','2|tampered'))

if __name__=='__main__':unittest.main()
