---
sidebar_label: set-metadata
description: Updates metadata on Stage or Freight resources during the promotion process.
---

# `set-metadata`

`set-metadata` updates metadata on `Stage` or `Freight` resources during the
promotion process. This step allows you to attach arbitrary key/value pairs to the
status of these resources, which can be useful for tracking promotion state, timing,
version details, or any other relevant metadata.

## Configuration

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `updates` | `[]object` | Y | List of metadata updates to apply. |
| `updates[].kind` | `string` | Y | Kind of resource to update metadata for. Must be either `Stage` or `Freight`. |
| `updates[].name` | `string` | Y | Name of the resource to update metadata for. |
| `updates[].values` | `object` | Y | Key/value pairs to set as metadata on the resource. Must contain at least one key/value pair. This field supports various types of values including strings, numbers, booleans, arrays, and nested objects. |

## Examples

### Common Usage

This example shows how to add simple metadata to both a `Stage` and a `Freight` resource:

```yaml
steps:
- uses: set-metadata
  config:
    updates:
      - kind: Stage
        name: production
        values:
          foo: "hello"
          bar: 42
      - kind: Freight
        name: my-app-freight
        values:
          baz: true
          qux: ["a", "b", "c"]
```

### Complex Metadata

This example demonstrates more complex metadata structures, including nested objects
and arrays:

```yaml
steps:
- uses: set-metadata
  config:
    updates:
      - kind: Stage
        name: staging
        values:
          foo:
            nested: "value"
            numbers: [1, 2, 3]
            items:
              - name: "item1"
                value: "abc"
              - name: "item2"
                value: "xyz"
          bar:
            alpha: "one"
            beta: "two"
```

### Using Metadata

Once metadata is set (as shown in the examples above), it can be retrieved using
[`freightMetadata()`](../40-expressions.md#freightmetadatafreightname) and
[`stageMetadata()`](../40-expressions.md#stagemetadatastagename) functions
available within expressions used in other promotion steps or in verification
processes.
