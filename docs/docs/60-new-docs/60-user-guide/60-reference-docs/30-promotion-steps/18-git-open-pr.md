---
sidebar_label: git-open-pr
description: Opens a pull request in a specified remote repository using specified source and target branches.
---

# `git-open-pr`

`git-open-pr` opens a pull request in a specified remote repository using
specified source and target branches. This step is often used after a
[`git-push` step](16-git-push.md) and is commonly followed by a
[`git-wait-for-pr` step](19-git-wait-for-pr.md).

At present, this feature only supports GitHub pull requests and GitLab merge
requests.

## Configuration

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `repoURL` | `string` | Y | The URL of a remote Git repository. |
| `provider` | `string` | N | The name of the Git provider to use. Currently only `github` and `gitlab` are supported. Kargo will try to infer the provider if it is not explicitly specified.  |
| `insecureSkipTLSVerify` | `boolean` | N | Indicates whether to bypass TLS certificate verification when interfacing with the Git provider. Setting this to `true` is highly discouraged in production. |
| `sourceBranch` | `string` | N | Specifies the source branch for the pull request. Mutually exclusive with `sourceBranchFromStep`. |
| `sourceBranchFromStep` | `string` | N | Indicates the source branch should be determined by the `branch` key in the output of a previous promotion step with the specified alias. Mutually exclusive with `sourceBranch`.<br/><br/>__Deprecated: Use `sourceBranch` with an expression instead. Will be removed in v1.3.0.__  |
| `targetBranch` | `string` | N | The branch to which the changes should be merged. |
| `createTargetBranch` | `boolean` | N | Indicates whether a new, empty orphaned branch should be created and pushed to the remote if the target branch does not already exist there. Default is `false`. |
| `title` | `string` | N | The title for the pull request. Kargo generates a title based on the commit messages if it is not explicitly specified. |
| `labels` | `[]string` | N | Labels to add to the pull request. |

## Output

| Name | Type | Description |
|------|------|-------------|
| `prNumber` | `number` | The numeric identifier of the pull request opened by this step. Typically, a subsequent [`git-wait-for-pr` step](19-git-wait-for-pr.md) will reference this output to learn what pull request to monitor. |

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
# Wait for the PR to be merged or closed...
```
