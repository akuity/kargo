---
sidebar_title: delete
description: Deletes a file or directory.
---

# `delete`

`delete` deletes a file or directory.

:::note
If you need to delete the entire contents of a Git working tree, consider using
the [`git-clear` step](git-clear.md) instead.
:::

## Configuration

| Name      | Type | Required | Description                              |
|-----------|------|----------|------------------------------------------|
| `path` | `string` | Y | Path to the file or directory to delete. |
| `strict` | `bool` | N | Strict will cause the directive to fail if the path does not exist. Defaults to `false`. |

## Examples

### Common Usage

One common usage of this step is to remove intermediate files produced by the
promotion process prior to a [`git-commit` step](git-commit.md). This is useful
when you want to ensure that only the final, desired files are committed to the
repository.

```yaml
vars:
- name: gitRepo
  value: https://github.com/example/repo.git
steps:
- uses: git-clone
  config:
    repoURL: ${{ vars.gitRepo }}
    checkout:
    - commit: ${{ commitFrom(vars.gitRepo) }}
      path: ./src
    - branch: stage/${{ ctx.stage }}
      create: true
      path: ./out

# Steps that produce intermediate files in ./out...

- uses: delete
  config:
    path: ./out/unwanted/file
- uses: git-commit
  config:
    path: ./out
```
