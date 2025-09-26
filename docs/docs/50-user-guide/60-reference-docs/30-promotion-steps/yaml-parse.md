---
sidebar_label: yaml-parse
description: Parses a YAML file and extracts values based on specified expressions.
---

# `yaml-parse`

`yaml-parse` is a utility step that parses a YAML file and extracts values
using [expr-lang] expressions.

## Configuration

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `path` | `string` | Y | Path to a YAML file. This path is relative to the temporary workspace that Kargo provisions for use by the promotion process. |
| `outputs` | `[]object` | Y | A list of rules for extracting values from the parsed YAML. |
| `outputs[].name` | `string` | Y | The name of the output variable. |
| `outputs[].fromExpression` | `string` | Y | An [expr-lang] expression that can extract the value from the YAML file. Note that this expression should not be offset by `${{` and `}}`. See [examples](#examples) for more details. |

## Expressions

The `fromExpression` field supports [expr-lang] expressions.

:::note
Expressions should _not_ be offset by `${{` and `}}` to prevent pre-processing
evaluation by Kargo. The `yaml-parse` step itself will evaluate these
expressions.
:::

An `outputs` object (a `map[string]any`) is available to these expressions. It
is structured as follows:

| Field | Type | Description |
|-------|------|-------------|
| `outputs` | `map[string]any` | The parsed YAML object. |

### Root Types & Synthetic `root` Key

The YAML document being parsed can have any root type. If the root of the YAML
is a mapping (object), its keys are exposed directly for expressions (as in the
example above). If the root is a sequence (list) or a scalar (string, number,
bool, or null), the value is wrapped under a synthetic key `root` to provide a
stable object map for expression evaluation.

Summary:

| Original YAML root type | Expression access pattern |
|-------------------------|---------------------------|
| Mapping (object)        | Use keys directly (e.g. `image.tag`) |
| Sequence (list)         | `root` holds the slice (e.g. `root[0].name`) |
| Scalar (string/number/bool/null) | `root` holds the scalar (e.g. `root`) |

Examples:

Sequence root:
```yaml
- name: api
  image: example/api:v1
- name: worker
  image: example/worker:v2
```
Expressions:
| Purpose | Expression | Result |
|---------|------------|--------|
| First name | `root[0].name` | `api` |
| Second image | `root[1].image` | `example/worker:v2` |

Scalar root (string):
```yaml
1.2.3
```
Expression to capture value:
| Purpose | Expression | Result |
|---------|------------|--------|
| Version | `root` | `1.2.3` |

Scalar root (boolean):
```yaml
true
```
Expression: `root` → `true`

Scalar root (number):
```yaml
42
```
Expression: `root` → `42`

Note: This wrapping only affects non-mapping roots. Mapping-based documents are
left unchanged for backward compatibility.

## Outputs

The `yaml-parse` step produces the outputs described by the `outputs` field in
its configuration.

## Examples

### Common Usage

In this example, a Helm values file is parsed to find the container image tag.
After cloning the repository and clearing the output directory, the `yaml-parse`
step parses `values.yaml` to extract the image tag from the `Freight` being
promoted. Using dot notation (`image.tag`), it extracts the nested value from
the YAML file.

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
- uses: yaml-parse
  as: values
  config:
    path: './src/charts/my-chart/values.yaml'
    outputs:
    - name: imageTag
      fromExpression: image.tag
# Render manifests to ./out, commit, push, etc...
```

Given the sample input YAML:

```yaml
image:
  tag: latest
rbac:
  installClusterRoles: true
```

The step would produce the following
[outputs](../15-promotion-templates.md#step-outputs):

| Name | Type | Value |
|------|------|-------|
| `imageTag` | `string` | `latest` |

[expr-lang]: https://expr-lang.org
