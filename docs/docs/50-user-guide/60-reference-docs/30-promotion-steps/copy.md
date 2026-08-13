---
sidebar_label: copy
description: Copies files or the contents of entire directories from one specified location to another.
---

# `copy`

`copy` copies files or the contents of entire directories from one specified
location to another.

## Configuration

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `inPath` | `string` | Y | Path to the file or directory to be copied. This path is relative to the temporary workspace that Kargo provisions for use by the promotion process. |
| `outPath` | `string` | Y | Path to the destination. This path is relative to the temporary workspace that Kargo provisions for use by the promotion process. |
| `ignore` | `string` | N | A multiline string of glob patterns to ignore when copying files. It accepts the same syntax as `.gitignore` files. |

Directories are copied recursively. If the destination directory already exists,
its contents are merged with those of the source. Existing destination files are
overwritten.

:::note
Symlinks encountered in the source are skipped, and `.git` directories are
always ignored, regardless of the `ignore` patterns.
:::

## Examples

### Basic Usage

Copy the contents of one directory to another:

```yaml
steps:
- uses: copy
  config:
    inPath: ./src/manifests
    outPath: ./out/manifests
```

### Ignore Patterns

Copy a directory while excluding some of its contents:

```yaml
steps:
- uses: copy
  config:
    inPath: ./src
    outPath: ./out
    ignore: |
      # Exclude test fixtures and editor artifacts.
      testdata/
      *.tmp
      .DS_Store
```

### Common Usage

The most common (though still advanced) usage of this step is to combine content
from two working trees to use as input to a subsequent step, such as one that
renders Stage-specific manifests.

Consider a Stage that requests Freight from two Warehouses, where one provides
Kustomize "base" configuration, while the other provides a Stage-specific
Kustomize overlay. Rendering the manifests intended for such a Stage will
require combining the base and overlay configurations:

```yaml
vars:
- name: gitRepo
  value: https://github.com/example/repo.git
steps:
- uses: git-clone
  config:
    repoURL: ${{ vars.gitRepo }}
    checkout:
    - commit: ${{ commitFrom(vars.gitRepo, warehouse("base")).ID }}
      path: ./src
    - commit: ${{ commitFrom(vars.gitRepo, warehouse(ctx.stage + "-overlay")).ID }}
      path: ./overlay
    - branch: stage/${{ ctx.stage }}
      create: true
      path: ./out
- uses: git-clear
  config:
    path: ./out
- uses: copy
  config:
    inPath: ./overlay/stages/${{ ctx.stage }}/kustomization.yaml
    outPath: ./src/stages/${{ ctx.stage }}/kustomization.yaml
# Render manifests to ./out, commit, push, etc...
```
