#!/usr/bin/env bash
# Starts everything on the macOS runner: Postgres, MongoDB, the fake model and map, the real
# server, and Leo. All of it is throwaway and stays on this machine. Waits are bounded.
set -euo pipefail
LOGS=$1
HERE=$(cd "$(dirname "$0")" && pwd)
ROOT=$(cd "$HERE/../.." && pwd)
mkdir -p "$LOGS"

wait_for() { # name, seconds, command
  for _ in $(seq 1 "$2"); do
    if eval "$3" >/dev/null 2>&1; then echo "$1 is up"; return 0; fi
    sleep 1
  done
  echo "$1 did not come up"; return 1
}

export HOMEBREW_NO_AUTO_UPDATE=1 HOMEBREW_NO_INSTALL_CLEANUP=1
brew install postgresql@17
brew tap mongodb/brew
brew install mongodb-community
PGBIN="$(brew --prefix postgresql@17)/bin"
echo "$PGBIN" >> "${GITHUB_PATH:-/dev/null}"
export PATH="$PGBIN:$PATH"

PGDATA=$(mktemp -d)
initdb -D "$PGDATA" -U "$USER" --auth=trust >/dev/null
pg_ctl -D "$PGDATA" -l "$LOGS/postgres.log" -o "-p 5432 -c listen_addresses=127.0.0.1" start
wait_for postgres 30 "pg_isready -h 127.0.0.1"
createdb -h 127.0.0.1 -U "$USER" syn

MONGO=$(mktemp -d)
nohup mongod --dbpath "$MONGO" --port 27017 --bind_ip 127.0.0.1 > "$LOGS/mongod.log" 2>&1 &
wait_for mongod 60 "nc -z 127.0.0.1 27017"

nohup python3 "$HERE/fake_services.py" 9000 > "$LOGS/fake.log" 2>&1 &
wait_for fake 15 "nc -z 127.0.0.1 9000"

(cd "$ROOT" && go build -o "$LOGS/../synagent-bin" ./cmd/synagent)
export SYN_DATABASE_URL="postgres://$USER@127.0.0.1:5432/syn?sslmode=disable"
export SYN_MONGO_URL="mongodb://127.0.0.1:27017" SYN_MONGO_DB=syn
export SYN_PHOTO_DIR="$(mktemp -d)"
export SYN_ADDR=:8080 SYN_PUBLIC_ORIGIN=http://localhost:8080
export SYN_LLM_BASE_URL=http://127.0.0.1:9000/v1 SYN_LLM_API_KEY=fake
export SYN_NOMINATIM_URL=http://127.0.0.1:9000 SYN_OVERPASS_URL=http://127.0.0.1:9000/overpass SYN_PHOTON_URL=http://127.0.0.1:9000/photon
export SYN_EXPO_PUSH_URL=http://127.0.0.1:9000/push
export SYN_DATE_MIN_LEAD=30s SYN_DEBRIEF_DELAY=20s
nohup "$LOGS/../synagent-bin" serve > "$LOGS/server.log" 2>&1 &
wait_for server 60 "curl -fsS http://127.0.0.1:8080/readyz"

nohup python3 "$HERE/peer.py" "$LOGS/server.log" "$LOGS" 70 > "$LOGS/peer.log" 2>&1 &
wait_for seeding 180 "test -f '$LOGS/seeded'" || { tail -30 "$LOGS/peer.log"; tail -30 "$LOGS/server.log"; exit 1; }
tail -8 "$LOGS/peer.log"
