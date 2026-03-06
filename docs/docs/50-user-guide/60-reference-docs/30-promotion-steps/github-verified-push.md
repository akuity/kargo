---
sidebar_label: github-verified-push
description: Pushes committed changes to a GitHub repository by replaying commits through the GitHub REST API, producing commits that GitHub marks as "Verified".
---

# `github-verified-push`

<span class="tag beta"></span>

`github-verified-push` pushes committed changes from a local working tree to a
GitHub repository using the GitHub REST API. It is a drop-in replacement for
[`git-push`](git-push.md) that replays commits through the API instead of using
`git push`.

This step is designed to work with repositories that enforce commit verification
via branch protection rules. When used with a
[GitHub App installation token](../../50-security/30-managing-secrets.md#github-app-authentication),
commits created through the API are automatically trusted by GitHub and marked as
"Verified" — without requiring GPG key management on the GitHub side. Under the
hood it:

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

This step requires a **GitHub App installation token** stored as Git
credentials for the repository. The GitHub App must have **Contents: read &
write** permission on the target repository. See
[GitHub App Authentication](../../50-security/30-managing-secrets.md#github-app-authentication)
for setup instructions.

:::

## Commit Verification Behavior

Which commits receive GitHub's "Verified" badge depends on whether the Kargo
controller is configured with its own
[GPG signing key](../../../40-operator-guide/20-advanced-installation/30-common-configurations.md#signing-commits).

**When a signing key is configured**, this step checks the GPG signature on
each local commit before replaying it. The GPG signature is used as a trust
signal to determine which commits are eligible for verification — it is not
the mechanism that produces the "Verified" badge on GitHub. Instead,
verification is achieved by replaying eligible commits through the GitHub API:

- **Kargo-signed commits** are replayed through the GitHub API and appear as
  "Verified" on GitHub. The original GPG signature is replaced by GitHub's
  verification.
- **Unsigned commits** (or commits signed by a different key) preserve their
  original author and committer and are _not_ marked as "Verified".
- **Commits with bad or revoked signatures** cause the step to fail with a
  terminal error, since they may indicate tampering.

**When no signing key is configured**, all commits preserve their original
author and committer. None are marked as "Verified" on GitHub.

:::note

To produce Kargo-signed commits that this step will verify, use the
[`git-commit`](git-commit.md) step while the controller has a signing key
configured. The `git-commit` step automatically GPG-signs commits when a
signing key is available.

:::

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
