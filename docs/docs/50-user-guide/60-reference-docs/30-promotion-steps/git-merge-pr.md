---
sidebar_label: git-merge-pr
description: Merges an open pull request.
---

# `git-merge-pr`

<span class="tag beta"></span>

`git-merge-pr` merges an open pull request. This step commonly follows a
[`git-open-pr`](git-open-pr.md) step.

:::important

This step only executes synchronous merges. It can neither initiate an
asynchronous merge by placing a PR on a merge queue (or similar), nor can it
recognize when an open PR is already _in_ a merge queue (having been placed
there by someone or something else), and thus cannot wait for an asynchronous
merge in-progress to complete.

:::

:::caution

**GitHub** repositories can be configured with branch protection rules that
require PRs to be merged via a merge queue. When such a rule is in place, the
results of the `git-merge-pr` step attempting a synchronous merge will depend
upon permissions. With sufficient permissions to bypass branch protection rules,
the merge queue will be bypassed. Without such permissions, the step's attempt
to merge will fail.

:::

## Credentials

Git steps are utilizing the [repository credentials](../../50-security/30-managing-secrets.md#repository-credentials)
system to access the git repos.

## Configuration

| Name                    | Type      | Required | Description                                                                                                                                                                                                    |
| ----------------------- | --------- | -------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `repoURL`               | `string`  | Y        | The URL of a remote Git repository. **Deprecated:** Support for SSH URLs (`ssh://` and SCP-style `git@host:path`) is deprecated as of v1.10.0 and will be removed in v1.13.0. Use HTTPS URLs instead.          |
| `provider`              | `string`  | N        | The name of the Git provider to use. Currently `azure`, `bitbucket`, `gitea`, `github`, and `gitlab` are supported. Kargo will try to infer the provider if it is not explicitly specified.                    |
| `insecureSkipTLSVerify` | `boolean` | N        | Indicates whether to bypass TLS certificate verification when interfacing with the Git provider. Setting this to `true` is highly discouraged in production.                                                   |
| `prNumber`              | `integer` | Y        | The pull request number to merge.                                                                                                                                                                              |
| `mergeMethod`           | `string`  | N        | The merge method to use when merging the pull request. The supported methods are provider-specific; refer to the [Merge Method](#merge-method) section. |
| `wait`                  | `boolean` | N        | If `true`, the step will return a running status instead of failing when the PR is not yet mergeable. The merge will be retried on the next reconciliation until it succeeds or times out. Default is `false`. |

:::warning

The `wait` option is unreliable for repositories hosted by Bitbucket due to API limitations.

:::

### Merge Method

The table below documents the supported merge methods/strategies for each of the
currently supported Git hosting providers.

| Provider | Supported Methods | Default |
| -------- | ----------------- | ------- |
| Azure | <ul><li>`noFastForward`</li><li>`rebase`</li><li>`rebaseMerge`</li><li>`squash`</li></ul> | First allowed strategy per the target branch's merge type policy; merge commit if no policy is configured |
| BitBucket | Client does not yet support specifying a merge strategy | The repository's configured default merge strategy |
| Gitea | <ul><li>`fast-forward-only`</li><li>`manually-merged`</li><li>`merge`</li><li>`rebase`</li><li>`rebase-merge`</li><li>`squash`</li></ul> | `merge` |
| GitHub | <ul><li>`merge`</li><li>`rebase`</li><li>`squash`</li></ul> | `merge` |
| GitLab | <ul><li>`merge`</li><li>`squash`</li></ul> | Defers to the merge request and project-level squash settings |

## Output

| Name     | Type     | Description                                                                                                                                                                                                                                                                                                             |
| -------- | -------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `commit` | `string` | The ID (SHA) of the merge commit created after successfully merging the pull request. Typically, a subsequent [`argocd-update`](argocd-update.md) step will reference this output to learn the ID of the commit that an applicable Argo CD `ApplicationSource` should be observably synced to under healthy conditions. |

## Examples

### Basic Usage

In this example, a pull request is merged immediately if it's ready. If the pull
request is not ready to merge (e.g., due to pending checks or conflicts), the step
will fail.

```yaml
steps:
- uses: git-merge-pr
  config:
    repoURL: https://github.com/example/repo.git
    prNumber: 42
```

### Merge with Wait

This example demonstrates merging a pull request with waiting enabled. If the pull
request is not yet mergeable for any reason, the step will return a running
status and Kargo will retry it on the next reconciliation.

```yaml
steps:
- uses: git-merge-pr
  config:
    repoURL: https://github.com/example/repo.git
    prNumber: 42
    wait: true
```

### Specifying a Merge Method

This example demonstrates merging a pull request with a specific merge method.
Refer to the [Merge Method](#merge-method) section for supported values per
provider.

```yaml
steps:
- uses: git-merge-pr
  config:
    repoURL: https://github.com/example/repo.git
    prNumber: 42
    mergeMethod: squash
```
