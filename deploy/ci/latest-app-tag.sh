#!/usr/bin/env bash
# The newest released app version's tag. Sorted by version, because 1.3.10 comes after 1.3.9.
set -euo pipefail
git tag --list 'app-v[0-9]*' | sort -V | tail -1
