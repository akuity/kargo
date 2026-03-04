---
sidebar_label: github-verified-push
description: Pushes committed changes to a GitHub repository as verified (signed) commits using the GitHub REST API.
---

# `github-verified-push`

`github-verified-push` pushes committed changes from a local working tree to a
GitHub repository as **verified (signed) commits**. It is a drop-in replacement
for [`git-push`](git-push.md) when you need commits to carry GitHub's
"Verified" badge.

This step is designed to work with repositories that enforce commit signing via
branch protection rules. Under the hood it:

1. Pushes local commits to a temporary, non-branch staging ref on GitHub
   (invisible in the branch list)
2. Replays each commit via the GitHub REST API, which automatically signs it
   with the GitHub App's identity
3. Fast-forwards the target branch to the final signed commit
4. Deletes the staging ref

Because commits are created through the GitHub API using a GitHub App
installation token, they are automatically marked as verified by GitHub.

:::info
This step requires a **GitHub App installation token** stored as Git
credentials for the repository. The GitHub App must have **Contents: read &
write** permission on the target repository.
:::

## Configuration

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `path` | `string` | Y | Path to a Git working tree containing committed changes. |
| `targetBranch` | `string` | N | The branch to push to in the remote repository. Mutually exclusive with `generateTargetBranch=true`. If neither of these is provided, the target branch will be the same as the branch currently checked out in the working tree. |
| `generateTargetBranch` | `boolean` | N | Whether to push to a remote branch named like `kargo/promotion/<promotionName>`. A value of `true` is mutually exclusive with `targetBranch`. This is useful when a subsequent step will open a pull request. |
| `insecureSkipTLSVerify` | `boolean` | N | Whether to skip TLS verification when communicating with the GitHub API. Default is `false`. Intended for GitHub Enterprise instances with self-signed certificates. |

## Output

| Name | Type | Description |
|------|------|-------------|
| `branch` | `string` | The name of the remote branch pushed to by this step. |
| `commit` | `string` | The ID (SHA) of the final signed commit. |
| `commitURL` | `string` | The URL of the final signed commit on GitHub. |

## Examples

### Common Usage

In this example, changes are committed locally and then pushed to the same
branch as verified commits. This replaces the typical
`git-commit` + `git-push` pattern when signed commits are required.

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
