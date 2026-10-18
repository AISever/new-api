#!/usr/bin/env python3
"""Deploy/inspect/stop an isolated test stack without touching production services."""
import argparse
import json
from pathlib import Path
import re
import shlex
import shutil
import subprocess
import tempfile

p = argparse.ArgumentParser(description=__doc__)
p.add_argument("action", choices=["deploy", "status", "stop"])
p.add_argument("--host", required=True)
p.add_argument("--ssh-user", default="ubuntu")
p.add_argument("--ssh-key", required=True)
p.add_argument("--source-ip")
p.add_argument("--run-id", default="20261018")
p.add_argument("--port", type=int, default=13018)
p.add_argument("--archive")
p.add_argument("--image")
p.add_argument("--image-id")
p.add_argument("--fixture-image")
p.add_argument("--version")
a = p.parse_args()
if not re.fullmatch(r"[a-z0-9][a-z0-9-]{0,39}", a.run_id) or not 1024 <= a.port <= 65535:
    p.error("invalid isolated test name or port")
if a.action == "deploy" and not all([a.archive, a.image, a.image_id, a.fixture_image, a.version]):
    p.error("deploy requires archive, image, image-id, fixture-image and version")
if a.image and not a.image.startswith("new-api:security-"):
    p.error("candidate image must use the security namespace")
with tempfile.TemporaryDirectory(prefix="new-api-security-ssh-") as d:
    key = Path(d) / "key"; shutil.copyfile(a.ssh_key, key); key.chmod(0o600)
    ssh = ["ssh", "-F", "/dev/null", "-i", str(key), "-o", "BatchMode=yes", "-o", "IdentitiesOnly=yes", "-o", "StrictHostKeyChecking=accept-new", "-o", "UserKnownHostsFile=" + d + "/known_hosts", "-o", "ConnectTimeout=10", "-o", "ServerAliveInterval=15", "-o", "ServerAliveCountMax=3"]
    if a.source_ip: ssh += ["-b", a.source_ip]
    ssh += [a.ssh_user + "@" + a.host]
    if a.action == "deploy":
        destination = ".local/state/new-api-security-tests/" + a.run_id
        command = "umask 077; mkdir -p " + shlex.quote(destination) + "; cat > " + shlex.quote(destination + "/images.tar.gz")
        print(json.dumps({"phase": "uploading_local_image_archive", "bytes": Path(a.archive).stat().st_size}), flush=True)
        with open(a.archive, "rb") as source:
            subprocess.run(ssh + [command], stdin=source, check=True, timeout=600)
    config = {"action": a.action, "run_id": a.run_id, "port": a.port, "image": a.image, "image_id": a.image_id, "fixture_image": a.fixture_image, "version": a.version}
    source = "CONFIG = " + repr(config) + "\n" + Path(__file__).with_name("kkidc_security_test_remote.py").read_text()
    subprocess.run(ssh + ["python3 -"], input=source, text=True, check=True, timeout=360)
