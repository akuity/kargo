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
| `message` | `string` | Y | The commit message. |
| `author` | `[]object` | N | Optional provider authorship information for the commit. |
| `author.name` | `string` | Y | The committer's name. |
| `author.email` | `string` | Y | The committer's email address. |
| `author.signingKey` | `string` | N | The GPG signing key for the author. This field is optional. |

## Output

| Name | Type | Description |
|------|------|-------------|
| `commit` | `string` | The ID (SHA) of the commit created by this step. If the step short-circuited and did not create a new commit because there were no differences from the current head of the branch, this value will be the ID of the existing commit at the head of the branch instead. Typically, a subsequent [`argocd-update`](argocd-update.md) step will reference this output to learn the ID of the commit that an applicable Argo CD `ApplicationSource` should be observably synced to under healthy conditions. |

## Examples

### Common Usage

In this example, the working tree is prepared by previous steps and then committed
with a message from the [`kustomize-set-image` step](kustomize-set-image.md) that
updated the `kustomization.yaml` file, summarizing the changes made.

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
    message: ${{ outputs['update-image'].commitMessage }}
# Push, etc...
```

### Composed Commit Message

The `message` field can be used to compose a commit message and enrich it with
contextual information. In this example, the commit message is prefixed with the
current stage using the
[pre-defined `ctx.stage` variable](../40-expressions.md#pre-defined-variables).

:::tip
The `message` field supports multi-line strings. Use `|` to indicate a block
scalar and preserve newlines.

This allows for multi-line commits, which can be useful for providing detailed
commit messages when several changes are being committed together.
:::

```yaml
steps:
# Update Kustomize manifests, etc...
- uses: git-commit
  config:
    path: ./out
    message: |
      ${{ ctx.stage }}: ${{ outputs['update-image'].commitMessage }}
```

### Commit with Custom Author

The `author` field can be used to specify the committer's name and email address
for the commit. This can be useful when the committer's identity should be
different from Kargo's default identity.

```yaml
steps:
# Update Kustomize manifests, etc...
- uses: git-commit
  config:
    path: ./out
    message: ${{ outputs['update-image'].commitMessage }}
    author:
      name: Kargo
      email: kargo@example.com
```
