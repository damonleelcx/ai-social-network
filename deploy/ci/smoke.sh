#!/usr/bin/env bash
# Checks a deployed server answers on the pages people land on.   smoke.sh <origin>
set -euo pipefail
origin=$1
for path in /readyz / /app/ /legal/terms; do
  code=000
  for _ in $(seq 1 12); do
    code=$(curl -s -o /dev/null -w '%{http_code}' --max-time 15 "$origin$path" || true)
    [ "$code" = 200 ] && break
    sleep 10
  done
  echo "$path $code"
  [ "$code" = 200 ] || exit 1
done
# Signed out, an API route has to refuse rather than fall through to a page.
code=$(curl -s -o /dev/null -w '%{http_code}' --max-time 15 "$origin/v1/referral" || true)
echo "/v1/referral $code"
[ "$code" = 401 ]
