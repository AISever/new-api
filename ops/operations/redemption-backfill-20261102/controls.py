"""Pure acceptance gates for the explicitly authorized one-off operation."""
import collections,copy,gzip,hashlib,json,tarfile,re

def scope(plan, approved):
    expected={e['redemption_id']:e for e in approved['entries']}
    actual=[e for e in plan['entries'] if not e.get('skip_reason')]
    assert len(expected)==len(approved['entries'])==len(actual),'approved code count changed'
    assert len({e['redemption_id'] for e in actual})==len(actual),'duplicate code'
    assert all(not e.get('already_backfilled') for e in actual),'prior completion requires reconciliation'
    assert {e['redemption_id']:e for e in actual}==expected,'approved financial facts changed'
    assert int(plan['pending_reward_quota'])==approved['total_quota']==sum(e['reward_quota'] for e in actual),'total changed'
    assert plan.get('before_exclusive')==approved.get('before'),'cutoff changed'
    totals=collections.defaultdict(int);counts=collections.Counter()
    for e in actual:totals[str(e['inviter_id'])]+=e['reward_quota'];counts[str(e['inviter_id'])]+=1
    assert set(totals)==set(approved['expected_beneficiaries']),'recipient set changed'
    for uid,want in approved['expected_beneficiaries'].items():assert totals[uid]==want['quota'] and counts[uid]==want['codes'],'recipient total changed'

def audit(before, after, approved, ledger, logs, plan_hash, partial=False):
    expected={e['redemption_id']:e for e in approved['entries']};seen=set();totals=collections.defaultdict(int);counts=collections.Counter()
    assert len({l['id'] for l in logs})==len(logs),'duplicate log IDs'
    by_log={l['id']:l for l in logs}
    assert len(ledger)==len(logs),'credit/log count mismatch'
    for row in ledger:
        rid=row['redemption_id'];assert rid in expected and rid not in seen,'unexpected/duplicate credit';seen.add(rid);e=expected[rid]
        assert all(row[k]==e[v] for k,v in [('inviter_id','inviter_id'),('invitee_id','invitee_id'),('reward_quota','reward_quota'),('fact_hash','fact_sha256')]),'ledger differs from approved facts'
        assert row['plan_hash']==plan_hash,'unexpected plan'
        if 'ratio' in e:assert row['ratio']==e['ratio'],'ratio changed'
        log=by_log.get(row['log_id']);assert log is not None and log['user_id']==e['inviter_id'] and log['type']==4,'log ownership/type mismatch'
        assert log['request_id']=='redemption-backfill:'+str(rid),'log/code link mismatch'
        assert log['content'].startswith('历史兑换返奖补发 ') and ('兑换码ID '+str(rid)+'，') in log['content'] and ('奖励额度 '+str(e['reward_quota'])+'，') in log['content'],'invalid visible reward log'
        totals[str(e['inviter_id'])]+=e['reward_quota'];counts[str(e['inviter_id'])]+=1
    assert {r['log_id'] for r in ledger}==set(by_log),'unlinked logs'
    if not partial:assert seen==set(expected),'not all approved codes completed'
    assert set(after['fingerprints'])-set(before['fingerprints'])<= {'redemption_reward_backfills'},'unexpected new table'
    for table,digest in before['fingerprints'].items():
        if table not in ['users','logs']:assert after['fingerprints'].get(table)==digest,'source table changed: '+table
    for key in ['users_normalized','old_logs','fhy131']:assert after[key]==before[key],'protected data changed: '+key
    for uid,old in before['selected_users'].items():
        want=copy.deepcopy(old);want['aff_quota']+=totals[uid];want['aff_history']+=totals[uid]
        assert after['selected_users'][uid]==want,'unexpected user delta'
    return {'credited_codes':len(seen),'visible_inviter_logs':len(logs),'total_quota':sum(totals.values()),'beneficiaries':{uid:{'quota':totals[uid],'codes':counts[uid]} for uid in approved['expected_beneficiaries']},'other_user_fields_configuration_subscriptions_and_old_logs_preserved':True,'fhy131_full_row_preserved':True}

def ready(proof):
    assert all(proof.get(k) is True for k in ['database_restored','all_tables_match','files_restored','image_verified','redis_restored','app_stopped']),'recovery proofs incomplete'

def verify_image(path,image):
    with tarfile.open(path) as archive:
        manifest=json.load(archive.extractfile('manifest.json'));assert len(manifest)==1
        member=manifest[0];data=archive.extractfile(member['Config']).read();digest=hashlib.sha256(data).hexdigest()
        assert member['Config'].rsplit('/',1)[-1].split('.')[0]==digest,'image config content digest mismatch'
        config=json.loads(data);assert config['rootfs']['diff_ids']==image['RootFS']['Layers'],'declared image filesystem changed'
        assert config['config']==image['Config'],'image runtime configuration differs'
        assert len(member['Layers'])==len(image['RootFS']['Layers'])
        for name,expected in zip(member['Layers'],image['RootFS']['Layers']):
            inp=archive.extractfile(name);magic=inp.read(2);inp.seek(0)
            if magic==b'\x1f\x8b':inp=gzip.GzipFile(fileobj=inp)
            h=hashlib.sha256()
            for block in iter(lambda:inp.read(1024*1024),b''):h.update(block)
            assert 'sha256:'+h.hexdigest()==expected,'image layer payload digest mismatch'
    return 'sha256:'+digest

def loaded_image_id(output):
    ids=re.findall(r'Loaded image ID: (sha256:[0-9a-f]{64})',output)
    assert len(ids)==1,'expected one untagged restored image'
    return ids[0]

def log_fingerprint(old,new):
    count,digest=old.split('|');n,ndigest=new.split('|')
    return [str(int(count)+int(n)),digest if int(n)==0 else hashlib.sha256((old+'|'+new).encode()).hexdigest()]
