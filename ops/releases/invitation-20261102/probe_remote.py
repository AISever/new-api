"""Read-only production acceptance. No new accounts, keys, payments or rewards."""
import json,subprocess,urllib.request,urllib.error

def sql(q):
 return subprocess.check_output(['docker','exec','-e','PGTZ=UTC','new-api-postgres','psql','-XqAt','-U','newapi','-d','new-api','-v','ON_ERROR_STOP=1','-c',q],text=True).strip()

def probe(phase):
 row=json.loads(sql("SELECT row_to_json(x) FROM (SELECT id,access_token FROM users WHERE role=100 AND status=1 AND deleted_at IS NULL AND access_token IS NOT NULL AND access_token<>'' ORDER BY id LIMIT 1) x"))
 assert row and row['id']>0
 headers={'Authorization':'Bearer '+row['access_token'],'New-Api-User':str(row['id'])}
 def request(path,auth=True):
  req=urllib.request.Request('http://127.0.0.1:3000'+path,headers=headers if auth else {})
  try:r=urllib.request.urlopen(req,timeout=15)
  except urllib.error.HTTPError as e:r=e
  return r.status,json.load(r)
 code,status=request('/api/status',False);assert code==200 and status['success']
 checks=['status/version']
 code,self_data=request('/api/user/self');assert code==200 and self_data['success'];checks.append('existing authenticated account remains readable')
 if phase=='candidate':
  code,page=request('/api/user/self/invitees');assert code==200 and page['success']
  assert page['data']['can_view_amounts'] is False
  assert all(set(item)=={'username'} for item in page['data']['items'])
  assert page['data']['total']==int(sql('SELECT count(*) FROM users WHERE inviter_id='+str(row['id'])+' AND deleted_at IS NULL'))
  checks+=['default username-only invitation list','no monetary fields exposed','direct-invite count matches database']
  code,denied=request('/api/user/self/invitees',False);assert not denied.get('success');checks.append('unauthenticated invitation access denied')
  code,options=request('/api/option/');assert code==200 and options['success']
  policy=json.loads(next(o['value'] for o in options['data'] if o['key']=='InvitationAmountVisibility'));assert policy=={'mode':'off','user_ids':[]};checks.append('root config reports off with no selected users')
  for path in ['/console/topup','/console/setting?tab=operation','/help']:
   with urllib.request.urlopen('http://127.0.0.1:3000'+path,timeout=15) as r:assert r.status==200 and b'<html' in r.read().lower()
  checks.append('wallet/admin/custom help pages served')
 return {'passed':True,'phase':phase,'checks':checks,'writes':False,'payments_or_redemptions_performed':False}
