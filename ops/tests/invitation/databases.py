# Isolated synthetic fixtures only. Never point these scripts at production.
import json,pathlib,secrets,subprocess,time,os
root=pathlib.Path(__file__).resolve().parents[3]/'.env/artifacts/invitation-test-20261101';root.mkdir(parents=True,exist_ok=True);prefix='new-api-invitation-20261101';label='codex.task='+prefix
run=lambda args,**kw:subprocess.run(args,check=True,capture_output=True,text=True,**kw).stdout.strip()
net=prefix+'-network'
p=subprocess.run(['docker','network','inspect',net],capture_output=True,text=True)
if p.returncode:run(['docker','network','create','--internal','--label',label,net])
else:
 network=json.loads(p.stdout)[0];assert network['Labels'].get('codex.task')==prefix and network['Internal']
creds=root/'database-credentials.json'
if creds.exists():c=json.loads(creds.read_text())
else:c={'password':secrets.token_hex(16)};creds.write_text(json.dumps(c));creds.chmod(0o600)
for kind,image,port,env in [('postgres','postgres:16-alpine','127.0.0.1:15419:5432',{'POSTGRES_USER':'invitation','POSTGRES_PASSWORD':c['password'],'POSTGRES_DB':'invitation_test'}),('mysql','mysql:8.0','127.0.0.1:15420:3306',{'MYSQL_ROOT_PASSWORD':c['password'],'MYSQL_DATABASE':'invitation_test'})]:
 name=prefix+'-'+kind;p=subprocess.run(['docker','inspect',name],capture_output=True,text=True)
 if p.returncode:
  args=['docker','run','-d','--name',name,'--network',net,'--label',label,'--memory','768m' if kind=='mysql' else '256m','--cpus','1','--pids-limit','256','--restart=no','-p',port]
  for k,v in env.items():args+=['-e',k+'='+v]
  run(args+[image])
 else:
  a=json.loads(p.stdout)[0];assert a['Config']['Labels'].get('codex.task')==prefix
  if not a['State']['Running']:run(['docker','start',name])
for i in range(90):
 pg=subprocess.run(['docker','exec',prefix+'-postgres','pg_isready','-U','invitation'],capture_output=True).returncode==0
 mysql=subprocess.run(['docker','exec','-e','MYSQL_PWD='+c['password'],prefix+'-mysql','mysqladmin','ping','-uroot','--silent'],capture_output=True).returncode==0
 if pg and mysql:break
 time.sleep(1)
else:raise RuntimeError('isolated DB startup timed out')
print('Owned loopback-only PostgreSQL/MySQL ready',flush=True)
envbuild=dict(os.environ,CGO_ENABLED='0')
run([os.environ.get('GO_BIN','go'),'test','-c','./model','-o',str(root/'model-db.test')],cwd=pathlib.Path(__file__).resolve().parents[3],env=envbuild)
for driver,dsn in [('postgres','postgresql://invitation:'+c['password']+'@127.0.0.1:5432/invitation_test?sslmode=disable'),('mysql','root:'+c['password']+'@tcp(127.0.0.1:3306)/invitation_test?charset=utf8mb4&parseTime=True&loc=Local')]:
 args=['docker','run','--rm','--label',label,'--network','container:'+prefix+'-'+driver,'--memory','512m','--cpus','1','--pids-limit','128','--read-only','-e','INVITATION_TEST_DRIVER='+driver,'-e','INVITATION_TEST_DSN='+dsn,'-v',str(root/'model-db.test')+':/invitation-model-test:ro','--entrypoint','/invitation-model-test','new-api:security-ba844d202','-test.run=InvitationAmountsExternalDatabase','-test.count=1','-test.v']
 with (root/('database-'+driver+'.log')).open('w') as out:
  r=subprocess.run(args,stdout=out,stderr=subprocess.STDOUT)
 assert r.returncode==0,driver+' test failed, see private workspace log'
 print(driver+' schema/migration/exact aggregation passed',flush=True)
