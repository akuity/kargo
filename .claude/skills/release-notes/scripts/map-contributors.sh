#!/usr/bin/env bash
# Phase 5: Identify first-time contributors and map emails to GitHub logins.
#
# Usage: map-contributors.sh <base> <tip> [<prev_minor_branch>]
#
# Finds all commit author emails in the range base..tip (and optionally
# base..prev_minor_branch for patch release contributors). For each email
# with no commits before the base, maps the email to a GitHub login using
# a deterministic method: find a commit SHA by that email, then query the
# GitHub API for the commit's author login.
#
# Excludes bot emails (dependabot, renovate, github-actions).
#
# stdout: email|login lines (one per first-time contributor)
# stderr: summary

set -euo pipefail

base="$1"
tip="$2"
prev_minor_branch="${3:-}"

tmpdir=$(mktemp -d)
trap 'rm -rf "$tmpdir"' EXIT

# Collect all author emails in the range
git log --format='%ae' --no-merges "$base".."$tip" | sort -u > "$tmpdir/range_emails.txt"

# Include contributors from the previous release branch (patch releases)
if [ -n "$prev_minor_branch" ]; then
  git log --format='%ae' --no-merges "$base".."$prev_minor_branch" | sort -u >> "$tmpdir/range_emails.txt"
  sort -u "$tmpdir/range_emails.txt" -o "$tmpdir/range_emails.txt"
fi

total_authors=$(wc -l < "$tmpdir/range_emails.txt" | tr -d ' ')

# Filter out bots (dependabot, renovate, github-actions, akuitybot, etc.)
grep -viE 'dependabot|renovate|github-actions|akuitybot|\[bot\]' "$tmpdir/range_emails.txt" \
  > "$tmpdir/human_emails.txt" || true

# Find first-time contributors (no commits before base)
> "$tmpdir/first_timers.txt"
while IFS= read -r email; do
  prior=$(git log --author="$email" --oneline "$base" -1 2>/dev/null || echo "")
  if [ -z "$prior" ]; then
    echo "$email" >> "$tmpdir/first_timers.txt"
  fi
done < "$tmpdir/human_emails.txt"

first_timer_count=$(wc -l < "$tmpdir/first_timers.txt" | tr -d ' ')

# Map emails to GitHub logins deterministically
mapped=0
failed=0

while IFS= read -r email; do
  # Find a commit SHA authored by this email
  sha=$(git log --author="$email" --format="%H" "$base".."$tip" -1 2>/dev/null || echo "")
  if [ -z "$sha" ] && [ -n "$prev_minor_branch" ]; then
    sha=$(git log --author="$email" --format="%H" "$base".."$prev_minor_branch" -1 2>/dev/null || echo "")
  fi

  if [ -n "$sha" ]; then
    login=$(gh api "/repos/akuity/kargo/commits/$sha" --jq '.author.login' 2>/dev/null || echo "")
    if [ -n "$login" ] && [ "$login" != "null" ]; then
      echo "$email|$login"
      mapped=$((mapped + 1))
    else
      echo "$email|UNKNOWN"
      failed=$((failed + 1))
      echo "WARNING: Could not resolve login for $email (SHA: $sha)" >&2
    fi
  else
    echo "$email|UNKNOWN"
    failed=$((failed + 1))
    echo "WARNING: No commit SHA found for $email" >&2
  fi
done < "$tmpdir/first_timers.txt"

echo "--- Contributor Mapping Summary ---" >&2
echo "Total unique authors: $total_authors" >&2
echo "First-time contributors: $first_timer_count" >&2
echo "Successfully mapped: $mapped" >&2
if [ "$failed" -gt 0 ]; then
  echo "Failed to map: $failed" >&2
fi
