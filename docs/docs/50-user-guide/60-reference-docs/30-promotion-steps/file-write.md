---
sidebar_label: file-write
description: Writes literal or rendered content to a file.
---

# `file-write`

`file-write` writes content to a file in the temporary workspace that Kargo
provisions for a promotion process. It is useful when previous steps have
produced structured data that should be rendered into a new YAML, JSON, or other
text file.

Unlike [`yaml-update`](yaml-update.md), this step replaces the whole file. It
does not preserve comments or formatting from an existing file.

## Configuration

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `path` | `string` | Y | Path to the file to write. This path is relative to the temporary workspace that Kargo provisions for use by the promotion process. |
| `contents` | `string` | Y | Contents to write to the file. This may be an empty string. |
| `overwrite` | `bool` | N | Whether an existing file may be replaced. Defaults to `false`. |

## Examples

### Writing YAML

In this example, a previous step has produced an object and `file-write` renders
that object as YAML in a Stage-specific output branch.

```yaml
steps:
- uses: yaml-parse
  as: read-config
  config:
    path: ./src/apps.yaml
    outputs:
    - name: appConfig
      fromExpression: apps[ctx.stage]
- uses: file-write
  config:
    path: ./out/app-config.yaml
    contents: ${{ asYAML(outputs['read-config'].appConfig) }}
    overwrite: true
# Commit, push, etc...
```

### Writing JSON

Use `asJSON()` when the destination file should contain pretty-printed JSON:

```yaml
steps:
- uses: file-write
  config:
    path: ./out/app-config.json
    contents: ${{ asJSON(outputs['read-config'].appConfig) }}
    overwrite: true
```

The [`git-commit`](git-commit.md) step will pick up files written by
`file-write` when they are inside the Git working tree configured for
`git-commit`.
