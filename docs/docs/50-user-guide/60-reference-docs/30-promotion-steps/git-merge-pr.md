---
sidebar_label: git-merge-pr
description: Merges an open pull request.
---

# `git-merge-pr`

<span class="tag beta"></span>

`git-merge-pr` merges an open pull request. This step commonly follows a
[`git-open-pr`](git-open-pr.md) step.

:::caution
This step only executes synchronous merges. It can neither initiate an
asynchronous merge by placing a PR on a merge queue (or similar), nor can it
recognize when an open PR is already _in_ a merge queue (having been placed
there by someone or something else), and thus cannot wait for an aynchronous
merge in-progress to complete.

If a repository _enforces_ merge queues, the behavior when attempting a
synchronous merge depends permissions.

- GitHub: If a merge queue is required, the PR will either be merged directly if the
  token has bypass permissions, or it will fail with an error if it does not.
- GitLab: This step will always merge directly without using merge trains.
:::

## Configuration

| Name                    | Type      | Required | Description                                                                                                                                                                                                    |
| ----------------------- | --------- | -------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `repoURL`               | `string`  | Y        | The URL of a remote Git repository.                                                                                                                                                                            |
| `provider`              | `string`  | N        | The name of the Git provider to use. Currently `azure`, `bitbucket`, `gitea`, `github`, and `gitlab` are supported. Kargo will try to infer the provider if it is not explicitly specified.                    |
| `insecureSkipTLSVerify` | `boolean` | N        | Indicates whether to bypass TLS certificate verification when interfacing with the Git provider. Setting this to `true` is highly discouraged in production.                                                   |
| `prNumber`              | `integer` | Y        | The pull request number to merge.                                                                                                                                                                              |
| `wait`                  | `boolean` | N        | If `true`, the step will return a running status instead of failing when the PR is not yet mergeable. The merge will be retried on the next reconciliation until it succeeds or times out. Default is `false`. |

:::warning
The `wait` option is unreliable for repositories hosted by Bitbucket due to API limitations.
:::

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
request is not immediately ready to merge (e.g., due to pending CI checks, required
reviews, or temporary conflicts), the step will return a running status and Kargo
will retry it on the next reconciliation.

```yaml
steps:
- uses: git-merge-pr
  config:
    repoURL: https://github.com/example/repo.git
    prNumber: 42
    wait: true
```

:::note
If the repository requires using a merge queue and the token lacks bypass permissions,
the step will fail regardless of the `wait` setting, as the step does not add PRs to
merge queues.
:::
