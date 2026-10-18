"""A bounded production smoke test, using only a dedicated verification user."""
import http.cookiejar,json,os,secrets,subprocess,time,urllib.error,urllib.request
from pathlib import Path
os.umask(0o077)
S=Path.home()/'.local/state/new-api-security-release-20261018'
def sql(q):return subprocess.check_output(['docker','exec','new-api-postgres','psql','-XqAt','-U','newapi','-d','new-api','-v','ON_ERROR_STOP=1','-c',q],text=True,stderr=subprocess.PIPE).strip()
def request(path,body=None,method=None,uid=None,management=None,opener=None,key=None):
 headers={'Content-Type':'application/json'}
 if uid:headers['New-Api-User']=str(uid)
 if management:headers['Authorization']='Bearer '+management
 if key:headers['Authorization']='Bearer sk-'+key
 req=urllib.request.Request('http://127.0.0.1:3000'+path,data=json.dumps(body).encode() if body is not None else None,headers=headers,method=method)
 try:r=(opener.open if opener else urllib.request.urlopen)(req,timeout=45)
 except urllib.error.HTTPError as e:r=e
 text=r.read().decode()
 try:data=json.loads(text)
 except ValueError:data=text
 return r.code,data

def probe(phase):
 root=json.loads(sql("SELECT row_to_json(x) FROM (SELECT id,access_token FROM users WHERE role=100 AND access_token IS NOT NULL AND access_token<>'' ORDER BY id LIMIT 1) x"));assert root and root['access_token']
 def admin(path,body=None,method=None):
  code,data=request(path,body,method,root['id'],root['access_token'])
  if code!=200 or not data.get('success'):
   (S/'probe-management-error.json').write_text(json.dumps({'path':path,'http':code,'response':data}))
   raise RuntimeError('management API validation failed')
  return data.get('data')
 f=S/'probe-private.json'
 if not f.exists():
  cred={'username':'sec261018','password':secrets.token_hex(8)}
  assert sql("SELECT count(*) FROM users WHERE username='sec261018'")=='0','probe username already exists'
  admin('/api/user/',dict(cred,display_name='Release verification',role=1))
  uid=int(sql("SELECT id FROM users WHERE username='sec261018'"))
  admin('/api/user/',{'id':uid,'username':cred['username'],'display_name':'Release verification','group':'gemini-01','quota':20000,'status':1,'role':1},'PUT')
  cred['id']=uid;f.write_text(json.dumps(cred));f.chmod(0o600)
 cred=json.loads(f.read_text());uid=cred['id'];opener=urllib.request.build_opener(urllib.request.HTTPCookieProcessor(http.cookiejar.CookieJar()))
 code,data=request('/api/user/login',{'username':cred['username'],'password':cred['password']},opener=opener);assert code==200 and data.get('success'),'probe user login failed'
 code,data=request('/api/user/self',uid=uid,opener=opener);assert code==200 and data.get('success'),'probe self API failed'
 if 'token_id' in cred and sql('SELECT model_limits FROM tokens WHERE id='+str(cred['token_id']))!='gemini-3.1-flash-lite':
  code,data=request('/api/token/?status_only=true',{'id':cred['token_id'],'status':2},'PUT',uid=uid,opener=opener)
  assert code==200 and data.get('success'),'old probe token disable failed'
  del cred['token_id'];f.write_text(json.dumps(cred))
 if 'token_id' not in cred:
  body={'name':'release-ba844d202','expired_time':int(time.time())+3600,'remain_quota':10000,'unlimited_quota':False,'model_limits_enabled':True,'model_limits':'gemini-3.1-flash-lite','group':'gemini-01','cross_group_retry':False}
  code,data=request('/api/token/',body,uid=uid,opener=opener);assert code==200 and data.get('success'),'probe token creation failed'
  cred['token_id']=int(sql("SELECT id FROM tokens WHERE user_id="+str(uid)+" AND name='release-ba844d202' ORDER BY id DESC LIMIT 1"));f.write_text(json.dumps(cred))
 key=sql('SELECT key FROM tokens WHERE id='+str(cred['token_id'])+' AND user_id='+str(uid));assert key
 before=int(sql('SELECT quota FROM users WHERE id='+str(uid)))
 body={'model':'gemini-3.1-flash-lite','messages':[{'role':'user','content':'Reply OK.'}],'max_tokens':8,'temperature':0}
 started=time.time();code,data=request('/v1/chat/completions',body,key=key)
 if code!=200 or not isinstance(data,dict) or not data.get('choices'):
  (S/('probe-'+phase+'-upstream-error.json')).write_text(json.dumps({'http':code,'body':data}))
  raise RuntimeError('real upstream chat failed; response retained privately')
 chat_seconds=round(time.time()-started,3)
 code,stream=request('/v1/chat/completions',dict(body,stream=True),key=key);assert code==200 and isinstance(stream,str) and 'data:' in stream and '[DONE]' in stream,'real upstream SSE failed'
 time.sleep(12) # Existing batch interval is 5s: observe committed, not cached, debit.
 after=int(sql('SELECT quota FROM users WHERE id='+str(uid)));assert 0<=before-after<=10000,'probe exceeded its bounded token budget'
 assert root['access_token']==json.loads(sql('SELECT to_json(access_token) FROM users WHERE id='+str(root['id']))),'existing management token changed'
 if phase=='candidate':
  code,data=request('/api/user/',uid=root['id'],management=root['access_token']);assert code==200 and data.get('success') and root['access_token'] not in json.dumps(data) and 'access_token' not in json.dumps(data),'privileged token leaked in list API'
  code,data=request('/api/subscription/self',uid=uid,opener=opener);assert code==200 and data.get('success'),'subscription self API failed'
  for path,b in [('/v1/images/generations',{'model':'gemini-3.1-flash-lite','prompt':'test','n':18446744073686646784}),('/v1/billing/charge',{'model':'gemini-3.1-flash-lite','quantity':0})]:
   code,data=request(path,b,key=key);assert code==400,'invalid quantity did not fail with 400'
 result={'phase':phase,'probe_user_id':uid,'probe_username':cred['username'],'real_model':'gemini-3.1-flash-lite','group':'gemini-01','chat_success':True,'sse_success':True,'login_and_profile_success':True,'chat_seconds':chat_seconds,'probe_wallet_debit':before-after,'existing_user_quotas_not_used':True}
 (S/('probe-'+phase+'.json')).write_text(json.dumps(result,indent=2));return result
