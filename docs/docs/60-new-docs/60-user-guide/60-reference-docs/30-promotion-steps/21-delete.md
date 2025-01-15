---
sidebar_title: delete
description: Deletes a file or directory.
---

# `delete`

`delete` deletes a file or directory.

## Configuration

| Name      | Type | Required | Description                              |
|-----------|------|----------|------------------------------------------|
| `path`    | `string` | Y | Path to the file or directory to delete. |

## Examples

One common usage of this step is to remove intermediate files produced by the
promotion process prior to a `git-commit` step:

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
