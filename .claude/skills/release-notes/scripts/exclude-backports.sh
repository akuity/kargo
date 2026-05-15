#!/usr/bin/env bash
# Phase 2: Exclude backported commits from the release commit range.
#
# Usage: exclude-backports.sh <base> <tip> <prev_minor_branch>
#
# Reads the commit range base..tip and removes:
#   1. Backport commits on the tip (chore(backport ...): prefix)
#   2. "Merge commit from fork" entries (security fixes)
#   3. Commits whose original PR was backported to the previous release branch
#
# For automated backports on the prev branch, fetches each backport PR's body
# to find the original PR number ("triggered by a label in #NNNN").
# For manual backports, extracts the original PR number from the subject.
#
# stdout: filtered commit list (hash + subject, one per line)
# stderr: summary of exclusions

set -euo pipefail

base="$1"
tip="$2"
prev_minor_branch="$3"

tmpdir=$(mktemp -d)
trap 'rm -rf "$tmpdir"' EXIT

# Collect all commits in the range
git log --oneline --no-merges "$base".."$tip" > "$tmpdir/all.txt"
total=$(wc -l < "$tmpdir/all.txt" | tr -d ' ')

# Collect commits on the previous release branch
git log --oneline --no-merges "$base".."$prev_minor_branch" > "$tmpdir/prev.txt"

# Extract original PR numbers from backport commits on the prev branch
> "$tmpdir/backported_prs.txt"
skipped_lookups=0

while IFS= read -r line; do
  subject="${line#* }"

  # Automated backports: chore(backport release-X.Y): original subject (#NNNN)
  if echo "$subject" | grep -q '^chore(backport'; then
    backport_pr=$(echo "$subject" | grep -oE '\(#[0-9]+\)' | tail -1 | sed 's/[^0-9]//g')
    if [ -n "$backport_pr" ]; then
      body=$(gh pr view "$backport_pr" --repo akuity/kargo --json body --jq '.body' 2>/dev/null || echo "")
      # Automated backports: "triggered by a label in #NNNN"
      original_pr=$(echo "$body" | grep -oE 'triggered by a label in #[0-9]+' | grep -oE '[0-9]+' | head -1 || true)
      if [ -n "$original_pr" ]; then
        echo "$original_pr" >> "$tmpdir/backported_prs.txt"
      else
        # Manual backports using chore(backport) prefix: body says
        # "Manual backport of #NNNN" (possibly multiple: "#NNNN and #MMMM")
        manual_prs=$(echo "$body" | grep -oiE '[Mm]anual backport of #[0-9]+(\s+and\s+#[0-9]+)*' | grep -oE '[0-9]+' || true)
        if [ -n "$manual_prs" ]; then
          for mpr in $manual_prs; do
            echo "$mpr" >> "$tmpdir/backported_prs.txt"
          done
        else
          skipped_lookups=$((skipped_lookups + 1))
          echo "WARNING: Could not find original PR for backport PR #$backport_pr" >&2
        fi
      fi
    fi
    continue
  fi

  # Manual backports: chore: manually backport #NNNN: ...
  if echo "$subject" | grep -qi 'manually backport #'; then
    original_pr=$(echo "$subject" | grep -oE 'backport #[0-9]+' | grep -oE '[0-9]+' | head -1)
    if [ -n "$original_pr" ]; then
      echo "$original_pr" >> "$tmpdir/backported_prs.txt"
    fi
  fi
done < "$tmpdir/prev.txt"

sort -u "$tmpdir/backported_prs.txt" -o "$tmpdir/backported_prs.txt"
backported_count=$(wc -l < "$tmpdir/backported_prs.txt" | tr -d ' ')

# Filter the main commit list
excluded_backport_entries=0
excluded_security_merges=0
excluded_backported_originals=0

while IFS= read -r line; do
  subject="${line#* }"

  # Drop backport commits on the tip branch (administrative/duplicate entries)
  if echo "$subject" | grep -q '^chore(backport'; then
    excluded_backport_entries=$((excluded_backport_entries + 1))
    continue
  fi

  # Drop "Merge commit from fork" entries
  if echo "$subject" | grep -q 'Merge commit from fork'; then
    excluded_security_merges=$((excluded_security_merges + 1))
    continue
  fi

  # Drop commits whose PR was backported to the prev branch
  pr_num=$(echo "$subject" | grep -oE '\(#[0-9]+\)' | tail -1 | sed 's/[^0-9]//g')
  if [ -n "$pr_num" ] && grep -qx "$pr_num" "$tmpdir/backported_prs.txt" 2>/dev/null; then
    excluded_backported_originals=$((excluded_backported_originals + 1))
    continue
  fi

  echo "$line"
done < "$tmpdir/all.txt"

excluded_total=$((excluded_backport_entries + excluded_security_merges + excluded_backported_originals))

echo "--- Backport Exclusion Summary ---" >&2
echo "Total commits in range: $total" >&2
echo "Backport entries dropped: $excluded_backport_entries" >&2
echo "Security merge entries dropped: $excluded_security_merges" >&2
echo "Originals backported to prev branch: $excluded_backported_originals (from $backported_count unique PRs)" >&2
if [ "$skipped_lookups" -gt 0 ]; then
  echo "WARNING: $skipped_lookups backport PRs could not be resolved" >&2
fi
echo "Total excluded: $excluded_total" >&2
echo "Remaining: $((total - excluded_total))" >&2
