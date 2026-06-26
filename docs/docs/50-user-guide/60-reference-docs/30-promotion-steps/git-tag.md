---
sidebar_label: git-tag
description: Creates a new tag for the latest committed changes.
---

# `git-tag`

The `git-tag` step creates a new, annotated tag in a local Git repository
referencing the current `HEAD` of a checked-out branch.

## Configuration

| Name   | Type     | Required | Description                                                                 |
|--------|----------|----------|-----------------------------------------------------------------------------|
| `path` | `string` | Y        | Path to a working directory of a local repository. This path is relative to the temporary workspace that Kargo provisions for use by the promotion process. |
| `tag`  | `string` | Y        | The tag to create. |
| `message` | `string` | Y | The message with which to annotate the tag. |
| `force` | `boolean` | N | Whether to overwrite an existing tag of the same name. Defaults to `false`. **Caution:** Overwriting a tag is unconventional—tags are normally immutable references to a specific commit—so enable this only with a deliberate reason. |

## Output

| Name  | Type     | Description                                                                 |
|-------|----------|-----------------------------------------------------------------------------|
| `commit` | `string` | The ID (SHA) of the commit pushed by this step. |

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

### Overwriting an Existing Tag

In this example, the `git-tag` step moves an existing tag to the current `HEAD`
by setting `force` to `true`. Without `force`, the step fails if the tag already
exists.

:::caution
Force overwriting a tag is an unconventional use of tags. Use it only with
extreme caution and a deliberate reason. See the warning under
[Configuration](#configuration).
:::

```yaml
steps:
- uses: git-tag
  config:
    path: ./out
    tag: dev-team-a
    force: true
```
