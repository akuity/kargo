---
sidebar_label: git-commit
description: Commits all changes in a working tree to its checked out branch.
---

# `git-commit`

`git-commit` commits all changes in a working tree to its checked out branch.
This step is often used after previous steps have put the working tree into the
desired state and is commonly followed by a [`git-push` step](git-push.md).

## Configuration

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `path` | `string` | Y | Path to a Git working tree containing changes to be committed. This path is relative to the temporary workspace that Kargo provisions for use by the promotion process. |
| `message` | `string` | N | The commit message. Mutually exclusive with `messageFromSteps`. |
| `messageFromSteps` | `[]string` | N | References the `commitMessage` output of previous steps. When one or more are specified, the commit message will be constructed by concatenating the messages from individual steps. Mutually exclusive with `message`. |
| `author` | `[]object` | N | Optionally provider authorship information for the commit. |
| `author.name` | `string` | N | The committer's name. |
| `author.email` | `string` | N | The committer's email address. |

## Output

| Name | Type | Description |
|------|------|-------------|
| `commit` | `string` | The ID (SHA) of the commit created by this step. If the step short-circuited and did not create a new commit because there were no differences from the current head of the branch, this value will be the ID of the existing commit at the head of the branch instead. Typically, a subsequent [`argocd-update`](argocd-update.md) step will reference this output to learn the ID of the commit that an applicable Argo CD `ApplicationSource` should be observably synced to under healthy conditions. |

## Examples

```yaml
vars:
- name: gitRepo
  value: https://github.com/example/repo.git
steps:
- uses: git-clone
  config:
    repoURL: ${{ vars.gitRepo }}
    checkout:
    - commit: ${{ commitFrom(vars.gitRepo).ID }}
      path: ./src
    - branch: stage/${{ ctx.stage }}
      create: true
      path: ./out
- uses: git-clear
  config:
    path: ./out
- uses: kustomize-set-image
  as: update-image
  config:
    images:
    - image: my/image
- uses: kustomize-build
  config:
    path: ./src/stages/${{ ctx.stage }}
    outPath: ./out
- uses: git-commit
  config:
    path: ./out
    messageFromSteps:
    - update-image
# Push, etc...
```
