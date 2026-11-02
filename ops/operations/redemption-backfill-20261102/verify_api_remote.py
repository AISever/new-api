"""Read-only live API checks; use existing credentials only, never mint access."""
import sys,json,time,os,urllib.request,urllib.parse,urllib.error
from pathlib import Path
SUFFIX=os.environ.get('BACKFILL_OPERATION_SUFFIX','');assert SUFFIX in ['', '-retry1']
S=Path.home()/('.local/state/new-api-reward-backfill-20261102'+SUFFIX);sys.path.insert(0,str(S));import operation as op
op.initialize();assert op.J['phase'] in ['observing','verified']
users=json.loads(op.sql("SELECT json_agg(row_to_json(t)) FROM (SELECT id,username,role,access_token FROM users WHERE id IN (1,21)) t"));byid={u['id']:u for u in users};root=byid[1];assert root['role']==100 and root['access_token']
def request(path,user=None):
 headers={}
 if user:headers={'Authorization':'Bearer '+user['access_token'],'New-Api-User':str(user['id'])}
 try:r=urllib.request.urlopen(urllib.request.Request('http://127.0.0.1:3000'+path,headers=headers),timeout=20)
 except urllib.error.HTTPError as err:r=err
 return r.status,json.load(r)
result={'production_payments_or_new_redemptions_performed':False,'new_credentials_created':False,'users':{}}
for uid,want in op.A['expected_beneficiaries'].items():
 uid=int(uid);u=byid[uid];own=bool(u.get('access_token'))
 if own:
  code,response=request('/api/log/self/?type=4&page_size=100&start_timestamp='+str(int(op.J['started'])-1),u)
 else:
  code,response=request('/api/log/?type=4&page_size=100&username='+urllib.parse.quote(u['username'])+'&start_timestamp='+str(int(op.J['started'])-1),root)
 assert code==200 and response['success'];logs=[l for l in response['data']['items'] if l.get('request_id','').startswith('redemption-backfill:')]
 assert len(logs)==want['codes'] and len({l['request_id'] for l in logs})==want['codes'] and all(l['user_id']==uid and l['type']==4 and l['content'].startswith('历史兑换返奖补发 ') for l in logs)
 expected={e['redemption_id']:e for e in op.A['entries'] if e['inviter_id']==uid}
 assert {int(l['request_id'].split(':')[1]) for l in logs}==set(expected)
 result['users'][str(uid)]={'username':u['username'],'log_count':len(logs),'log_ownership_and_content_verified':True,'own_authenticated_endpoint_verified':own,'admin_filtered_endpoint_verified':not own}
 if own:
  code,self_data=request('/api/user/self',u);assert code==200 and self_data['success'] and self_data['data']['id']==uid
code,denied=request('/api/log/self/?type=4');assert not denied.get('success');result['unauthenticated_log_access_denied']=True
# Root self-log route must not expose another invitee's rewards through a user_id hint.
code,own_only=request('/api/log/self/?type=4&user_id=21&page_size=100&start_timestamp='+str(int(op.J['started'])-1),root);assert code==200 and own_only['success'] and all(l['user_id']==1 for l in own_only['data']['items']);result['caller_supplied_user_id_does_not_override_ownership']=True
(S/'api-verification.json').write_text(json.dumps(result,indent=2));print(json.dumps(result,indent=2))
