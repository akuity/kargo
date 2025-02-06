---
sidebar_label: json-parse
description: Parses a JSON string and extracts values based on specified expressions.
---

# `json-parse`

`json-parse` is a utility step that parses a JSON string and extracts values using [expr-lang][] expressions.

## Configuration

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `path` | `string` | Y | Path to a JSON file. This path is relative to the temporary workspace that Kargo provisions for use by the promotion process. |
| `outputs` | `[]object` | Y | A list of rules for extracting values from the parsed JSON. |
| `outputs[].name` | `string` | Y | The name of the output variable. |
| `outputs[].fromExpression` | `string` | Y | An [expr-lang](https://expr-lang.org/) expression that can extract the value from the JSON file. Note that this expression should not be offset by `${{` and `}}`. See examples for more details. |

## Expressions

The `fromExpression` field supports [expr-lang](https://expr-lang.org/) expressions.

:::note
Expressions should _not_ be offset by `${{` and `}}` to prevent pre-processing evaluation by Kargo. The `json_parser` step itself will evaluate these expressions.
:::

A `outputs` object (a `map[string]any`) is available to these expressions. It is structured as follows:

| Field | Type | Description |
|-------|------|-------------|
| `outputs` | `map[string]any` | The parsed JSON object. |

## Outputs

The `json_parser` step produces the outputs described by the `outputs` field in its configuration.

## Examples

### Basic Usage

This example extracts values from a JSON object representing a user.

```yaml
steps:
  # ...
  - uses: json_parser
    as: parse-user
    config:
      path: './sample.json'
      outputs:
      - name: userName
        fromExpression: parsed.name
      - name: userAge
        fromExpression: parsed.age
      - name: userCity
        fromExpression: parsed.address.city
```

Given the sample input JSON:

```json
{
  "name": "Alice",
  "age": 30,
  "address": {
    "city": "New York"
  }
}
```

The step would produce the following outputs:

| Name | Type | Value |
|------|------|-------|
| `userName` | `string` | `Alice` |
| `userAge` | `int` | `30` |
| `userCity` | `string` | `New York` |
