---
sidebar_label: git-push
description: Pushes the committed changes in a specified working tree to a specified branch in the remote repository.
---

# `git-push`

`git-push` pushes the committed changes in a specified working tree to a
specified branch in the remote repository. This step typically follows a
[`git-commit` step](git-commit.md) and is often followed by a
[`git-open-pr` step](git-open-pr.md).

This step also implements its own, internal retry logic. If a push fails, with
the cause determined to be the presence of new commits in the remote branch that
are not present in the local branch, the step will attempt to rebase before
retrying the push. Any merge conflict requiring manual resolution will
immediately halt further attempts.

:::info
This step's internal retry logic is helpful in scenarios when concurrent
Promotions to multiple Stages may all write to the same branch of the same
repository.

Because conflicts requiring manual resolution will halt further attempts, it is
recommended to design your Promotion processes such that Promotions to multiple
Stages that write to the same branch do not write to the same files.
:::

## Configuration

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `path` | `string` | Y | Path to a Git working tree containing committed changes. |
| `targetBranch` | `string` | N | The branch to push to in the remote repository. Mutually exclusive with `generateTargetBranch=true`. If neither of these is provided, the target branch will be the same as the branch currently checked out in the working tree. |
| `maxAttempts` | `int32` | N | The maximum number of attempts to make when pushing to the remote repository. Default is 50. |
| `generateTargetBranch` | `boolean` | N | Whether to push to a remote branch named like `kargo/promotion/<promotionName>`. If such a branch does not already exist, it will be created. A value of 'true' is mutually exclusive with `targetBranch`. If neither of these is provided, the target branch will be the currently checked out branch. This option is useful when a subsequent promotion step will open a pull request against a Stage-specific branch. In such a case, the generated target branch pushed to by the `git-push` step can later be utilized as the source branch of the pull request. |

## Output

| Name | Type | Description |
|------|------|-------------|
| `branch` | `string` | The name of the remote branch pushed to by this step. This is especially useful when the `generateTargetBranch=true` option has been used, in which case a subsequent [`git-open-pr`](git-open-pr.md) will typically reference this output to learn what branch to use as the head branch of a new pull request. |
| `commit` | `string` | The ID (SHA) of the commit pushed by this step. |
| `commitURL` | `string` | The URL to the commit that was pushed on the hosting provider e.g. Github. |


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