def disable_probe():
 cred=json.loads((S/'probe-private.json').read_text());uid=cred['id']
 assert sql('SELECT username FROM users WHERE id='+str(uid))==cred['username']=='sec261018'
 opener=urllib.request.build_opener(urllib.request.HTTPCookieProcessor(http.cookiejar.CookieJar()))
 code,data=request('/api/user/login',{'username':cred['username'],'password':cred['password']},opener=opener);assert code==200 and data.get('success')
 code,data=request('/api/token/?status_only=true',{'id':cred['token_id'],'status':2},'PUT',uid=uid,opener=opener);assert code==200 and data.get('success')
 root=json.loads(sql("SELECT row_to_json(x) FROM (SELECT id,access_token FROM users WHERE role=100 AND access_token IS NOT NULL AND access_token<>'' ORDER BY id LIMIT 1) x"))
 code,data=request('/api/user/manage',{'id':uid,'action':'disable'},uid=root['id'],management=root['access_token']);assert code==200 and data.get('success')
 assert sql('SELECT status FROM users WHERE id='+str(uid))=='2'
 (S/'probe-disabled.json').write_text(json.dumps({'probe_user_id':uid,'user_disabled':True,'token_disabled':True}))

if __name__=='__main__':
 import sys
 assert len(sys.argv)==2 and sys.argv[1] in ['baseline','candidate'],'explicit probe phase required'
 print(json.dumps(probe(sys.argv[1]),indent=2))
