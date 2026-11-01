# Isolated synthetic fixtures only. Never point these scripts at production.
import hashlib,http.cookiejar,json,os,pathlib,secrets,socket,subprocess,time,urllib.request,urllib.error
repo=pathlib.Path(__file__).resolve().parents[3];root=repo/'.env/artifacts/invitation-test-20261101';root.mkdir(parents=True,exist_ok=True);prefix='new-api-invitation-20261101';label='codex.task='+prefix
c=json.loads((root/'database-credentials.json').read_text());envfile=root/'ui.env'
run=lambda args,**kw:subprocess.run(args,check=True,capture_output=True,text=True,**kw).stdout.strip()
pg=json.loads(run(['docker','inspect',prefix+'-postgres']))[0];assert pg['Config']['Labels'].get('codex.task')==prefix
# Only create this task's separate UI database, leaving all other databases alone.
exists=run(['docker','exec',prefix+'-postgres','psql','-U','invitation','-d','postgres','-At','-c',"SELECT count(*) FROM pg_database WHERE datname='invitation_ui'"])
if exists=='0':run(['docker','exec',prefix+'-postgres','createdb','-U','invitation','invitation_ui'])
(root/'Dockerfile.ui').write_text('FROM new-api:security-ba844d202\nCOPY new-api /new-api\n')
run(['docker','build','-q','-t','new-api:invitation-20261101-test','-f',str(root/'Dockerfile.ui'),str(root)])
if not envfile.exists():
 envfile.write_text('SQL_DSN=postgresql://invitation:'+c['password']+'@'+prefix+'-postgres:5432/invitation_ui?sslmode=disable\nSESSION_SECRET='+secrets.token_hex(24)+'\nCRYPTO_SECRET='+secrets.token_hex(24)+'\nVERSION=codex/invitation-20261101+test\nNODE_TYPE=master\nTZ=Asia/Shanghai\nMEMORY_CACHE_ENABLED=false\nBATCH_UPDATE_ENABLED=false\nERROR_LOG_ENABLED=true\n')
 envfile.chmod(0o600)
name=prefix+'-app';old=subprocess.run(['docker','inspect',name],capture_output=True,text=True)
if old.returncode:
 run(['docker','run','-d','--name',name,'--label',label,'--network',prefix+'-network','--env-file',str(envfile),'--memory','768m','--cpus','1','--pids-limit','256','--restart=no','new-api:invitation-20261101-test','--log-dir','/tmp/invitation-logs'])
else:
 item=json.loads(old.stdout)[0];assert item['Config']['Labels'].get('codex.task')==prefix
 if not item['State']['Running']:run(['docker','start',name])
app=json.loads(run(['docker','inspect',name]))[0];ip=app['NetworkSettings']['Networks'][prefix+'-network']['IPAddress']
(root/'ui-address.json').write_text(json.dumps({'target':ip,'port':3000,'public_local':'http://127.0.0.1:13019','container':name}))
for i in range(60):
 try:
  with urllib.request.urlopen('http://'+ip+':3000/api/status',timeout=2) as r:d=json.load(r)
  if d.get('success'):break
 except Exception:pass
 time.sleep(1)
else:raise RuntimeError('UI fixture did not start; inspect owned app logs')
print('Owned internal fixture application ready; source version '+d['data']['version'])
