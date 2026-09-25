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
| `provider`              | `string`  | N        | The name of the Git provider to use. Currently `azure`, `bitbucket`, `bitbucket-datacenter`, `gitea`, `github`, and `gitlab` are supported. Kargo will try to infer the provider if it is not explicitly specified.                    |
| `insecureSkipTLSVerify` | `boolean` | N        | Indicates whether to bypass TLS certificate verification when interfacing with the Git provider. Setting this to `true` is highly discouraged in production.                                                   |
| `prNumber`              | `integer` | Y        | The pull request number to merge.                                                                                                                                                                              |
| `mergeMethod`           | `string`  | N        | The merge method to use when merging the pull request. The supported methods are provider-specific; refer to the [Merge Method](#merge-method) section. |
| `wait`                  | `boolean` | N        | If `true`, the step will return a running status instead of failing when the PR is not yet mergeable. The merge will be retried on the next reconciliation until it succeeds or times out. Default is `false`. |
| `pollInterval`          | `string`  | N        | When `wait` is `true`, the suggested interval at which to re-attempt the merge while the PR is not yet mergeable (e.g. `10s`, `1m`). This is only a suggestion: Kargo enforces a lower bound of 10 seconds and may reconcile sooner in response to other events. Defaults to `10s`. |
| `deleteSourceBranch`    | `boolean` | N        | If `true`, the PR's source branch is deleted after the PR has been merged. A failure to delete the branch does not fail the step; it is reported in the step's message instead. See [Deleting the Source Branch](#deleting-the-source-branch). Default is `false`. |

:::warning

The `wait` option is unreliable for repositories hosted by Bitbucket. The
Bitbucket Cloud API does not provide a way to check merge eligibility before
attempting a merge, so Kargo cannot determine in advance whether a PR is blocked
by conflicts, failing checks, or other conditions. As a result, Kargo will
attempt the merge regardless, which may fail unexpectedly.

:::

### Merge Method

The table below documents the supported merge methods/strategies for each of the
currently supported Git hosting providers.

| Provider | Supported Methods | Default |
| -------- | ----------------- | ------- |
| Azure | <ul><li>`noFastForward`</li><li>`rebase`</li><li>`rebaseMerge`</li><li>`squash`</li></ul> | First allowed strategy per the target branch's merge type policy; merge commit if no policy is configured |
| Bitbucket Cloud | <ul><li>`fast_forward`</li><li>`merge_commit`</li><li>`squash`</li></ul> | The repository's configured default merge strategy |
| Bitbucket Data Center | <ul><li>`ff`</li><li>`ff-only`</li><li>`no-ff`</li><li>`rebase-ff-only`</li><li>`rebase-no-ff`</li><li>`squash`</li><li>`squash-ff-only`</li></ul> | The repository's configured default merge strategy |
| Gitea | <ul><li>`fast-forward-only`</li><li>`manually-merged`</li><li>`merge`</li><li>`rebase`</li><li>`rebase-merge`</li><li>`squash`</li></ul> | `merge` |
| GitHub | <ul><li>`merge`</li><li>`rebase`</li><li>`squash`</li></ul> | `merge` |
| GitLab | <ul><li>`merge`</li><li>`squash`</li></ul> | Defers to the merge request and project-level squash settings |

### Deleting the Source Branch

Pipelines that open a PR from a branch generated by the
[`git-push`](git-push.md) step (`kargo/promotion/<promotion-name>`) leave a new
branch behind for every promotion. Setting `deleteSourceBranch` to `true`
deletes the PR's source branch once the PR has been merged, using the same
credentials used to merge it.

Because the merge itself is the step's job, a failure to delete the branch
afterward does not fail the step, and the `commit` output remains available to
subsequent steps. The failure is recorded in the step's message and in the
controller logs. A branch that has already been deleted (for instance, by the
Git hosting provider itself) is not treated as a failure, so the step is safe to
re-run.

:::note

GitHub repositories can be configured to automatically delete head branches
when PRs are merged. That setting does not apply to merges performed with a
GitHub App installation token, which is how Kargo authenticates when configured
with GitHub App credentials. `deleteSourceBranch` deletes the branch explicitly
and works regardless of how Kargo authenticates.

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

### Deleting the Source Branch After Merging

This example merges a PR opened from a branch generated by a preceding
[`git-push`](git-push.md) step and deletes that branch once the merge has
completed.

```yaml
steps:
# Clone, update, commit, and push to a generated branch...
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
    targetBranch: main
- uses: git-merge-pr
  config:
    repoURL: https://github.com/example/repo.git
    prNumber: ${{ outputs['open-pr'].pr.id }}
    wait: true
    deleteSourceBranch: true
```
