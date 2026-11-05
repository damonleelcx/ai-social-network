#!/usr/bin/env bash
# Prints true if any of these paths differ from a tag, or the tag doesn't exist yet.
#   changed.sh <tag> <path>...
set -euo pipefail
tag=$1
shift
if ! git rev-parse -q --verify "refs/tags/$tag" >/dev/null; then
  echo true
elif git diff --quiet "refs/tags/$tag" HEAD -- "$@"; then
  echo false
else
  echo true
fi
