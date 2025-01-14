---
sidebar_label: git-wait-for-pr
description: Waits for a specified open pull request to be merged or closed.
---

# `git-wait-for-pr`

`git-wait-for-pr` waits for a specified open pull request to be merged or
closed. This step commonly follows a [`git-open-pr` step](18-git-open-pr.md)
and is commonly followed by an `argocd-update` step.

## Configuration

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `repoURL` | `string` | Y | The URL of a remote Git repository. |
| `provider` | `string` | N | The name of the Git provider to use. Currently only `github` and `gitlab` are supported. Kargo will try to infer the provider if it is not explicitly specified.  |
| `insecureSkipTLSVerify` | `boolean` | N | Indicates whether to bypass TLS certificate verification when interfacing with the Git provider. Setting this to `true` is highly discouraged in production. |
| `prNumber` | `string` | N | The number of the pull request to wait for. Mutually exclusive with `prNumberFromStep`. |
| `prNumberFromStep` | `string` | N | References the `prNumber` output from a previous step. Mutually exclusive with `prNumber`.<br/><br/>__Deprecated: Use `prNumber` with an expression instead. Will be removed in v1.3.0.__ |

## Output

| Name | Type | Description |
|------|------|-------------|
| `commit` | `string` | The ID (SHA) of the new commit at the head of the target branch after merge. Typically, a subsequent [`argocd-update` step](50-argocd-update.md) will reference this output to learn the ID of the commit that an applicable Argo CD `ApplicationSource` should be observably synced to under healthy conditions. |

## Examples

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
