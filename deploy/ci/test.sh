#!/usr/bin/env bash
# Tests the release scripts against a throwaway repository.
set -euo pipefail
here=$(cd "$(dirname "$0")" && pwd)
repo=$(mktemp -d)
trap 'rm -rf "$repo"' EXIT
cd "$repo"
git init -q
git config user.email ci@example.invalid
git config user.name ci

check() {
  if [ "$2" != "$3" ]; then
    echo "FAIL $1: got '$2', want '$3'" >&2
    exit 1
  fi
  echo "ok   $1"
}

mkdir mobile weixin
echo 1 > mobile/a
echo 1 > weixin/a
git add .
git commit -qm one

check "no releases yet" "$("$here/next-version.sh")" 0.0.1
check "a tag that doesn't exist counts as changed" "$("$here/changed.sh" shipped/server mobile)" true

git tag app-v1.3.9
git tag app-v1.3.10
git tag app-v1.2.99
git tag app-vnot-a-version
check "1.3.10 is newer than 1.3.9" "$("$here/latest-app-tag.sh")" app-v1.3.10
check "patch" "$("$here/next-version.sh" patch)" 1.3.11
check "minor" "$("$here/next-version.sh" minor)" 1.4.0
check "major" "$("$here/next-version.sh" major)" 2.0.0

git tag shipped/server
check "nothing since it shipped" "$("$here/changed.sh" shipped/server mobile)" false
echo 2 > weixin/a
git commit -qam two
check "another part changed" "$("$here/changed.sh" shipped/server mobile)" false
check "this part changed" "$("$here/changed.sh" shipped/server mobile weixin)" true
echo 3 > mobile/a
git commit -qam three
echo 4 > mobile/b
git add .
git commit -qm four
check "a change two merges back still counts" "$("$here/changed.sh" shipped/server mobile)" true
