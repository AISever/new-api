"""Package the already tested binary; no source compilation on production host."""
import pathlib,subprocess,json,hashlib,os,shutil
os.umask(0o077)
S=pathlib.Path.home()/'.local/state/new-api-invitation-production-20261102-r2'
C={'source':'b77dd30084f2f0fe842647b2d77b86e6542bfb1f','version':'codex/invite+b77dd3008+production','binary_sha256':'b8fb9f0b495c176ac78e948219ca37949dc22172ebec0935ba4fac265d15489c','base':'sha256:d07e5682b016202338524f832d8206f2d99ac4b916a1391db92100cdfb9405e1','tag':'new-api:invitation-b77dd3008-production'}
assert json.loads((S/'owner.json').read_text())['candidate_source']==C['source']
preview=S.parent/'new-api-invitation-preview/20261101';assert json.loads((preview/'owner.json').read_text())['task']=='new-api-invitation-20261101'
source=preview/'refresh-b77dd3008/new-api';assert hashlib.sha256(source.read_bytes()).hexdigest()==C['binary_sha256']
context=S/'candidate';context.mkdir();shutil.copyfile(source,context/'new-api');(context/'new-api').chmod(0o755)
(context/'Dockerfile').write_text('FROM '+C['base']+'\nCOPY new-api /new-api\nLABEL org.opencontainers.image.revision="'+C['source']+'"\nLABEL org.opencontainers.image.version="'+C['version']+'"\nLABEL codex.release="invitation-20261102-r2"\n')
(context/'.dockerignore').write_text('*\n!new-api\n!Dockerfile\n')
def run(*a):return subprocess.check_output(a,text=True).strip()
run('docker','build','--network=none','-q','-t',C['tag'],str(context))
image=json.loads(run('docker','image','inspect',C['tag']))[0];C['image_id']=image['Id']
assert image['Config']['Labels']['org.opencontainers.image.revision']==C['source']
assert 'codex.task' not in image['Config']['Labels']
assert run('docker','run','--rm','--network','none','--read-only','--entrypoint','sha256sum',C['image_id'],'/new-api').split()[0]==C['binary_sha256']
assert run('docker','run','--rm','--network','none','--read-only','-e','VERSION='+C['version'],C['image_id'],'--version')==C['version']
(S/'candidate.json').write_text(json.dumps(C,indent=2));print(json.dumps(C,indent=2))
