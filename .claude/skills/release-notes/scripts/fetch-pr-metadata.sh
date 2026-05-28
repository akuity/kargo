#!/usr/bin/env bash
# Phase 4: Fetch PR and linked issue metadata from GitHub.
#
# Usage: fetch-pr-metadata.sh <input_file> <output_dir>
#
# Reads a commit list (hash + subject per line), extracts PR numbers from
# subjects using the (#NNNN) pattern, and fetches metadata for each PR.
# Also discovers and fetches linked issues from PR bodies.
#
# Creates in output_dir:
#   - pr-NNNN.json for each PR (title, body, labels, author)
#   - issue-NNNN.json for each linked issue (title, body, labels)
#   - index.tsv with columns: PR_NUMBER, TITLE, AUTHOR, LABELS, LINKED_ISSUES
#
# Skips PRs/issues that have already been fetched (idempotent).
#
# stderr: progress and summary

set -euo pipefail

input_file="$1"
output_dir="$2"

mkdir -p "$output_dir"

# Extract unique PR numbers from commit subjects
pr_numbers=()
while IFS= read -r line; do
  subject="${line#* }"
  pr_num=$(echo "$subject" | grep -oE '\(#[0-9]+\)' | tail -1 | sed 's/[^0-9]//g')
  if [ -n "$pr_num" ]; then
    pr_numbers+=("$pr_num")
  fi
done < "$input_file"

# Deduplicate
mapfile -t pr_numbers < <(printf '%s\n' "${pr_numbers[@]}" | sort -un)

total_prs=${#pr_numbers[@]}
fetched_prs=0
skipped_prs=0
total_issues=0

echo "Fetching metadata for $total_prs PRs..." >&2

# Fetch each PR
for pr_num in "${pr_numbers[@]}"; do
  pr_file="$output_dir/pr-$pr_num.json"

  if [ -f "$pr_file" ]; then
    skipped_prs=$((skipped_prs + 1))
  else
    if gh pr view "$pr_num" --repo akuity/kargo --json title,body,labels,author > "$pr_file" 2>/dev/null; then
      fetched_prs=$((fetched_prs + 1))
    else
      echo "{}" > "$pr_file"
      echo "WARNING: Failed to fetch PR #$pr_num" >&2
    fi
  fi

  # Extract linked issues from PR body
  body=$(jq -r '.body // ""' "$pr_file" 2>/dev/null || echo "")

  # Match: Fixes #N, Closes #N, Resolves #N
  linked_issues=$(echo "$body" | grep -oiE '(fixes|closes|resolves)\s+#[0-9]+' | grep -oE '[0-9]+' || true)
  # Match: github.com/akuity/kargo/issues/N
  linked_issues="$linked_issues $(echo "$body" | grep -oE 'github\.com/akuity/kargo/issues/[0-9]+' | grep -oE '[0-9]+' || true)"

  for issue_num in $linked_issues; do
    issue_file="$output_dir/issue-$issue_num.json"
    if [ ! -f "$issue_file" ]; then
      if gh issue view "$issue_num" --repo akuity/kargo --json title,body,labels > "$issue_file" 2>/dev/null; then
        total_issues=$((total_issues + 1))
      else
        echo "{}" > "$issue_file"
        echo "WARNING: Failed to fetch issue #$issue_num" >&2
      fi
    fi
  done
done

# Build index.tsv
echo -e "PR\tTITLE\tAUTHOR\tLABELS\tLINKED_ISSUES" > "$output_dir/index.tsv"
for pr_num in "${pr_numbers[@]}"; do
  pr_file="$output_dir/pr-$pr_num.json"
  title=$(jq -r '.title // "N/A"' "$pr_file" 2>/dev/null || echo "N/A")
  author=$(jq -r '.author.login // "N/A"' "$pr_file" 2>/dev/null || echo "N/A")
  labels=$(jq -r '[.labels[]?.name] | join(",")' "$pr_file" 2>/dev/null || echo "")
  body=$(jq -r '.body // ""' "$pr_file" 2>/dev/null || echo "")
  issues=$(echo "$body" | grep -oiE '(fixes|closes|resolves)\s+#[0-9]+' | grep -oE '[0-9]+' | tr '\n' ',' | sed 's/,$//' || true)
  issues2=$(echo "$body" | grep -oE 'github\.com/akuity/kargo/issues/[0-9]+' | grep -oE '[0-9]+' | tr '\n' ',' | sed 's/,$//' || true)
  if [ -n "$issues" ] && [ -n "$issues2" ]; then
    issues="$issues,$issues2"
  elif [ -n "$issues2" ]; then
    issues="$issues2"
  fi
  echo -e "$pr_num\t$title\t$author\t$labels\t$issues" >> "$output_dir/index.tsv"
done

echo "--- PR Metadata Summary ---" >&2
echo "PRs fetched: $fetched_prs (skipped $skipped_prs already cached)" >&2
echo "Linked issues fetched: $total_issues" >&2
echo "Index written to $output_dir/index.tsv" >&2
