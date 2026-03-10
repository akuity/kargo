---
sidebar_label: github-verified-push
description: Pushes committed changes to a GitHub repository by replaying commits through the GitHub REST API, enabling "Verified" commits when authenticating with a GitHub App.
---

# `github-verified-push`

<span class="tag beta"></span>

`github-verified-push` pushes committed changes from a local working tree to a
GitHub repository using the GitHub REST API. It is a drop-in replacement for
[`git-push`](git-push.md) that replays commits through the API instead of using
`git push`.

This step exists because GitHub Apps have no associated GPG to sign their commits
with in the standard way. If your Kargo installation authenticates to GitHub via a
[GitHub App installation token](../../50-security/30-managing-secrets.md#github-app-authentication)
and your branch protection rules require verified signatures, this step provides
an alternative solution. Commits created through the API with an
App token are automatically signed by GitHub and marked as "Verified" — without
requiring GPG key management on the GitHub side.

Under the hood it:

1. Compares the local branch to the remote target branch to identify new commits
1. Pushes local commits to a temporary, non-branch staging ref on GitHub
   (invisible in the branch list)
1. Replays each commit from the staging ref onto the target branch via the
   GitHub REST API, creating new commits with new SHAs
1. Updates the target branch to the final replayed commit (fast-forward by
   default, or force-update when `force: true`)
1. Deletes the staging ref

Because commits are recreated through the API, the resulting remote commits have
different SHAs than the original local commits — even when the content is
identical. This means the local and remote branches will have diverged after this
step completes. This is expected and does not affect subsequent promotions, since
each promotion clones a fresh working tree.

:::info

This step requires Git credentials with write access to the target repository.
Two authentication methods are supported:

- **GitHub App installation token** (recommended) — Commits replayed through
  the API are automatically signed by GitHub and marked as "Verified." The
  GitHub App must have **Contents: read & write** permission. See
  [GitHub App Authentication](../../50-security/30-managing-secrets.md#github-app-authentication)
  for setup instructions.
- **Personal access token (PAT)** — Commits are replayed successfully and
  attributed to the PAT owner, but are **not** marked as "Verified." This
  can be useful for testing.

:::

## Commit Verification and Authorship

All commits replayed by this step are created through the GitHub API without
explicit author or committer fields. This means the authenticated identity
(GitHub App or PAT owner) becomes the commit's author and committer, and — when
using a GitHub App — the commit receives GitHub's "Verified" badge.

When the original commit's author differs from Kargo's system identity, a
`Co-authored-by` trailer is added to the commit message to credit the original
author. GitHub renders these trailers as linked avatars in the commit UI.

:::note

If you use the [`git-commit`](git-commit.md) step with a custom `author`
configuration, that author will appear as a co-author on the resulting
GitHub commit.

:::

### Integrity Checking

When the Kargo controller is configured with a
[GPG signing key](../../../40-operator-guide/20-advanced-installation/30-common-configurations.md#signing-commits),
this step verifies GPG signatures on local commits before replaying them.
This is an integrity check — it does not affect which commits receive the
"Verified" badge. Commits with bad or revoked GPG signatures cause the step
to fail with a terminal error. All other signature statuses (including
unsigned commits) pass through without error.

## Configuration

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `path` | `string` | Y | Path to a Git working tree containing committed changes. |
| `targetBranch` | `string` | N | The branch to push to in the remote repository. Mutually exclusive with `generateTargetBranch=true`. If neither of these is provided, the target branch will be the same as the branch currently checked out in the working tree. |
| `generateTargetBranch` | `boolean` | N | Whether to push to a remote branch named like `kargo/promotion/<promotionName>`. A value of `true` is mutually exclusive with `targetBranch`. This is useful when a subsequent step will open a pull request. |
| `force` | `boolean` | N | Whether to force push to the target branch, overwriting any existing history. This is useful for scenarios where you want to completely replace the branch content (e.g., pushing rendered manifests that don't depend on previous state). **Use with caution** as this will overwrite any commits that exist on the remote branch but not in your local branch. Default is `false`. |
| `insecureSkipTLSVerify` | `boolean` | N | Whether to skip TLS verification when communicating with the GitHub API. Default is `false`. Intended for GitHub Enterprise instances with self-signed certificates. |

## Output

| Name | Type | Description |
|------|------|-------------|
| `branch` | `string` | The name of the remote branch pushed to by this step. |
| `commit` | `string` | The ID (SHA) of the final commit created by this step. |
| `commitURL` | `string` | The URL of the final commit on GitHub. |

## Examples

### Common Usage

In this example, changes are committed locally and then pushed to the same
branch as verified commits. This replaces the typical
`git-commit` + `git-push` pattern when commit verification is required.

```yaml
steps:
# Clone, prepare the contents of ./out, etc...
- uses: git-commit
  config:
    path: ./out
    message: rendered updated manifests
- uses: github-verified-push
  config:
    path: ./out
```

### For Use With a Pull Request

In this example, changes are pushed to a generated branch as verified commits,
then a pull request is opened. The step's output includes the generated branch
name, which is referenced by the subsequent
[`git-open-pr` step](git-open-pr.md).

```yaml
steps:
# Clone, prepare the contents of ./out, etc...
- uses: git-commit
  config:
    path: ./out
    message: rendered updated manifests
- uses: github-verified-push
  as: push
  config:
    path: ./out
    generateTargetBranch: true
- uses: git-open-pr
  config:
    repoURL: https://github.com/example/repo
    sourceBranch: ${{ outputs.push.branch }}
    targetBranch: main
# Wait for PR to be merged or closed...
```

### Explicit Target Branch

Push verified commits to a specific branch rather than the currently checked out
branch.

```yaml
steps:
# Clone, prepare the contents of ./out, etc...
- uses: git-commit
  config:
    path: ./out
    message: rendered updated manifests
- uses: github-verified-push
  config:
    path: ./out
    targetBranch: main
```

### Force Push

Force push to the target branch, replacing its history with the local branch
content. This is useful when pushing rendered manifests that don't depend on
previous state.

```yaml
steps:
# Clone, prepare the contents of ./out, etc...
- uses: git-commit
  config:
    path: ./out
    message: rendered updated manifests
- uses: github-verified-push
  config:
    path: ./out
    targetBranch: stage/test
    force: true
```
