#!/usr/bin/env bash
# The version after the latest released one.   next-version.sh [patch|minor|major]
set -euo pipefail
bump=${1:-patch}
latest=$("$(dirname "$0")/latest-app-tag.sh")
latest=${latest#app-v}
IFS=. read -r major minor patch <<<"${latest:-0.0.0}"
case "$bump" in
  patch) patch=$((patch + 1)) ;;
  minor) minor=$((minor + 1)) patch=0 ;;
  major) major=$((major + 1)) minor=0 patch=0 ;;
  *) echo "unknown bump: $bump" >&2; exit 2 ;;
esac
echo "$major.$minor.$patch"
