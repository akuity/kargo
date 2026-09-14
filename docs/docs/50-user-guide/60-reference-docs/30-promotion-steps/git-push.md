---
sidebar_label: git-push
description: Pushes the committed changes in a specified working tree to a specified branch in the remote repository.
---

# `git-push`

`git-push` pushes committed changes or new tags from a specified working tree to
the remote repository. This step typically follows a
[`git-commit` step](git-commit.md) and/or [`git-tag` step](git-tag.md) and is
often followed by a [`git-open-pr` step](git-open-pr.md).

This step also implements its own, internal retry logic. If a push fails, with
the cause determined to be the presence of new commits in the remote branch that
are not present in the local branch, the step will integrate the remote changes
and retry the push. Any merge conflict requiring manual resolution will
immediately halt further attempts.

How remote changes are integrated is controlled by the system-level __push
integration policy__, which is configured by a Kargo operator. The available
policies are:

- `AlwaysRebase` — Unconditionally rebase. Simplest, but may re-sign or strip
  commit signatures.

- `RebaseOrMerge` — Rebase when a signature-trust analysis determines it is
  safe; merge otherwise. This preserves linear history when possible without
  undermining trust.

- `RebaseOrFail` — Rebase when the signature-trust analysis determines it is
  safe; fail the step otherwise.

- `AlwaysMerge` — Unconditionally create a merge commit. Most conservative.

:::caution

The current default policy is `AlwaysRebase`.

Starting with v1.12.0, the default will change to `RebaseOrMerge`.

:::

When the policy evaluates rebase safety (`RebaseOrMerge` and `RebaseOrFail`),
the decision to use rebase or not is based on the GPG signature status of the
local commits that would be replayed:

- If all local commits are __signed by a trusted key__ and signing was enabled
  at the time the repository was cloned, a __rebase__ is performed. The
  replacement commits are re-signed by Kargo.

- If all local commits are __unsigned__ and signing was _not_ enabled at the
  time the repository was cloned, a __rebase__ is performed. The replacement
  commits remain unsigned.

- In all other cases, the policy's fallback behavior applies (merge or fail).

A "trusted key" is one that was imported with ultimate trust when the repository
was cloned, either through explicit configuration of the
[`git-clone`](git-clone.md) step or via fallback on a system-level signing key
configured by a Kargo admin.

:::info

For more information on configuring the push integration policy, see the
[operator guide](../../../40-operator-guide/20-advanced-installation/30-common-configurations.md#push-integration-policy).

:::

:::info

This step's internal retry logic is helpful in scenarios when concurrent
Promotions to multiple Stages may all write to the same branch of the same
repository.

Because conflicts requiring manual resolution will halt further attempts, it is
recommended to design your Promotion processes such that Promotions to multiple
Stages that write to the same branch do not write to the same files.

:::

:::note

For a tag push, there is no pull/rebase retry loop.

:::

:::info

If you authenticate to GitHub using a GitHub App, you may want to consider using
[`github-push`](github-push.md) instead.

:::

## Credentials

This step utilizes the [repository credentials](../../50-security/30-managing-secrets.md#repository-credentials)
system to access Git repositories.

## Configuration

| Name | Type | Required | Description |
| ---- | ---- | -------- | ----------- |
| `path` | `string` | Y | Path to a Git working tree containing committed changes. |
| `targetBranch` | `string` | N | The branch to push to in the remote repository. Mutually exclusive with `generateTargetBranch=true` and `tag`. If none of these are provided, the target branch will be the same as the branch currently checked out in the working tree. |
| `maxAttempts` | `int32` | N | The maximum number of attempts to make when pushing to the remote repository. Default is 10. |
| `generateTargetBranch` | `boolean` | N | Whether to push to a remote branch named like `kargo/promotion/<promotionName>`. If such a branch does not already exist, it will be created. A value of `true` is mutually exclusive with `targetBranch` and `tag`. If none of these are provided, the target branch will be the currently checked out branch. This option is useful when a subsequent promotion step will open a pull request against a Stage-specific branch. In such a case, the generated target branch pushed to by the `git-push` step can later be utilized as the source branch of the pull request. |
| `tag` | `string` | N | An tag to push to the remote repository. Mutually exclusive with `generateTargetBranch` and `targetBranch`. |
| `force` | `boolean` | N | Whether to force push to the target branch, overwriting any existing history. This is useful for scenarios where you want to completely replace the branch content (e.g., pushing rendered manifests that don't depend on previous state). **Use with caution** as this will overwrite any commits that exist on the remote branch but not in your local branch. Default is `false`. A value of `true` is mutually exclusive with `tag`. |
| `provider` | `string` | N | The name of the Git provider to use. Currently `azure`, `bitbucket`, `bitbucket-datacenter`, `gitea`, `github`, and `gitlab` are supported. Kargo will try to infer the provider if it is not explicitly specified. This setting does not affect the push operation but helps generate the correct [`commitURL` output](#output) when working with repositories where the provider cannot be automatically determined, such as self-hosted instances. |

## Output

| Name | Type | Description |
| ---- | ---- | ----------- |
| `branch` | `string` | The name of the remote branch pushed to by this step. This is especially useful when the `generateTargetBranch=true` option has been used, in which case a subsequent [`git-open-pr`](git-open-pr.md) will typically reference this output to learn what branch to use as the head branch of a new pull request. |
| `commit` | `string` | The ID (SHA) of the commit pushed by this step. |
| `commitURL` | `string` | The URL of the commit that was pushed to the remote repository. |

## Examples

### Common Usage

In this example, changes prepared in a working directory are committed and
pushed to the same branch that was checked out. The `git-push` step takes the
path to the working directory containing the committed changes and pushes them
to the remote repository.

This is the most basic and common pattern for updating a branch with new changes
during a promotion process.

```yaml
steps:
# Clone, prepare the contents of ./out, etc...
- uses: git-commit
  config:
    path: ./out
    message: rendered updated manifests
- uses: git-push
  config:
    path: ./out
```

### For Use With a Pull Request

In this example, changes are pushed to a generated branch name that follows
the pattern `kargo/promotion/<promotionName>`. By setting
`generateTargetBranch: true`, the step creates a unique branch name that can
be referenced by subsequent steps.

This is commonly used as part of a pull request workflow, where changes are
first pushed to an intermediate branch before being proposed as a pull request.
The step's output includes the generated branch name, which can then be used by
a subsequent [`git-open-pr` step](git-open-pr.md).

```yaml
steps:
# Clone, prepare the contents of ./out, etc...
- uses: git-commit
  config:
    path: ./out
    message: rendered updated manifests
- uses: git-push
  as: push
  config:
    path: ./out
    generateTargetBranch: true
# Open a PR and wait for it to be merged or closed...
```

### Pushing Tags

In this example, a new tag is pushed to the remote repository.

```yaml
# Create a new tag
- uses: git-tag
  config:
    path: ./out
    tag: v1.0.0
- uses: git-push
  config:
    path: ./out
    tag: v1.0.0
```

:::caution

If the specified tag already exists in the remote repository, the `git-push`
step will fail.

:::
