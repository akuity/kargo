---
sidebar_label: git-tag
description: Creates a new tag for the latest committed changes.
---

# `git-tag`

The `git-tag` step creates a new tag in a local Git repository. This step is commonly used to mark specific commits with a tag, which can be useful for versioning or tracking changes in a repository.

## Configuration

| Name   | Type     | Required | Description                                                                 |
|--------|----------|----------|-----------------------------------------------------------------------------|
| `path` | `string` | Y        | Path to a working directory of a local repository. This path is relative to the temporary workspace that Kargo provisions for use by the promotion process. |
| `tag`  | `string` | Y        | The name of the tag to create. |

## Output

| Name  | Type     | Description                                                                 |
|-------|----------|-----------------------------------------------------------------------------|
| `tag` | `string` | The name of the tag that was created by this step. This can be referenced in subsequent steps. |
| `commit` | `string` | The ID (SHA) of the commit pushed by this step. |
| `commitURL` | `string` | The URL of the commit that was pushed to the remote repository. |

:::caution

If the specified tag already exists in the remote repository, the `git-tag` step
will fail.

:::

## Examples

### Basic Usage

In this example, the `git-tag` step creates a tag named `v1.0.0` in a local Git repository.

```yaml
steps:
- uses: git-tag
  config:
    path: ./out
    tag: v1.0.0
```

### Tagging After a Commit

This example demonstrates how to use the git-tag step after a git-commit step to tag the latest commit with a version number.

```yaml
steps:
- uses: git-commit
  config:
    path: ./out
    message: "Committing changes for release v1.0.0"
- uses: git-tag
  config:
    path: ./out
    tag: v1.0.0
```

### Pushing After Tagging

In this example, the `git-tag` step creates a tag, and the `git-push` step pushes the tag to the remote repository.

```yaml
steps:
- uses: git-tag
  config:
    path: ./out
    tag: v1.0.0
- uses: git-push
  config:
    path: ./out
    tag: v1.0.0
```