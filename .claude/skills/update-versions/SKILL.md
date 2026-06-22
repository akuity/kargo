---
name: update-versions
description: Check for and update hard-coded tool and chart versions that dependabot does not cover
disable-model-invocation: true
---

Find hard-coded version pins in the repository that are not managed by
dependabot, check whether newer versions are available, and (with user
approval) update them consistently across all locations.

## Scope

Dependabot already manages Go module dependencies (`go.mod` files), GitHub
Actions versions, Docker base image tags, and npm/UI dependencies. Some tool
versions in `hack/tools.mk` are read from `hack/tools/go.mod` at make-time
(via `$(shell grep ...)`); those are also covered. This skill targets
everything else.

**Do NOT touch:** `**/go.mod`, `**/go.sum`, `**/package.json`, Docker `FROM`
lines, or GitHub Actions workflow versions.

## Phase 1 -- Discovery

Search for version pins using the patterns below. Do NOT assume a fixed list of
files or tools -- new pins may be introduced over time.

### Pattern A: Makefile hard-coded versions

Search `Makefile` and `hack/**/*.mk` for variable assignments whose values are
literal version strings:

```
<NAME>_VERSION  := v1.2.3
<NAME>_CHART_VERSION  := 1.2.3
```

**Skip** any line whose right-hand side contains `$(shell` -- those pull from
`go.mod` and are managed by dependabot.

### Pattern B: Shell script version variables

Search `hack/**/*.sh` for assignments like:

```
<name>_version=1.2.3
<name>_chart_version=1.2.3
```

These are typically duplicates of Makefile variables and must stay in sync.

### Pattern C: Inline GitHub release URLs

Search all non-vendored, non-generated files for URLs embedding a version:

```
github.com/<owner>/<repo>/releases/download/v<version>/
```

### Pattern D: Dockerfile ARG version pins

Search `Dockerfile*` for `ARG <NAME>_VERSION=<value>` and inline assignments
like `<NAME>_VERSION=v<value>`. Skip `ARG VERSION` (the project's own build-
time version) and skip base image tags (`FROM ...`) which dependabot handles.

### Pattern E: Helm chart versions

Search `Makefile`, `hack/**/*.mk`, and `hack/**/*.sh` for `--version` flags
used with `helm install` or `helm upgrade`. Cross-reference with the chart
version variables found in Patterns A and B to build a complete mapping of
chart name, Helm repo URL, and all files where the version appears.

## Phase 2 -- Determine latest versions

For each discovered pin, determine the latest stable release. Skip pre-release,
alpha, beta, and RC versions.

### GitHub-hosted tools

Map each tool to its GitHub repo by inspecting install functions in
`hack/tools.mk` -- the download URLs or `go install` paths reveal the
`<owner>/<repo>`. Then:

```bash
gh api repos/<owner>/<repo>/releases/latest --jq '.tag_name'
```

### Helm charts

For each chart, use the repo URL found near `helm install` commands:

```bash
helm repo add <temp-name> <repo-url> --force-update 2>/dev/null
helm search repo <temp-name>/<chart-name> --versions -o json
```

Use the newest non-pre-release entry.

## Phase 3 -- Report

Present a table:

```
| Tool/Chart       | Current | Latest  | Status   | Files                                 |
|------------------|---------|---------|----------|---------------------------------------|
| protoc           | v25.3   | v28.1   | outdated | hack/tools.mk                         |
| argo-cd (chart)  | 8.1.4   | 8.2.0   | outdated | Makefile, install.sh, kind.sh, k3d.sh |
| kind             | v0.31.0 | v0.31.0 | current  | hack/tools.mk                         |
```

Flag any **major version bumps** prominently -- these may require build or
configuration changes.

**Do not proceed to updates** until the user reviews this table and confirms
which items to update.

## Phase 4 -- Apply updates

For each version the user approves:

1. Identify **all** locations where that version appears. Re-grep to confirm;
   do not rely on Phase 1 results alone.
2. Update every location. Preserve each file's conventions:
   - `v` prefix vs bare number (if current is `8.1.4`, new should be `8.2.0`,
     not `v8.2.0`)
   - `UPPER_SNAKE_CASE` in Makefiles, `lower_snake_case` in shell scripts
3. Grep for the **old** version string across the repo to confirm nothing was
   missed.

## Phase 5 -- Review

1. Show the full diff.
2. Summarize what changed: which tools/charts, old → new version, which files.
3. Do **not** commit automatically -- wait for the user to review and approve.
