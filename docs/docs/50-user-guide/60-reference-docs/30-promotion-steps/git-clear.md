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

### Common Usage

In this example, all content in a stage-specific branch is removed before new
content is added. After cloning the repository with a stage-specific branch
checked out to `./out`, the `git-clear` step removes all files from this
working directory (except the `.git` directory).

This pattern is commonly used when you want to completely replace the contents
of a branch with newly generated content, rather than updating existing files.
This ensures there are no stale files left over from previous states of the
branch, providing a clean slate for the new content that will be added by
subsequent steps.

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
