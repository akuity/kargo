#!/usr/bin/env bash
# Phase 3: Classify commits by subject line alone (no API calls).
#
# Usage: classify-by-subject.sh <input_file> <included_file> <excluded_file> <uncertain_file>
#
# Reads a commit list (hash + subject per line) and classifies each into:
#   - included: clearly noteworthy (features, breaking changes)
#   - excluded: clearly uninteresting (docs, deps, CI, typos, chores)
#   - uncertain: needs PR/issue context to decide
#
# stderr: summary counts

set -euo pipefail

input_file="$1"
included_file="$2"
excluded_file="$3"
uncertain_file="$4"

> "$included_file"
> "$excluded_file"
> "$uncertain_file"

while IFS= read -r line; do
  subject="${line#* }"

  # --- Auto-exclude patterns ---

  # Docs-only changes
  if echo "$subject" | grep -qE '^docs[:(]'; then
    echo "$line" >> "$excluded_file"
    continue
  fi

  # Dependency bumps
  if echo "$subject" | grep -qE '^chore\(deps[)/:]|^chore\(dep\):|^build\(deps'; then
    echo "$line" >> "$excluded_file"
    continue
  fi

  # CI/build chores
  if echo "$subject" | grep -qE '^chore\(ci\):|^fix\(ci\):'; then
    echo "$line" >> "$excluded_file"
    continue
  fi

  # Typo fixes
  if echo "$subject" | grep -qiE '^fix:.*typo'; then
    echo "$line" >> "$excluded_file"
    continue
  fi

  # Miscellaneous chores (unscoped and scoped)
  if echo "$subject" | grep -qE '^chore[:(]'; then
    echo "$line" >> "$excluded_file"
    continue
  fi

  # --- Auto-include patterns ---

  # Breaking changes (conventional commits ! indicator)
  if echo "$subject" | grep -qE '!:'; then
    echo "$line" >> "$included_file"
    continue
  fi

  # Breaking changes (keyword)
  if echo "$subject" | grep -qi 'breaking'; then
    echo "$line" >> "$included_file"
    continue
  fi

  # New features
  if echo "$subject" | grep -qE '^feat[:(]'; then
    echo "$line" >> "$included_file"
    continue
  fi

  # --- Everything else is uncertain ---
  echo "$line" >> "$uncertain_file"

done < "$input_file"

included_count=$(wc -l < "$included_file" | tr -d ' ')
excluded_count=$(wc -l < "$excluded_file" | tr -d ' ')
uncertain_count=$(wc -l < "$uncertain_file" | tr -d ' ')

echo "--- Subject Classification Summary ---" >&2
echo "Auto-included: $included_count" >&2
echo "Auto-excluded: $excluded_count" >&2
echo "Need enrichment: $uncertain_count" >&2
