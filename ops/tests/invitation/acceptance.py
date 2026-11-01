# Isolated synthetic fixtures only. Never point these scripts at production.
import http.cookiejar,json,pathlib,subprocess,time,urllib.request,urllib.error
root=pathlib.Path(__file__).resolve().parents[3]/'.env/artifacts/invitation-test-20261101';root.mkdir(parents=True,exist_ok=True);prefix='new-api-invitation-20261101';base='http://127.0.0.1:13019';password='InviteTest2026!'
c=json.loads(subprocess.check_output(['docker','inspect',prefix+'-postgres']))[0];assert c['Config']['Labels'].get('codex.task')==prefix
class Client:
 def __init__(self):self.uid=None;self.opener=urllib.request.build_opener(urllib.request.ProxyHandler({}),urllib.request.HTTPCookieProcessor(http.cookiejar.CookieJar()))
 def req(self,path,body=None,method=None):
  headers={'Content-Type':'application/json'}
  if self.uid:headers['New-Api-User']=str(self.uid)
  req=urllib.request.Request(base+path,data=json.dumps(body).encode() if body is not None else None,method=method,headers=headers)
  try:r=self.opener.open(req,timeout=20)
  except urllib.error.HTTPError as e:r=e
  return r.code,json.load(r)
 def ok(self,path,body=None,method=None):
  code,value=self.req(path,body,method);assert code==200 and value.get('success'),(path,code,value);return value.get('data')
 def login(self,name):self.uid=self.ok('/api/user/login',{'username':name,'password':password})['id'];return self
