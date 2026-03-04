---
sidebar_label: github-sign
description: Signs commits via the GitHub API by replaying them through a GitHub App.
---

# `github-sign`

`github-sign` replays commits in a specified revision range as GitHub-signed
commits via the GitHub REST API. This step is designed for use with GitHub App
credentials, producing verified commit signatures.

The typical workflow uses a staging branch pattern:

1. A [`git-push`](git-push.md) step with `generateTargetBranch: true` pushes
   unsigned commits to a temporary branch (e.g. `kargo/promotion/<name>`)
2. `github-sign` reads the unsigned commits, replays them as signed commits
   with the target branch tip as their parent, and fast-forward updates the
   target branch

This approach is compatible with branch protection rules that prevent force
pushes, because the target branch is updated via a fast-forward.

:::info

This step requires a GitHub App installation token configured as a Git
credential secret. Commits created via the GitHub API are automatically signed
and appear as "Verified" on GitHub.

:::

## Configuration

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `repoURL` | `string` | Y | The URL of the GitHub repository. |
| `targetBranch` | `string` | Y | The branch whose ref will be updated to point to the final signed commit. |
| `base` | `string` | Y | The exclusive base of the revision range (commit SHA). Typically set from a prior `git-clone` step's output, e.g. `${{ outputs.clone.commits.main }}`. |
| `head` | `string` | Y | The inclusive head of the revision range (commit SHA). All revisions in the range `base..head` will be replayed as GitHub-signed commits. Typically set from a prior `git-push` step's output, e.g. `${{ outputs.push.commit }}`. |
| `insecureSkipTLSVerify` | `boolean` | N | Whether to skip TLS verification when communicating with the GitHub API. Default is `false`. |

## Output

| Name | Type | Description |
|------|------|-------------|
| `commit` | `string` | The SHA of the final signed commit. |
| `commitURL` | `string` | The URL of the final signed commit. |
| `branch` | `string` | The name of the target branch that was updated. |

## Examples

### Staging Branch Pattern

In this example, unsigned commits are first pushed to a generated staging
branch, then signed and fast-forwarded onto the target branch. This is the
recommended pattern because it works with branch protection rules that
prevent force pushes.

```yaml
steps:
- uses: git-clone
  as: clone
  config:
    repoURL: https://github.com/example/repo
    checkout:
    - branch: main
      path: ./out
# Prepare the contents of ./out...
- uses: git-commit
  config:
    path: ./out
    message: rendered updated manifests
- uses: git-push
  as: push
  config:
    path: ./out
    generateTargetBranch: true
- uses: github-sign
  config:
    repoURL: https://github.com/example/repo
    targetBranch: main
    base: ${{ outputs.clone.commits.main }}
    head: ${{ outputs.push.commit }}
```
