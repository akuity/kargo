---
sidebar_label: toml-update
description: Updates the values of specified keys in any TOML file.
---

# `toml-update`

<span class="tag beta"></span>

`toml-update` updates the values of specified keys in any TOML file, in-place,
without disruption to existing formatting choices.

:::note[Limitations]

`toml-update` updates scalar values only and can only update the values of
existing keys.

:::

## Configuration

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `path` | `string` | Y | Path to a TOML file. This path is relative to the temporary workspace that Kargo provisions for use by the promotion process. |
| `updates` | `[]object` | Y | The details of changes to be applied to the file. At least one must be specified. |
| `updates[].key` | `string` | Y | The key to update within the file. For nested values, use dots to delimit key parts. e.g. `image.tag`. The syntax is identical to that supported by the `json-update` step and is documented in more detail [here](https://github.com/tidwall/sjson?tab=readme-ov-file#path-syntax). |
| `updates[].value` | `any` | Y | The new scalar value for the key. Typically specified using an expression. Supports strings, numbers, and booleans. |

## Output

| Name | Type | Description |
|------|------|-------------|
| `commitMessage` | `string` | A description of the change(s) applied by this step. Typically, a subsequent [`git-commit` step](git-commit.md) will reference this output and aggregate this commit message fragment with others like it to build a comprehensive commit message that describes all changes. |

## Writing Keys

**Nested keys:**

```toml
[package]
version = "1.0.0"
```

Update key: `package.version`

**Keys with literal dots:**

```toml
[labels]
"example.com/version" = "1.0.0"
```

Update key: `labels.example\.com/version`

**Sequences:**

```toml
values = [1, 2, 3]
```

Update key: `values.1`

:::note

See the [sjson path syntax documentation](https://github.com/tidwall/sjson?tab=readme-ov-file#path-syntax)
for the full description of the syntax.

:::

## Examples

### Common Usage

In this example, a TOML file's values are updated according to changes in a
container image tag. After cloning the repository and clearing the output
directory, the `toml-update` step updates the `image.tag` field in
`configs/settings.toml` to match the tag of the image being promoted.

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
- uses: toml-update
  config:
    path: ./src/configs/settings.toml
    updates:
    - key: image.tag
      value: ${{ imageFrom("my/image").Tag }}
# Render manifests to ./out, commit, push, etc...
```

:::info

For more information on `imageFrom()` and expressions used in the example above,
see the [Expressions](../40-expressions.md#functions) documentation.

:::
