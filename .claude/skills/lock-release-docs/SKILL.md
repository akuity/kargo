---
name: lock-release-docs
description: Lock docs and scripts on a release branch to its minor line
argument-hint: <release-branch>
disable-model-invocation: true
---

Lock all documentation and scripts on a release branch so they reference that
branch's minor line instead of floating references like `main` or `latest`.

**Arguments:** `$0` = release branch (e.g. `release-1.12`). The line is the
branch name without the `release-` prefix (e.g. `1.12`).

Nothing a lock writes names a patch version, so a locked branch never needs
re-locking when new patches ship. The line-level references resolve once the
line's first release (e.g. v1.12.0) is published.

## Before starting

1. Fetch from the `upstream` remote.
2. Check out `$0` and make sure it is up to date with `upstream/$0`.
3. Create a working branch (e.g. `lock-$0-docs`) off of `$0`.

## What to search for

Search `docs/` and `hack/` thoroughly for the patterns below. Do not assume a
fixed set of files -- new docs or scripts may introduce new instances over time.
Leave release notes alone.

A branch locked by an older version of this procedure pins an exact patch
(e.g. `--version 1.11.6`, `releases/download/v1.11.6/...`). Replace those pins
with the line-level forms below.

### Floating GitHub URLs pointing at `main`

Any `raw.githubusercontent.com/akuity/kargo/main/...`,
`github.com/akuity/kargo/tree/main/...`, or
`github.com/akuity/kargo/blob/main/...` URL should be updated to reference the
release branch instead (e.g. `refs/heads/$0` for raw content URLs, `tree/$0`
and `blob/$0` for links).

### CLI downloads

GitHub has no URL for the newest release of a line, so CLI download commands
first read the line's newest version from the pointer that the release workflow
publishes to GitHub Pages, then download that release. For line `1.12`:

```shell
version=$(curl -sL https://akuity.github.io/kargo/1.12/version.txt)
curl -L -o kargo https://github.com/akuity/kargo/releases/download/${version}/kargo-"$(uname -s | tr '[:upper:]' '[:lower:]')-${arch}"
```

```powershell
$version = Invoke-RestMethod -URI https://akuity.github.io/kargo/1.12/version.txt
Invoke-WebRequest -URI "https://github.com/akuity/kargo/releases/download/$version/kargo-windows-amd64.exe" -OutFile kargo.exe
```

Apply this to every `github.com/akuity/kargo/releases/latest/download/...` URL.
Keep the surrounding commands (e.g. the `arch` detection) as they are.

### Kargo Helm chart commands without a line constraint

Any `helm install`, `helm upgrade`, or `helm inspect values` command that
references `oci://ghcr.io/akuity/kargo-charts/kargo` needs `--version <line>.x`
(e.g. `--version 1.12.x`). Helm resolves the constraint to the newest
non-prerelease chart in the line. Place it on the line immediately after the
chart URL, before any `--namespace` or `--set` flags, to match the style used
by nearby third-party chart installs.

Do **not** modify helm commands for third-party charts (cert-manager, Argo CD,
Argo Rollouts, etc.) -- those manage their own version pins.

## After all edits

1. Search again for `main` URLs, `latest` downloads, and Kargo chart installs
   or CLI downloads naming a patch, to confirm nothing was missed.
2. Summarize every change (file, line, what changed) and present the diff.
3. Do **not** commit automatically -- wait for the user to review and approve.
