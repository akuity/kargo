---
sidebar_title: delete
description: Deletes a file or directory.
---

# `delete`

`delete` deletes one or more files or directories.

:::note

If you need to delete the entire contents of a Git working tree, consider using
the [`git-clear` step](git-clear.md) instead.

:::

## Configuration

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `path` | `string` | N | Path to the file or directory to delete. Mutually exclusive with `paths`. Exactly one of `path` or `paths` must be specified. |
| `paths` | `string[]` | N | List of paths to files or directories to delete. Mutually exclusive with `path`. |
| `pathsAreGlobs` | `bool` | N | Treats `path` or `paths` as glob patterns (instead of literal paths). Defaults to `false`. |
| `strict` | `bool` | N | Causes the directive to fail if a path does not exist or a glob pattern matches nothing. Defaults to `false`. |

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
    - commit: ${{ commitFrom(vars.gitRepo).ID }}
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

### Deleting Multiple Paths

Use `paths` to delete several files or directories in a single step.

```yaml
- uses: delete
  config:
    paths:
    - ./out/unwanted/file
    - ./out/tmp
    - ./out/build.log
```

### Deleting With Glob Patterns

Set `pathsAreGlobs: true` to expand `path` or `paths` as glob patterns.

```yaml
- uses: delete
  config:
    paths:
    - ./out/**/*.tmp
    - ./out/build/
    pathsAreGlobs: true
    strict: true
```

This will delete all `*.tmp` files in the `./out` directory (and recursively in subdirectories) and the `./out/build` directory.
