---
sidebar_label: json-update
description: Updates the values of specified keys in any JSON file.
---

# `json-update`

`json-update` updates the values of specified keys in any JSON file.

## Configuration

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `path` | `string` | Y | Path to a JSON file. This path is relative to the temporary workspace that Kargo provisions for use by the promotion process. |                      |
| `updates` | `[]object` | Y | The details of changes to be applied to the file. At least one must be specified. |
| `updates[].key` | `string` | Y | The key to update within the file. For nested values, use a JSON dot notation path. See [sjson documentation](https://github.com/tidwall/sjson) for supported syntax. |
| `updates[].value`| `any` | Y | The new value for the key. Typically specified using an expression. Supports strings, numbers, booleans, arrays, and objects. |

## Output

| Name | Type | Description |
|------|------|-------------|
| `commitMessage` | `string` | A description of the change(s) applied by this step. Typically, a subsequent [`git-commit` step](git-commit.md) reference this output and aggregate this commit message fragment with other like it to build a comprehensive commit message that describes all changes. |

## Examples

### Common Usage

In this example, a JSON file's values are updated according to changes in a
container image tag. After cloning the repository and clearing the output
directory, the `json-update` step updates the `image.tag` field in
`configs/settings.json` to match the tag of the image being promoted.
This demonstrates how to modify nested JSON values using dot notation
(similar to how you would reference nested object properties).

This pattern is commonly used when managing configuration files that need to
stay synchronized with deployed container versions.

:::info
For more information on `imageFrom` and expressions, see the
[Expressions](../40-expressions.md#functions) documentation.
:::

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
- uses: json-update
  config:
    path: configs/settings.json
    updates:
    - key: image.tag
      value: ${{ imageFrom("my/image").Tag }}
# Render manifests to ./out, commit, push, etc...
```
