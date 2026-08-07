---
name: lock-release-docs
description: Lock docs and scripts on a release branch to a specific patch version
argument-hint: <release-branch> <version>
disable-model-invocation: true
---

Lock all documentation and scripts on a release branch so they reference a
specific version instead of floating references like `main` or `latest`.

**Arguments:** `$0` = release branch (e.g. `release-1.9`), `$1` = version to
lock to (e.g. `1.9.3`)./

## Before starting

1. Fetch from the `upstream` remote.
2. Check out `$0` and make sure it is up to date with `upstream/$0`.
3. Create a working branch (e.g. `lock-$0-docs`) off of `$0`.

## What to search for

Search `docs/` and `hack/` thoroughly for the patterns below. Do not assume a
fixed set of files -- new docs or scripts may introduce new instances over time.

### Floating GitHub URLs pointing at `main`

Any `raw.githubusercontent.com/akuity/kargo/main/...` or
`github.com/akuity/kargo/tree/main/...` URL should be updated to reference
the release branch instead (e.g. `refs/heads/$0` for raw content URLs,
`tree/$0` for tree links).

### Floating GitHub release URLs using `latest`

Any `github.com/akuity/kargo/releases/latest/download/...` URL should be
pinned to the specific version: `releases/download/v$1/...`.

### Kargo Helm chart commands without a version pin

Any `helm install`, `helm upgrade`, or `helm inspect values` command that
references `oci://ghcr.io/akuity/kargo-charts/kargo` without a `--version`
flag needs `--version $1` added. Place it on the line immediately after the
chart URL, before any `--namespace` or `--set` flags, to match the style used
by nearby third-party chart installs.

Do **not** modify helm commands for third-party charts (cert-manager, Argo CD,
Argo Rollouts, etc.) -- those manage their own version pins.

## After all edits

1. Summarize every change (file, line, what changed) and present the diff.
2. Do **not** commit automatically -- wait for the user to review and approve.
