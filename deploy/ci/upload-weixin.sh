#!/usr/bin/env bash
# Uploads the mini program as a new version. It then has to be submitted for review in the
# WeChat console: WeChat offers no way to do that from CI for a mini program's own developer.
#   WEIXIN_APPID=... WEIXIN_UPLOAD_KEY=... [WEIXIN_API_URL=...] upload-weixin.sh <version>
set -euo pipefail
version=${1:?usage: upload-weixin.sh <version>}
: "${WEIXIN_APPID:?set WEIXIN_APPID}"
: "${WEIXIN_UPLOAD_KEY:?set WEIXIN_UPLOAD_KEY}"
cd "$(dirname "$0")/../.."

key=$(mktemp)
trap 'rm -f "$key"' EXIT
printf '%s\n' "$WEIXIN_UPLOAD_KEY" > "$key"

# The repo has the tourist AppID, for trying it in DevTools, and a domain that isn't ICP-filed.
node -e '
  const fs = require("fs");
  const p = JSON.parse(fs.readFileSync("weixin/project.config.json", "utf8"));
  p.appid = process.env.WEIXIN_APPID;
  fs.writeFileSync("weixin/project.config.json", JSON.stringify(p, null, 2));
  if (process.env.WEIXIN_API_URL) {
    fs.writeFileSync("weixin/config.js", "module.exports = { API_URL: " + JSON.stringify(process.env.WEIXIN_API_URL) + " };\n");
  }
'

npx --yes miniprogram-ci@2.1.31 upload \
  --project-path weixin \
  --private-key-path "$key" \
  --appid "$WEIXIN_APPID" \
  --upload-version "$version" \
  --upload-description "$(git log -1 --format=%h)" \
  --robot 1 \
  --enable-es6 true \
  --enable-minify true
