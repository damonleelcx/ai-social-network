#!/usr/bin/env bash
# Moves a tag to this commit, so the next run knows what has already shipped.
#   mark-shipped.sh <tag>
set -euo pipefail
git tag -f "$1" HEAD
git push -f origin "refs/tags/$1"