admin=Client();assert admin.ok('/api/status')['version']=='codex/invitation-20261101+test', 'unexpected server version; no fixture writes allowed';status=admin.ok('/api/setup')
if not status['status']:admin.ok('/api/setup',{'username':'inviteadmin','password':password,'confirmPassword':password})
admin.login('inviteadmin');assert admin.uid==1
now=int(time.time());next_reset=((now+8*3600)//86400+1)*86400-8*3600
sql="""UPDATE users SET aff_code='fixture-admin' WHERE id=1;
INSERT INTO users (id,username,password,display_name,role,status,quota,aff_code,aff_count,inviter_id,\"group\")
 SELECT 2,'invite-owner',password,'Invite owner',1,1,9876,'fixture-owner',22,0,'default' FROM users WHERE id=1
 ON CONFLICT (id) DO NOTHING;
INSERT INTO users (id,username,password,display_name,role,status,quota,aff_code,aff_count,inviter_id,\"group\")
 SELECT n,'invitee-'||lpad((n-2)::text,2,'0'),password,'Synthetic invitee',1,1,9876,'fixture-code-'||n::text,0,2,'default' FROM users CROSS JOIN generate_series(3,24) n WHERE id=1 ON CONFLICT (id) DO NOTHING;
INSERT INTO users (id,username,password,display_name,role,status,quota,aff_code,aff_count,inviter_id,\"group\")
 SELECT 30,'other-inviter',password,'Other inviter',1,1,9876,'fixture-other',1,0,'default' FROM users WHERE id=1 ON CONFLICT (id) DO NOTHING;
INSERT INTO users (id,username,password,display_name,role,status,quota,aff_code,aff_count,inviter_id,\"group\")
 SELECT 31,'foreign-user',password,'Foreign user',1,1,9876,'fixture-foreign',0,30,'default' FROM users WHERE id=1 ON CONFLICT (id) DO NOTHING;
INSERT INTO top_ups (id,user_id,money,trade_no,payment_provider,status) VALUES
 (1,24,99,'fixture-paid','stripe','success'),(2,24,3.2,'fixture-sub','','success'),(3,24,9.99,'fixture-old','','success'),(4,24,10,'fixture-cny','epay','success') ON CONFLICT (id) DO NOTHING;
INSERT INTO subscription_plans (id,title,price_amount,currency,total_amount,quota_reset_period,quota_reset_custom_seconds) VALUES (1,'Synthetic plan',3.2,'USD',50000000,'custom',86400) ON CONFLICT (id) DO NOTHING;
INSERT INTO subscription_orders (id,user_id,plan_id,money,trade_no,payment_provider,status) VALUES (1,24,1,3.2,'fixture-sub','stripe','success') ON CONFLICT (id) DO NOTHING;
INSERT INTO user_subscriptions (id,user_id,plan_id,amount_total,amount_used,status,source,start_time,end_time,last_reset_time,next_reset_time) VALUES (1,24,1,50000000,314,'active','order',%(now)d,%(end)d,%(last)d,%(next)d) ON CONFLICT (id) DO NOTHING;
INSERT INTO redemptions (id,key,status,quota,redeem_type,used_user_id,redeemed_time) VALUES (1,'fixture-redeemed',3,500000,'quota',24,%(now)d) ON CONFLICT (id) DO NOTHING;
INSERT INTO invitation_amount_snapshots (source_kind,source_id,user_id,amount,currency,basis,quota,quota_per_unit,exchange_rate,created_at) VALUES
 ('topup',1,24,'2.10','USD','paid',0,'','',%(now)d),('subscription',1,24,'3.20','USD','paid',0,'','',%(now)d),('redemption',1,24,'1','USD','redemption',500000,'500000','1',%(now)d),('topup',4,24,'10','CNY','paid',0,'','',%(now)d) ON CONFLICT (source_kind,source_id) DO NOTHING;
SELECT setval(pg_get_serial_sequence('users','id'), (SELECT max(id) FROM users));
"""%{'now':now,'end':now+30*86400,'last':next_reset-86400,'next':next_reset}
p=subprocess.run(['docker','exec','-i',prefix+'-postgres','psql','-XqAt','-U','invitation','-d','invitation_ui','-v','ON_ERROR_STOP=1'],input=sql,text=True,capture_output=True);assert p.returncode==0,p.stderr
for key,value in [('QuotaPerUnit','500000'),('general_setting.quota_display_type','USD'),('InvitationAmountVisibility','{"mode":"off","user_ids":[]}')]:admin.ok('/api/option/',{'key':key,'value':value},'PUT')
(root/'ui-credentials.json').write_text(json.dumps({'admin':'inviteadmin','owner':'invite-owner','other':'other-inviter','password':password,'ids':{'admin':1,'owner':2,'other':30}},indent=2));(root/'ui-credentials.json').chmod(0o600)
owner=Client().login('invite-owner');other=Client().login('other-inviter');checks=[]
def policy(mode,ids=[]):admin.ok('/api/option/',{'key':'InvitationAmountVisibility','value':json.dumps({'mode':mode,'user_ids':ids})},'PUT')
def fingerprint():
 statement="SELECT id,quota,aff_quota,aff_history,aff_count,inviter_id FROM users ORDER BY id; SELECT id,amount_total,amount_used,last_reset_time,next_reset_time FROM user_subscriptions ORDER BY id;"
 return subprocess.check_output(['docker','exec',prefix+'-postgres','psql','-XqAt','-U','invitation','-d','invitation_ui','-c',statement],text=True)
before=fingerprint()
page=owner.ok('/api/user/self/invitees');assert page['total']==22 and len(page['items'])==20 and all(set(u)=={'username'} for u in page['items']);checks.append('default username-only response and pagination')
policy('selected',[2]);page=owner.ok('/api/user/self/invitees');cash=next(u for u in page['items'] if u['username']=='invitee-22')['amounts'];curr={r['currency']:r for r in cash['currencies']};assert curr['USD']['total']=='6.3' and curr['CNY']['total']=='10' and cash['unknown_count']==1;checks.append('three-source exact totals, subscription dedup and separate currencies')
foreign=other.ok('/api/user/self/invitees?inviter_id=2');assert [u['username'] for u in foreign['items']]==['foreign-user'] and 'amounts' not in foreign['items'][0];checks.append('selected grant, unselected denial and inviter parameter ignored')
code,value=owner.req('/api/option/',{'key':'InvitationAmountVisibility','value':'{"mode":"all","user_ids":[]}'},'PUT');assert value.get('success') is False;checks.append('ordinary user cannot self-authorize (legacy auth uses HTTP 200 + success=false)')
policy('off');page=owner.ok('/api/user/self/invitees');assert not page['can_view_amounts'] and all('amounts' not in u for u in page['items']);checks.append('revocation effective on next request')
policy('all');assert other.ok('/api/user/self/invitees')['can_view_amounts'];checks.append('all inviters mode applies without changing ownership')
policy('selected',[]);assert not owner.ok('/api/user/self/invitees')['can_view_amounts'];checks.append('empty selected list grants nobody')
code,bad=admin.req('/api/option/',{'key':'InvitationAmountVisibility','value':'{"mode":"wrong","user_ids":[]}'},'PUT');assert not bad.get('success');checks.append('malformed policy rejected')
policy('off');assert fingerprint()==before;checks.append('wallets, rewards, invite links and subscription quota/reset anchors preserved')
(root/'http-acceptance.json').write_text(json.dumps({'passed':True,'checks':checks,'version':'codex/invitation-20261101+test','production_touched':False},indent=2))
print('Isolated actual HTTP acceptance: '+str(len(checks))+' checks passed')
