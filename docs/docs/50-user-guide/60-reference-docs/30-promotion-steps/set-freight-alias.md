---
sidebar_label: set-freight-alias
description: Updates the alias of a Freight resource.
---

# `set-freight-alias`

`set-freight-alias` updates the alias of a `Freight` resource.

When a `Warehouse` produces a new `Freight` resource, it computes a SHA-1 hash
from a canonical representation of the artifacts referenced by that resource.
This becomes the value of the `metadata.name` field — a unique "fingerprint" of
the collection of artifacts. While every `Freight` resource is also automatically
assigned a human-friendly alias, users may sometimes wish to override that alias
with one of their own choosing. This can make it easier to identify a
particularly important (or problematic) `Freight` resource as it progresses
through the `Stage`s of a pipeline.

## Configuration

| Name          | Type     | Required | Description                                                 |
|---------------|----------|----------|-------------------------------------------------------------|
| `freightName` | `string` | Y | The name of a `Freight` resource to update. |
| `newAlias`    | `string` | Y | The desired new alias to set on the `Freight`.                |

:::note

Alias uniqueness is enforced by the Kargo API. If the requested alias is already
in use by another `Freight` resource in the project, this step will fail.

:::

## Examples

### Common Usage

```yaml
steps:
- uses: set-freight-alias
  config:
    freightName: ${{ ctx.targetFreight.name }}
    newAlias: ${{ imageFrom('some/repo').Tag }}
```