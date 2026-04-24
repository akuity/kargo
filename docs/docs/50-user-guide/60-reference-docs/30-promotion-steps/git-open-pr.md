---
sidebar_label: git-open-pr
description: Opens a pull request in a specified remote repository using specified source and target branches.
---

# `git-open-pr`

`git-open-pr` opens a pull request in a specified remote repository using
specified source and target branches. This step is often used after a
[`git-push` step](git-push.md) and is commonly followed by a
[`git-wait-for-pr` step](git-wait-for-pr.md).

## Credentials

Git steps are utilizing the [repository credentials](../../50-security/30-managing-secrets.md#repository-credentials)
system to access the git repos.

## Configuration

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `repoURL` | `string` | Y | The URL of a remote Git repository. **Deprecated:** Support for SSH URLs (`ssh://` and SCP-style `git@host:path`) is deprecated as of v1.10.0 and will be removed in v1.13.0. Use HTTPS URLs instead. |
| `provider` | `string` | N | The name of the Git provider to use. Currently `azure`, `bitbucket`, `bitbucket-datacenter`, `gitea`, `github`, and `gitlab` are supported. Kargo will try to infer the provider if it is not explicitly specified. |
| `insecureSkipTLSVerify` | `boolean` | N | Indicates whether to bypass TLS certificate verification when interfacing with the Git provider. Setting this to `true` is highly discouraged in production. |
| `sourceBranch` | `string` | Y | Specifies the source branch for the pull request. |
| `targetBranch` | `string` | N | The branch to which the changes should be merged. |
| `createTargetBranch` | `boolean` | N | **Deprecated**. Is a no-op if set. Will be removed in a future release.|
| `title` | `string` | N | The title for the pull request. Kargo generates a title based on the commit messages if it is not explicitly specified. |
| `description` | `string` | N | The description for the pull request. |
| `labels` | `[]string` | N | Labels to add to the pull request. |

## Output

| Name | Type | Description |
|------|------|-------------|
| `pr` | `object` | An object containing details about the pull request. |
| `pr.id` | `number` | The numeric identifier of the pull request opened by this step. Typically, a subsequent [`git-wait-for-pr` step](git-wait-for-pr.md) will reference this output to learn what pull request to monitor. |
| `pr.url` | `string` | The URL of the pull request. |

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

:::note

The `git-open-pr` step will fail if the `targetBranch` doesn't exist.

:::

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
  as: open-pr
  config:
    repoURL: https://github.com/example/repo.git
    sourceBranch: ${{ outputs.push.branch }}
    targetBranch: stage/${{ ctx.stage }}
    title: Deploy to ${{ ctx.stage }}
    labels: ["infra", "needs-review"]
- if: ${{ status('open-pr') != 'Skipped' }}
  uses: git-wait-for-pr
  as: wait-for-pr
  config:
    repoURL: https://github.com/example/repo.git
    prNumber: ${{ outputs['open-pr'].pr.id }}
```

### Skipped

The following example conditionally runs the 
[`git-wait-for-pr` step](git-wait-for-pr.md) based on whether or not the 
`git-open-pr` step was skipped. If there are no changes between the 
`sourceBranch` and `targetBranch`, the `git-open-pr` step will be skipped. The 
[`status`](../40-expressions.md#statusstepalias) expression function can be used 
by subsequent steps to determine if a preceding step was skipped.

```yaml
- uses: git-push
  as: push
  config:
    path: ./out
    generateTargetBranch: true
  - uses: git-open-pr
    as: open-pr
    config:
      repoURL: https://github.com/example/repo.git
      sourceBranch: ${{ outputs.push.branch }}
      targetBranch: stage/${{ ctx.stage }}
  - if: ${{ status('open-pr') != 'Skipped' }}
    uses: git-wait-for-pr
    as: wait-for-pr
    config:
      repoURL: https://github.com/example/repo.git
      prNumber: ${{ outputs['open-pr'].pr.id }}
```