#!/usr/bin/env bash
# Fetch a Kargo GitHub security advisory and save its title, URL, and
# description verbatim to ADVISORY-<id>.md in the current directory.
#
# Usage: fetch-advisory.sh <GHSA-id | advisory URL>
set -euo pipefail

if [ $# -ne 1 ]; then
  echo "usage: $0 <GHSA-id | advisory URL>" >&2
  exit 1
fi

id=$(printf '%s' "$1" | grep -oE 'GHSA-[a-z0-9]{4}-[a-z0-9]{4}-[a-z0-9]{4}' | head -1)
if [ -z "$id" ]; then
  echo "could not parse a GHSA id from: $1" >&2
  exit 1
fi

out="ADVISORY-${id}.md"

gh api "repos/akuity/kargo/security-advisories/${id}" \
  --jq '"# \(.ghsa_id): \(.summary)\n\n\(.html_url)\n\nReporter severity: \(.severity // "none")  CVSS: \(.cvss_severities.cvss_v4.vector_string // .cvss.vector_string // "none")\n\n\(.description)"' \
  | sed 's/\r$//' > "$out"

echo "wrote $out ($(wc -l < "$out" | tr -d ' ') lines)"
