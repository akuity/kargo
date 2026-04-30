---
sidebar_label: json-update
description: Updates the values of specified keys in any JSON file.
---

# `json-update`

`json-update` updates the values of specified keys in any JSON file, in-place,
without disruption to existing formatting choices.

:::note[Limitations]

`json-update` updates scalar values only.

:::

## Configuration

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `path` | `string` | Y | Path to a JSON file. This path is relative to the temporary workspace that Kargo provisions for use by the promotion process. |
| `updates` | `[]object` | Y | The details of changes to be applied to the file. At least one must be specified. |
| `updates[].key` | `string` | Y | The key to update within the file. For nested values, use a JSON dot notation path. See [sjson documentation](https://github.com/tidwall/sjson?tab=readme-ov-file#path-syntax) for supported syntax. |
| `updates[].value`| `any` | Y | The new scalar value for the key. Typically specified using an expression. Supports strings, numbers, and booleans. |

## Output

| Name | Type | Description |
|------|------|-------------|
| `commitMessage` | `string` | A description of the change(s) applied by this step. Typically, a subsequent [`git-commit` step](git-commit.md) will reference this output and aggregate this commit message fragment with others like it to build a comprehensive commit message that describes all changes. |

## Writing Keys

**Nested keys:**

```json
{
  "image": {
    "tag": "v1.0.0"
  }
}
```

Update key: `image.tag`

**Keys with literal dots:**

```json
{
  "example.com/version": "v1.0.0"
}
```

Update key: `example\.com/version`

**Sequences:**

```json
{
  "containers": [
    { "name": "my-app", "image": "my-app:v1.0" }
  ]
}
```

Update key: `containers.0.image`

:::note

See the [sjson path syntax documentation](https://github.com/tidwall/sjson?tab=readme-ov-file#path-syntax)
for the full description of the syntax.

:::

## Examples

### Common Usage

In this example, a JSON file's values are updated according to changes in a
container image tag. After cloning the repository and clearing the output
directory, the `json-update` step updates the `image.tag` field in
`configs/settings.json` to match the tag of the image being promoted.

This pattern is commonly seen when managing configuration files that need to
stay synchronized with deployed container versions.

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
    path: ./src/configs/settings.json
    updates:
    - key: image.tag
      value: ${{ imageFrom("my/image").Tag }}
# Render manifests to ./out, commit, push, etc...
```

:::info

For more information on `imageFrom()` and expressions used in the example above,
see the [Expressions](../40-expressions.md#functions) documentation.

:::
