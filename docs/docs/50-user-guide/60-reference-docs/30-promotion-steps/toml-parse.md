---
sidebar_label: toml-parse
description: Parses a TOML file and extracts values based on specified expressions.
---

# `toml-parse`

`toml-parse` is a utility step that parses a TOML file and extracts values
using [expr-lang] expressions.

## Configuration

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `path` | `string` | Y | Path to a TOML file. This path is relative to the temporary workspace that Kargo provisions for use by the promotion process. |
| `outputs` | `[]object` | Y | A list of rules for extracting values from the parsed TOML. |
| `outputs[].name` | `string` | Y | The name of the output variable. |
| `outputs[].fromExpression` | `string` | Y | An [expr-lang] expression that can extract the value from the TOML file. Note that this expression should not be offset by `${{` and `}}`. See [examples](#examples) for more details. |

## Expressions

The `fromExpression` field supports [expr-lang] expressions.

:::note

Expressions should _not_ be offset by `${{` and `}}` to prevent pre-processing
execution by Kargo. The `toml-parse` step itself evaluates these expressions.

:::

An `outputs` object (a `map[string]any`) is available to these expressions. It
contains the parsed TOML document.

## Outputs

The `toml-parse` step produces the outputs described by the `outputs` field in
its configuration.

## Examples

### Common Usage

In this example, a TOML file is parsed to find the package version. After
cloning the repository and clearing the output directory, the `toml-parse` step
parses `Cargo.toml` and extracts the package version from the repository being
promoted.

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
- uses: toml-parse
  as: cargo
  config:
    path: ./src/Cargo.toml
    outputs:
    - name: version
      fromExpression: package.version
# Commit, push, etc...
```

Given the sample input TOML:

```toml
[package]
name = "demo"
version = "1.2.3"
```

The step would produce the following
[outputs](../15-promotion-templates.md#step-outputs):

| Name | Type | Value |
|------|------|-------|
| `version` | `string` | `1.2.3` |

[expr-lang]: https://expr-lang.org
