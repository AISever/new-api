"""Remote half of the isolated security-test entrypoint; CONFIG is injected by its launcher."""
import json
import os
from pathlib import Path
import secrets
import socket
import subprocess
import tarfile
import time
import urllib.request

LABEL = "new-api-security-20261018"
prefix = "new-api-security-" + CONFIG["run_id"]
root = Path.home() / ".local/state/new-api-security-tests" / CONFIG["run_id"]
names = {kind: prefix + "-" + kind for kind in ["app", "postgres", "redis", "fixture"]}


def run(*args, timeout=60):
    p = subprocess.run(args, capture_output=True, text=True, timeout=timeout)
    if p.returncode:
        raise RuntimeError("command failed: " + " ".join(args[:3]) + ": " + p.stderr[-1200:])
    return p.stdout.strip()


def inspect(name):
    p = subprocess.run(["docker", "inspect", name], capture_output=True, text=True, timeout=10)
    return json.loads(p.stdout)[0] if p.returncode == 0 else None


def status(url):
    with urllib.request.urlopen(url, timeout=5) as r:
        data = json.load(r)
        return {"http_status": r.status, "success": data.get("success"), "version": (data.get("data") or {}).get("version")}


def production_snapshot():
    result = {}
    for name in ["new-api-local", "new-api-postgres", "new-api-redis", "snowlight-caddy", "new-api-test"]:
        c = inspect(name)
        if c:
            result[name] = {"id": c["Id"], "image": c["Image"], "started_at": c["State"]["StartedAt"], "restart_count": c["RestartCount"], "state": c["State"]["Status"]}
    return result


def assert_owned(name):
    c = inspect(name)
    if c and (c["Config"].get("Labels") or {}).get("codex.task") != LABEL:
        raise RuntimeError("refusing to touch a container not owned by this test: " + name)
    return c


before = production_snapshot()
health_before = status("http://127.0.0.1:3000/api/status")
if not health_before["success"]:
    raise RuntimeError("production preflight is not healthy")
for name in names.values():
    assert_owned(name)

