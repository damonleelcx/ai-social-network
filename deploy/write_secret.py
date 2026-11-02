#!/usr/bin/env python3
"""Create or update the synagent/prod secret. Passwords are generated once and kept.
The model key comes from FORGE_LLM_API_KEY in the env file. Nothing secret is printed
or passed on a command line.

    python deploy/write_secret.py ../.env us-east-1
"""
import json
import os
import secrets
import subprocess
import sys
import tempfile

env_file, region = sys.argv[1], sys.argv[2]


def aws(*a):
    return subprocess.run(["aws", "secretsmanager", *a, "--region", region], capture_output=True, text=True)


key = ""
for line in open(env_file, encoding="utf8"):
    line = line.strip()
    if line.startswith("FORGE_LLM_API_KEY="):
        key = line.split("=", 1)[1].strip().strip("\"'")
if not key:
    sys.exit("FORGE_LLM_API_KEY not found in " + env_file)

cur = aws("get-secret-value", "--secret-id", "synagent/prod", "--query", "SecretString", "--output", "text")
if cur.returncode == 0:
    data = json.loads(cur.stdout)
    target = ["put-secret-value", "--secret-id", "synagent/prod"]
else:
    pg, mg = secrets.token_urlsafe(24), secrets.token_urlsafe(24)
    data = {
        "POSTGRES_PASSWORD": pg,
        "MONGO_PASSWORD": mg,
        "SYN_DATABASE_URL": f"postgres://synagent:{pg}@postgres.synagent.svc.cluster.local:5432/synagent?sslmode=disable",
        "SYN_MONGO_URL": f"mongodb://synagent:{mg}@mongodb.synagent.svc.cluster.local:27017/?authSource=admin",
    }
    target = ["create-secret", "--name", "synagent/prod"]
data["SYN_LLM_API_KEY"] = key
# Optional third argument: an env file holding the voice key as PLAY_TTS_API_KEY.
if len(sys.argv) > 3:
    for line in open(sys.argv[3], encoding="utf8"):
        if line.startswith("PLAY_TTS_API_KEY="):
            data["SYN_TTS_API_KEY"] = line.split("=", 1)[1].strip().strip("\"'")

# Moderators' addresses: this repository is public, so they aren't in a manifest.
if os.environ.get("ADMIN_EMAILS"):
    data["SYN_ADMIN_EMAILS"] = os.environ["ADMIN_EMAILS"]

fd, path = tempfile.mkstemp(suffix=".json")
try:
    with os.fdopen(fd, "w") as f:
        json.dump(data, f)
    r = aws(*target, "--secret-string", "file://" + path.replace("\\", "/"))
    if r.returncode != 0:
        sys.exit("secret write failed: " + r.stderr[:300].replace(key, "***"))
finally:
    os.remove(path)
print("secret ready:", target[0])
