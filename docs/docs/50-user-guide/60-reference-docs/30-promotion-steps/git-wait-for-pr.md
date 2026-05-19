---
sidebar_label: git-wait-for-pr
description: Waits for a specified open pull request to be merged or closed.
---

# `git-wait-for-pr`

`git-wait-for-pr` waits for a specified open pull request to be merged or
closed. This step commonly follows a [`git-open-pr` step](git-open-pr.md)
and is commonly followed by an `argocd-update` step.

:::tip[Accelerate with webhooks]

By default, Kargo polls the Git provider every few minutes to check whether the
PR has been merged or closed. If you configure a webhook receiver that supports
PR/MR closed events
([Azure DevOps](../80-webhook-receivers/azure/index.md),
[Bitbucket](../80-webhook-receivers/bitbucket/index.md),
[GitHub](../80-webhook-receivers/github/index.md),
[GitLab](../80-webhook-receivers/gitlab/index.md),
[Gitea](../80-webhook-receivers/gitea/index.md)),
Kargo detects PR changes near-instantly. The polling fallback remains active for
reliability. See [Webhook Receivers](../80-webhook-receivers/index.md) for setup
instructions.

:::

## Credentials

Git steps are utilizing the [repository credentials](../../50-security/30-managing-secrets.md#repository-credentials)
system to access the git repos.

## Configuration

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `repoURL` | `string` | Y | The URL of a remote Git repository. **Deprecated:** Support for SSH URLs (`ssh://` and SCP-style `git@host:path`) is deprecated as of v1.10.0 and will be removed in v1.13.0. Use HTTPS URLs instead. |
| `provider` | `string` | N | The name of the Git provider to use. Currently `azure`, `bitbucket`, `bitbucket-datacenter`, `gitea`, `github`, and `gitlab` are supported. Kargo will try to infer the provider if it is not explicitly specified. |
| `insecureSkipTLSVerify` | `boolean` | N | Indicates whether to bypass TLS certificate verification when interfacing with the Git provider. Setting this to `true` is highly discouraged in production. |
| `prNumber` | `integer` | Y | The pull request number to wait for. |

## Output

| Name | Type | Description |
|------|------|-------------|
| `commit` | `string` | The ID (SHA) of the new commit at the head of the target branch after merge. Typically, a subsequent [`argocd-update` step](argocd-update.md) will reference this output to learn the ID of the commit that an applicable Argo CD `ApplicationSource` should be observably synced to under healthy conditions. |
| `pr` | `object` | An object containing details about the pull request being monitored. |
| `pr.id` | `number` | The numeric identifier of the pull request. |
| `pr.url` | `string` | The URL of the pull request. |
| `pr.open` | `boolean` | Whether the pull request is still open. |
| `pr.merged` | `boolean` | Whether the pull request has been merged. |

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
    prNumber: ${{ outputs['open-pr'].pr.id }}
```
