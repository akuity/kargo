---
sidebar_label: toml-update
description: Updates the values of specified keys in any TOML file.
---

# `toml-update`

`toml-update` updates the values of specified keys in any TOML file.

## Configuration

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `path` | `string` | Y | Path to a TOML file. This path is relative to the temporary workspace that Kargo provisions for use by the promotion process. |
| `updates` | `[]object` | Y | The details of changes to be applied to the file. At least one must be specified. |
| `updates[].key` | `string` | Y | The key to update within the file. For nested values, use dots to delimit key parts. Use `\.` for literal dots in key names and numeric segments for array indexes. |
| `updates[].value` | `any` | Y | The new scalar value for the key. Typically specified using an expression. |

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

**Arrays:**

```toml
values = [1, 2, 3]
```

Update key: `values.1`

:::note

`toml-update` updates existing scalar values in place. It preserves untouched
bytes in the file, but it does not create missing keys or rewrite tables.

:::

## Examples

### Common Usage

In this example, a KCL module manifest is updated to use a new dependency
version. After cloning the repository and clearing the output directory, the
`toml-update` step modifies `kcl.mod` to set the `dependencies.k8s` field.

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
    path: ./src/kcl.mod
    updates:
    - key: dependencies.k8s
      value: ${{ vars.k8sVersion }}
# Commit, push, etc...
```
