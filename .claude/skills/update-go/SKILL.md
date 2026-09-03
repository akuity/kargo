---
name: update-go
description: Bump every pinned Go toolchain version on one or more branches, respecting the ship/test split
disable-model-invocation: true
---

Raise the Go versions pinned across a branch's Dockerfiles and workflows, on
one or more branches at a time, then hand the user a script that commits,
pushes, and opens the pull requests.

Dependabot cannot do this. Its `docker` ecosystem sees only `Dockerfile` and
`Dockerfile.dev`; it cannot see the `container.image` pins in the workflows,
and it cannot tell the two roles below apart. That is why Go is excluded from
`.github/dependabot.yml` and why this skill exists.

## The two roles

Every Go version pin in this repository serves exactly one of two roles. The
comments at each site say which. **Read the comment before editing the line.**

| Role | Target version | Why |
|------|----------------|-----|
| **ship** | latest stable Go, period | A release branch outlives the Go minor it was cut on. Building on the latest minor keeps released binaries off unsupported toolchains and picks up Go stdlib CVE fixes. |
| **test** | latest patch **within the minor named by the root `go.mod` `go` directive** | Unit tests, linters, and codegen run here. The minor is capped: a golangci-lint binary built with go1.X hard-panics under go1.(X+1). |

**Never raise the test minor.** Patch bumps inside a minor are always safe. A
minor raise forces a golangci-lint upgrade, which surfaces new findings needing
code or config changes -- a separate, human-driven change. Root `go.mod`'s `go`
directive is the authority on which minor the branch tests against; if the test
sites disagree with it, stop and report rather than guessing.

## Site inventory

The layout differs by branch, so **discover, do not assume**. Sites seen so
far:

**ship**
- `Dockerfile` -- every `FROM ... golang:<ver>-<variant>` stage
  (`back-end-builder`, `helm-builder`)
- `.github/workflows/release.yaml` -- `image: &golangImage golang:<ver>-...`
  and the `actions/setup-go` step's `go-version:`
- `.github/workflows/ci.yaml` -- the `build-cli` job's `container.image`
- `.github/actions/setup-go/action.yaml` -- the `go-version` input `default:`
  (this action's default *is* the ship version; jobs that test override it)

**test**
- `Dockerfile.dev` -- the `FROM golang:<ver>-<variant>` base image
- `.github/workflows/ci.yaml` -- `go-version: &goTestVersion "<ver>"` or
  `image: &golangTestImage golang:<ver>-<variant>`, depending on branch

**Out of scope.** Do not touch:
- `go`/`toolchain` directives in any `go.mod`. The `go` directive is a language
  floor, not a toolchain pin; here it is the *input* that tells you the test
  minor.
- The golangci-lint pin (`hack/tools.mk` on newer branches, `hack/tools/go.mod`
  on older ones).
- Non-Go base images (`node`, `alpine`), which dependabot does manage.

## Phase 1 -- Resolve versions and prepare worktrees

`scripts/go-releases.sh` answers the version questions; it takes no repo state:

```bash
.claude/skills/update-go/scripts/go-releases.sh latest          # ship target
.claude/skills/update-go/scripts/go-releases.sh latest-in 1.26  # test target
.claude/skills/update-go/scripts/go-releases.sh check-tag 1.27.1-trixie
```

Run `check-tag` for every `<ver>-<variant>` combination you intend to write.
A Go release can land on go.dev before the Docker Hub image exists, and a
missing tag breaks every build on the branch.

`scripts/prepare.sh` creates the isolated workspaces:

```bash
.claude/skills/update-go/scripts/prepare.sh bump-go main release-1.8 release-1.9
```

It resolves the canonical remote by URL (`github.com/akuity/kargo`) rather than
by name -- `upstream` is one contributor's convention, not a guarantee --
fetches it, and puts each target branch in its own worktree at
`<repo>-worktrees/<repo>-<version>`, on a fresh feature branch cut from the
canonical tip. Branches are therefore current with respect to the real
repository by construction; do not merge or rebase afterward. It refuses to
reuse a worktree with uncommitted changes. Capture its `KEY=value` output --
the finish script needs the remote, owner, feature branch, and path.

## Phase 2 -- Per branch, classify then edit

In each worktree, independently:

1. Enumerate candidate sites:

   ```bash
   git grep -nIE 'golang:[0-9]+\.[0-9]+|1\.[0-9]{2}\.[0-9]+' \
     -- Dockerfile Dockerfile.dev .github/
   ```

   Match on the version literal, not on `go-version`: the `setup-go` action's
   ship default sits on a `default:` line, and a `go-version: *anchor`
   reference carries no version of its own.

2. Read the surrounding comment for each hit and assign it a role. **If a hit
   matches no entry in the inventory above, or its comment contradicts the
   role you would have assigned, stop and ask.** A silently misclassified site
   is the exact failure this skill exists to prevent: a shipped binary built on
   the capped toolchain still passes CI.

3. Establish the branch's test target: read the root `go.mod` `go` directive,
   take its minor, and resolve `latest-in <minor>`. Confirm every test site
   currently sits on that same minor.

4. Rewrite each site to its role's target, preserving the tag variant
   (`-trixie`, etc.) and the existing quoting style exactly.

5. Verify: re-grep and confirm every remaining Go version is either the ship
   target or the test target, and that no site kept an old value. Grep the
   whole branch for the old version strings to catch a site outside the
   inventory.

## Phase 3 -- Report

Present one table for the whole run, then the diffs:

```
| Branch       | Role | Current | Target | Sites |
|--------------|------|---------|--------|-------|
| main         | ship | 1.27.0  | 1.27.1 | Dockerfile x2, setup-go action |
| main         | test | 1.27.0  | 1.27.1 | Dockerfile.dev, ci.yaml anchor |
| release-1.8  | ship | 1.27.0  | 1.27.1 | Dockerfile x2, ci.yaml, release.yaml x2 |
| release-1.8  | test | 1.25.14 | 1.25.14 (current) | -- |
```

Call out separately, without acting on any of it:
- branches whose test minor is behind the latest Go minor, and the
  golangci-lint upgrade that unblocking would require
- any site whose role you could not determine

## Phase 4 -- Leave a finish script

Write `bump-go-finish.sh` to the **primary checkout's root** (not a worktree,
not `/tmp`) so it is visible in the editor. Make it executable. Do not run it
-- the user commits, pushes, and opens PRs themselves.