if CONFIG["action"] == "deploy":
    mem = {line.split(":")[0]: int(line.split()[1]) for line in open("/proc/meminfo") if line.startswith(("MemAvailable:", "MemTotal:"))}
    if mem["MemAvailable"] < 1024 * 1024 or os.getloadavg()[0] > 4:
        raise RuntimeError("insufficient host resources for isolated tests")
    root.mkdir(parents=True, exist_ok=True)
    marker = root / "owner.json"
    expected = {"task": LABEL, "run_id": CONFIG["run_id"]}
    if marker.exists() and json.loads(marker.read_text()) != expected:
        raise RuntimeError("test directory ownership mismatch")
    marker.write_text(json.dumps(expected))
    archive = root / "images.tar.gz"
    with tarfile.open(archive, "r:gz") as t:
        manifest = json.load(t.extractfile("manifest.json"))
    tags = [tag for image in manifest for tag in image.get("RepoTags", [])]
    if not tags or any(not tag.startswith(("new-api:security-", "new-api-security-fixture:")) for tag in tags):
        raise RuntimeError("archive contains an image outside the security-test namespace")
    print(json.dumps({"phase": "loading_locally_built_images"}), flush=True)
    run("docker", "load", "-i", str(archive), timeout=180)
    image_id = json.loads(run("docker", "image", "inspect", CONFIG["image"]))[0]["Id"]
    if image_id != CONFIG["image_id"]:
        raise RuntimeError("loaded image ID does not match locally tested candidate")
    for name in names.values():
        if assert_owned(name):
            run("docker", "rm", "-f", name)
    network = subprocess.run(["docker", "network", "inspect", prefix], capture_output=True, text=True)
    if network.returncode:
        run("docker", "network", "create", "--label", "codex.task=" + LABEL, prefix)
    elif (json.loads(network.stdout)[0].get("Labels") or {}).get("codex.task") != LABEL:
        raise RuntimeError("network ownership mismatch")
    for name in ["data", "logs", "postgres", "redis"]:
        (root / name).mkdir(exist_ok=True)
    secret_file = root / "secrets.json"
    if secret_file.exists():
        secret = json.loads(secret_file.read_text())
    else:
        secret = {k: secrets.token_hex(24) for k in ["db", "session", "crypto"]}
        secret_file.write_text(json.dumps(secret)); secret_file.chmod(0o600)
    pg_env = root / "postgres.env"
    pg_env.write_text("POSTGRES_USER=security\nPOSTGRES_DB=security_test\nPOSTGRES_PASSWORD=" + secret["db"] + "\n"); pg_env.chmod(0o600)
    app_env = root / "app.env"
    app_env.write_text("\n".join([
        "SQL_DSN=postgresql://security:" + secret["db"] + "@" + names["postgres"] + ":5432/security_test",
        "REDIS_CONN_STRING=redis://" + names["redis"], "SESSION_SECRET=" + secret["session"], "CRYPTO_SECRET=" + secret["crypto"],
        "TZ=Asia/Shanghai", "VERSION=" + CONFIG["version"], "ERROR_LOG_ENABLED=true", "BATCH_UPDATE_ENABLED=false", "MEMORY_CACHE_ENABLED=true", "ANONYMOUS_REQUEST_BODY_LIMIT_KB=512", "SQL_MAX_OPEN_CONNS=10", "SQL_MAX_IDLE_CONNS=5",
    ]) + "\n"); app_env.chmod(0o600)
    def start(kind, memory, image, options, command=()):
        run("docker", "run", "-d", "--name", names[kind], "--label", "codex.task=" + LABEL,
            "--network", prefix, "--memory=" + memory, "--cpus=0.5", "--pids-limit=128",
            "--restart=no", "--log-opt", "max-size=10m", "--log-opt", "max-file=2", *options, image, *command)
    start("postgres", "384m", "postgres:16-alpine", ["--env-file", str(pg_env), "-v", str(root / "postgres") + ":/var/lib/postgresql/data"], ["postgres", "-c", "max_connections=20", "-c", "shared_buffers=32MB"])
    start("redis", "96m", "redis:7-alpine", ["-v", str(root / "redis") + ":/data"], ["redis-server", "--maxmemory", "64mb", "--maxmemory-policy", "noeviction"])
    start("fixture", "64m", CONFIG["fixture_image"], [])
    for _ in range(40):
        p = subprocess.run(["docker", "exec", names["postgres"], "pg_isready", "-U", "security", "-d", "security_test"], capture_output=True)
        if p.returncode == 0: break
        time.sleep(1)
    else: raise RuntimeError("test database did not become ready")
    with socket.socket() as s:
        s.bind(("127.0.0.1", CONFIG["port"]))
    start("app", "512m", CONFIG["image"], ["--env-file", str(app_env), "-p", "127.0.0.1:" + str(CONFIG["port"]) + ":3000", "-v", str(root / "data") + ":/data", "-v", str(root / "logs") + ":/app/logs"], ["--log-dir", "/app/logs"])
    for _ in range(60):
        try:
            ready = status("http://127.0.0.1:" + str(CONFIG["port"]) + "/api/status")
            if ready["success"] and ready["version"] == CONFIG["version"]: break
        except Exception: pass
        time.sleep(1)
    else: raise RuntimeError("isolated test application did not become healthy")
elif CONFIG["action"] == "stop":
    for name in names.values():
        c = assert_owned(name)
        if c and c["State"]["Running"]:
            run("docker", "stop", "-t", "10", name)

after = production_snapshot()
health_after = status("http://127.0.0.1:3000/api/status")
if before != after or not health_after["success"] or health_before["version"] != health_after["version"]:
    raise RuntimeError("production state changed during test operation; investigate before proceeding")
result = {"action": CONFIG["action"], "test_prefix": prefix, "test_port": CONFIG["port"], "production_unchanged": True, "production_before": before, "production_after": after, "production_health": health_after, "test_containers": {}}
for kind, name in names.items():
    c = assert_owned(name)
    if c:
        result["test_containers"][kind] = {"name": name, "id": c["Id"], "image_id": c["Image"], "state": c["State"]["Status"], "networks": list(c["NetworkSettings"]["Networks"]), "mounts": [{k: m.get(k) for k in ["Source", "Destination"]} for m in c["Mounts"]], "ports": c["HostConfig"]["PortBindings"]}
print(json.dumps(result, ensure_ascii=False, indent=2))
