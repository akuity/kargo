---
sidebar_label: git-clear
description: Deletes the entire contents of a specified Git working tree.
---

# `git-clear`

`git-clear` deletes _the entire contents_ of a specified Git working tree
(except for the `.git` file). It is equivalent to executing
`git add . && git rm -rf --ignore-unmatch .`. This step is useful for the common
scenario where the entire content of a Stage-specific branch is to be replaced
with content from another branch or with content rendered using some
configuration management tool.

## Configuration

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `path` | `string` | Y | Path to a Git working tree whose entire contents are to be deleted. |

## Examples

```yaml
steps:
- uses: git-clone
  config:
    repoURL: https://github.com/example/repo.git
    checkout:
    - branch: stage/${{ ctx.stage }}
      create: true
      path: ./out
- uses: git-clear
  config:
    path: ./out
# Prepare the contents of ./out ...
# Commit, push, etc...
```
