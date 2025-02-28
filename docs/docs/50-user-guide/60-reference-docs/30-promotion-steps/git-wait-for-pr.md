---
sidebar_label: git-wait-for-pr
description: Waits for a specified open pull request to be merged or closed.
---

# `git-wait-for-pr`

`git-wait-for-pr` waits for a specified open pull request to be merged or
closed. This step commonly follows a [`git-open-pr` step](git-open-pr.md)
and is commonly followed by an `argocd-update` step.

## Configuration

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `repoURL` | `string` | Y | The URL of a remote Git repository. |
| `provider` | `string` | N | The name of the Git provider to use. Currently only `github`, `gitlab`, `azure`, and `gitea` are supported. Kargo will try to infer the provider if it is not explicitly specified. |
| `insecureSkipTLSVerify` | `boolean` | N | Indicates whether to bypass TLS certificate verification when interfacing with the Git provider. Setting this to `true` is highly discouraged in production. |
| `prNumber` | `string` | Y | The number of the pull request to wait for. |

## Output

| Name | Type | Description |
|------|------|-------------|
| `commit` | `string` | The ID (SHA) of the new commit at the head of the target branch after merge. Typically, a subsequent [`argocd-update` step](argocd-update.md) will reference this output to learn the ID of the commit that an applicable Argo CD `ApplicationSource` should be observably synced to under healthy conditions. |

## Examples

### Common Usage

In this example, a complete promotion flow is demonstrated where changes are
pushed to a generated branch, a pull request is opened, and then the process
waits for the pull request to be merged or closed. The `git-wait-for-pr` step
references both the repository URL and the PR number (obtained from the
[`open-pr` step's output](git-open-pr.md#output)) to track the PR's status.

This pattern is common when you want to ensure changes have been properly
reviewed and merged before proceeding with subsequent steps in your promotion
process, such as [updating Argo CD applications](argocd-update.md).

```yaml
steps:
# Clone, prepare the contents of ./out, commit, etc...
- uses: git-push
  as: push
  config:
    path: ./out
    generateTargetBranch: true
- uses: git-open-pr
  as: open-pr
  config:
    repoURL: https://github.com/example/repo.git
    createTargetBranch: true
    sourceBranch: ${{ outputs.push.branch }}
    targetBranch: stage/${{ ctx.stage }}
- uses: git-wait-for-pr
  as: wait-for-pr
  config:
    repoURL: https://github.com/example/repo.git
    prNumber: ${{ outputs['open-pr'].prNumber }}
```
