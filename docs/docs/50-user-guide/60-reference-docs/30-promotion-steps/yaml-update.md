---
sidebar_label: yaml-update
description: Updates the values of specified keys in any YAML file.
---

# `yaml-update`

`yaml-update` updates the values of specified keys in any YAML file, in-place,
without disruption to existing formatting choices.

:::note[Limitations]

`yaml-update` updates scalar values only and can only update the values of
existing keys.

:::

## Configuration

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `path` | `string` | Y | Path to a YAML file. This path is relative to the temporary workspace that Kargo provisions for use by the promotion process. |
| `updates` | `[]object` | Y | The details of changes to be applied to the file. At least one must be specified. |
| `updates[].key` | `string` | Y | The key to update within the file. For nested values, use dots to delimit key parts. e.g. `image.tag`. The syntax is identical to that supported by the `json-update` step and is documented in more detail [here](https://github.com/tidwall/sjson?tab=readme-ov-file#path-syntax). |
| `updates[].value` | `any` | Y | The new scalar value for the key. Typically specified using an expression. Supports strings, numbers, and booleans. |

## Output

| Name | Type | Description |
|------|------|-------------|
| `commitMessage` | `string` | A description of the change(s) applied by this step. Typically, a subsequent [`git-commit` step](git-commit.md) will reference this output and aggregate this commit message fragment with others like it to build a comprehensive commit message that describes all changes. |

## Writing Keys

**Nested keys:**

```yaml
image:
  tag: v1.0.0
```

Update key: `image.tag`

**Keys with literal dots:**

```yaml
example.com/version: v1.0.0
```

Update key: `example\.com/version`

**Sequences:**

```yaml
containers:
- name: my-app
  image: my-app:v1.0
```

Update key: `containers.0.image`

:::note

See the [sjson path syntax documentation](https://github.com/tidwall/sjson?tab=readme-ov-file#path-syntax)
for the full description of the syntax.

:::

## Examples

### Common Usage

In this example, a Helm values file is updated to use a new container image tag.
After cloning the repository and clearing the output directory, the
`yaml-update` step modifies `values.yaml` to use the image tag from the
`Freight` being promoted.

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
- uses: yaml-update
  config:
    path: ./src/charts/my-chart/values.yaml
    updates:
    - key: image.tag
      value: ${{ imageFrom("my/image").Tag }}
# Render manifests to ./out, commit, push, etc...
```

:::info

For more information on `imageFrom()` and expressions used in the example above,
see the [Expressions](../40-expressions.md#functions) documentation.

:::
