---
sidebar_label: git-open-pr
description: Opens a pull request in a specified remote repository using specified source and target branches.
---

# `git-open-pr`

`git-open-pr` opens a pull request in a specified remote repository using
specified source and target branches. This step is often used after a
[`git-push` step](git-push.md) and is commonly followed by a
[`git-wait-for-pr` step](git-wait-for-pr.md).

At present, this feature only supports GitHub, Gitea, Azure DevOps, and
GitLab pull/merge requests.

## Configuration

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `repoURL` | `string` | Y | The URL of a remote Git repository. |
| `provider` | `string` | N | The name of the Git provider to use. Currently only `github`, `gitlab`, `azure`, and `gitea` are supported. Kargo will try to infer the provider if it is not explicitly specified. |
| `insecureSkipTLSVerify` | `boolean` | N | Indicates whether to bypass TLS certificate verification when interfacing with the Git provider. Setting this to `true` is highly discouraged in production. |
| `sourceBranch` | `string` | Y | Specifies the source branch for the pull request. |
| `targetBranch` | `string` | N | The branch to which the changes should be merged. |
| `createTargetBranch` | `boolean` | N | Indicates whether a new, empty orphaned branch should be created and pushed to the remote if the target branch does not already exist there. Default is `false`. |
| `title` | `string` | N | The title for the pull request. Kargo generates a title based on the commit messages if it is not explicitly specified. |
| `labels` | `[]string` | N | Labels to add to the pull request. |

## Output

| Name | Type | Description |
|------|------|-------------|
| `prNumber` | `number` | The numeric identifier of the pull request opened by this step. Typically, a subsequent [`git-wait-for-pr` step](git-wait-for-pr.md) will reference this output to learn what pull request to monitor. |

## Examples

### Common Usage

The following example demonstrates a common use case for `git-open-pr`. It
follows a [`git-push` step](git-push.md) that has pushed changes to a remote
repository to a branch with a generated name. The `git-open-pr` step then
opens a pull request to merge the changes from the source branch which was
created by the `git-push` step into the `stage/${{ ctx.stage }}` branch.

This is a common pattern when implementing GitOps-based promotion workflows,
where changes are first pushed to an intermediate branch and then merged into
a stage-specific branch through a pull request.

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

### Custom Title and Labels

The following example demonstrates how to specify a custom title and labels for
the pull request opened by `git-open-pr`. After pushing changes to a generated
branch, the `git-open-pr` step creates a pull request with a title that
references the current stage (`Deploy to ${{ ctx.stage }}`) and adds two
labels: "infra" and "needs-review".

This is useful when you want to provide more context about the changes being
proposed or need to integrate with existing PR review workflows that rely on
specific labels for automation or filtering.

```yaml

steps:
# Clone, prepare the contents of ./out, commit, etc...
- uses: git-push
  as: push
  config:
    path: ./out
    generateTargetBranch: true
- uses: git-open-pr
  config:
    repoURL: https://github.com/example/repo.git
    sourceBranch: ${{ outputs.push.branch }}
    targetBranch: stage/${{ ctx.stage }}
    title: Deploy to ${{ ctx.stage }}
    labels: ["infra", "needs-review"]
# Wait for the PR to be merged or closed...
```

### Custom Git Provider

The following example demonstrates how to specify a custom Git provider for
`git-open-pr`. This is useful when the provider cannot be inferred from the
`repoURL`. For example, if the repository is hosted on a self-hosted GitLab
instance, the provider must be specified as `gitlab`.

```yaml
steps:
# Clone, push, prepare the contents of ./out, commit, etc...
- uses: git-open-pr
  config:
    repoURL: https://gitlab.example.com/example/repo.git
    provider: gitlab
    # Additional configuration...
```
