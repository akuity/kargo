#!/usr/bin/env bash
# Reports Go release versions from go.dev and checks golang image tag
# availability on Docker Hub.
#
# Usage:
#   go-releases.sh latest              # latest stable Go, e.g. 1.27.1
#   go-releases.sh latest-in 1.26      # latest stable patch of 1.26, e.g. 1.26.8
#   go-releases.sh check-tag 1.27.1-trixie
#
# Requires: curl, python3

set -euo pipefail

fetch() {
  curl -fsSL 'https://go.dev/dl/?mode=json&include=all'
}

# Prints every stable version, one per line, newest first.
stable_versions() {
  fetch | python3 -c '
import json, sys
rels = json.load(sys.stdin)
vers = []
for r in rels:
    if not r.get("stable"):
        continue
    v = r["version"]
    if not v.startswith("go"):
        continue
    parts = v[2:].split(".")
    if len(parts) == 2:          # e.g. go1.27 means go1.27.0
        parts.append("0")
    if len(parts) != 3 or not all(p.isdigit() for p in parts):
        continue
    vers.append(tuple(int(p) for p in parts))
for v in sorted(set(vers), reverse=True):
    print("%d.%d.%d" % v)
'
}

case "${1:-}" in
latest)
  stable_versions | head -1
  ;;
latest-in)
  minor="${2:?usage: go-releases.sh latest-in <major.minor>}"
  out="$(stable_versions | grep -E "^${minor//./\\.}\." | head -1 || true)"
  if [ -z "$out" ]; then
    echo "no stable release found in Go $minor" >&2
    exit 1
  fi
  echo "$out"
  ;;
check-tag)
  tag="${2:?usage: go-releases.sh check-tag <tag>}"
  code="$(
    curl -sS -o /dev/null -w '%{http_code}' \
      "https://hub.docker.com/v2/repositories/library/golang/tags/${tag}"
  )"
  if [ "$code" = "200" ]; then
    echo "ok: golang:${tag} exists"
  else
    echo "MISSING: golang:${tag} (HTTP ${code})" >&2
    exit 1
  fi
  ;;
*)
  sed -n '2,12p' "$0" >&2
  exit 64
  ;;
esac
