---
sidebar_label: json-parse
description: Parses a JSON string and extracts values based on specified expressions.
---

# `json-parse`

`json-parse` is a utility step that parses a JSON string and extracts values
using [expr-lang] expressions.

## Configuration

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `path` | `string` | Y | Path to a JSON file. This path is relative to the temporary workspace that Kargo provisions for use by the promotion process. |
| `outputs` | `[]object` | Y | A list of rules for extracting values from the parsed JSON. |
| `outputs[].name` | `string` | Y | The name of the output variable. |
| `outputs[].fromExpression` | `string` | Y | An [expr-lang] expression that can extract the value from the JSON file. Note that this expression should not be offset by `${{` and `}}`. See [examples](#examples) for more details. |

## Expressions

The `fromExpression` field supports [expr-lang] expressions.

:::note
Expressions should _not_ be offset by `${{` and `}}` to prevent pre-processing
evaluation by Kargo. The `json-parse` step itself will evaluate these
expressions.
:::

An `outputs` object (a `map[string]any`) is available to these expressions. It
is structured as follows:

| Field | Type | Description |
|-------|------|-------------|
| `outputs` | `map[string]any` | The parsed JSON object. |

## Outputs

The `json-parse` step produces the outputs described by the `outputs` field in
its configuration.

## Examples

### Common Usage

In this example, a values file is parsed to find the container image tag.
After cloning the repository and clearing the output directory, the `json-parse`
step parses `values.json` to extract the image tag from the `Freight` being
promoted. Using dot notation (`image.tag`), it extracts the nested value from
the JSON file.

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
- uses: json-parse
  as: values
  config:
    path: './src/charts/my-chart/values.json'
    outputs:
    - name: imageTag
      fromExpression: image.tag
# Render manifests to ./out, commit, push, etc...
```

Given the sample input JSON:

```json
{
  "image": {
    "tag": "latest"
  },
  "rbac": {
    "installClusterRoles": true
  }
}
```

The step would produce the following
[outputs](../15-promotion-templates.md#step-outputs):

| Name | Type | Value |
|------|------|-------|
| `imageTag` | `string` | `latest` |

[expr-lang]: https://expr-lang.org