The script must, per branch, `cd` to the worktree and:

1. `git commit -s` (DCO sign-off is mandatory) with a subject that names what
   actually moved:
   - both roles, same target: `chore(deps): bump Go to <ship>`
   - both roles, different targets:
     `chore(deps): bump Go to <ship> and CI Go to <test>`
   - ship only: `chore(deps): bump Go to <ship> for shipped artifacts`
   - test only: `chore(deps): bump CI Go to <test>`

   End the message with:

   ```
   Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>
   ```

2. `git push --force-with-lease` to the fork remote resolved in Phase 1.

3. Open the PR against the canonical repository:

   ```bash
   gh pr create --repo akuity/kargo \
     --base "<base>" --head "<fork-owner>:<feature-branch>" \
     --title "..." --body-file - --assignee @me \
     --label kind/chore --label area/security \
     --label area/ci-process --label area/release-process \
     --label priority/normal --label dependencies --label docker
   ```

   Those labels follow the prior art for this change (#6922, #7011). No
   `backport/*` label: each branch gets its own PR, so there is nothing to
   backport. Write PR bodies with one line per paragraph -- GitHub reflows, and
   hard wraps only make editing harder.

Structure the script so each branch is a self-contained block that can be
re-run, with `set -euo pipefail` at the top and an echo naming each branch as
it goes. Skip a branch that already has an open PR for its head, and skip the
commit when the worktree is clean, so a partial run can be resumed.

## Phase 5 -- Stray dependabot PRs

Dependabot no longer proposes Go bumps, but PRs opened before that exclusion
landed can still be open. Check:

```bash
gh pr list --repo akuity/kargo --state open --author app/dependabot \
  --json number,title,baseRefName \
  --jq '.[] | select(.title|test("golang")) | "\(.number) (\(.baseRefName)) \(.title)"'
```

Reference any hit from the corresponding branch's PR body as superseded, and
tell the user to close it once the replacement is open. Read its diff first --
say specifically what it got wrong, rather than asserting it was wrong.
