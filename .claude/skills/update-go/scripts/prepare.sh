#!/usr/bin/env bash
# Prepares one isolated worktree per target branch, each on a fresh feature
# branch cut from the canonical remote's tip.
#
# Usage: prepare.sh <slug> <branch>...
#   e.g. prepare.sh bump-go main release-1.8 release-1.9
#
# Emits a `KEY=value` block per branch on stdout for the caller to consume.
# Refuses to touch a worktree with uncommitted changes.

set -euo pipefail

CANONICAL_SLUG="akuity/kargo"

slug="${1:?usage: prepare.sh <slug> <branch>...}"
shift
[ "$#" -gt 0 ] || { echo "no target branches given" >&2; exit 64; }

repo_root="$(git rev-parse --show-toplevel)"
repo_name="$(basename "$repo_root")"
worktree_parent="$(dirname "$repo_root")/${repo_name}-worktrees"

# normalize turns any GitHub remote URL into <owner>/<repo>.
normalize() {
  printf '%s' "$1" |
    sed -E \
      -e 's#^git@github\.com:#/#' \
      -e 's#^ssh://git@github\.com/#/#' \
      -e 's#^https?://[^/]*github\.com/#/#' \
      -e 's#\.git$##' \
      -e 's#^/##'
}

# Resolve the canonical remote by URL, never by name -- "upstream" is one
# person's convention, not a guarantee.
canonical=""
for r in $(git remote); do
  if [ "$(normalize "$(git remote get-url "$r")")" = "$CANONICAL_SLUG" ]; then
    canonical="$r"
    break
  fi
done
if [ -z "$canonical" ]; then
  echo "no remote points at https://github.com/${CANONICAL_SLUG}" >&2
  exit 1
fi

login="$(gh api user --jq .login)"

# Resolve the fork to push to: a remote owned by the authenticated user.
# Prefer an SSH push URL when the same fork is configured more than once.
fork=""
for r in $(git remote); do
  owner="$(normalize "$(git remote get-url "$r")")"
  owner="${owner%%/*}"
  [ "$owner" = "$login" ] || continue
  if [ -z "$fork" ]; then
    fork="$r"
  elif [[ "$(git remote get-url --push "$r")" == git@* ]] &&
       [[ "$(git remote get-url --push "$fork")" != git@* ]]; then
    fork="$r"
  fi
done
if [ -z "$fork" ]; then
  echo "WARNING: no remote owned by ${login}; falling back to ${canonical}" >&2
  fork="$canonical"
fi

echo "CANONICAL_REMOTE=$canonical"
echo "FORK_REMOTE=$fork"
echo "FORK_OWNER=$login"
echo "WORKTREE_PARENT=$worktree_parent"
echo

git fetch --quiet "$canonical" "$@"

mkdir -p "$worktree_parent"

for base in "$@"; do
  # main -> "main"; release-1.8 -> "1.8"
  version="${base#release-}"
  path="${worktree_parent}/${repo_name}-${version}"
  if [ "$base" = "main" ]; then
    feature="${login}/${slug}"
  else
    feature="${login}/${slug}-${version}"
  fi

  if [ -d "$path" ] && git -C "$path" rev-parse --git-dir >/dev/null 2>&1; then
    if [ -n "$(git -C "$path" status --porcelain)" ]; then
      echo "REFUSING: ${path} has uncommitted changes" >&2
      exit 1
    fi
    git -C "$path" switch --quiet --force-create "$feature" \
      "${canonical}/${base}"
  else
    git worktree add --quiet -B "$feature" "$path" "${canonical}/${base}"
  fi

  echo "BASE=$base"
  echo "VERSION=$version"
  echo "FEATURE_BRANCH=$feature"
  echo "WORKTREE=$path"
  echo "TIP=$(git -C "$path" rev-parse --short HEAD)"
  echo
done
