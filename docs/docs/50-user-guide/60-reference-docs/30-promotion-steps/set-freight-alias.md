---
sidebar_label: set-freight-alias
description: Updates the alias of a Freight resource.
---

# `set-freight-alias`

`set-freight-alias` updates the alias of a `Freight` resource.

When a `Warehouse` produces a new `Freight` resource, it computes a SHA-1 hash
from a canonical representation of the artifacts referenced by that resource.
This unique "fingerprint" for that collection of artifacts becomes the value of
the `metadata.name`. Because this is not a very human-friendly name, every
`Freight` resource is also automatically assigned a human-friendly alias. Users
may sometimes wish to update aliases at various points in their pipelines. This
can make it easier to identify a particularly important (or problematic)
`Freight` resource as it progresses through the `Stage`s of a pipeline.

## Configuration

| Name    | Type     | Required | Description                                                 |
|---------|----------|----------|-------------------------------------------------------------|
| `name`  | `string` | Y | The name of a `Freight` resource to update. |
| `alias` | `string` | Y | The desired new alias to set on the `Freight`.                |

:::note

Alias uniqueness is enforced by the Kargo API. If the requested alias is already
in use by another `Freight` resource in the project, this step will fail.

:::

## Examples

In this example, a `Freight` resource that references only a single container
image and no other artifacts is updated to reflect the image's tag. This makes
it easier to identify such resources and make inferences about their payload.

:::caution

This example is safe only because the `Freight` resource contains only a single
artifact. If that were not the case, contention over a single alias would arise
when two or more Freight resources referenced the same version of the container
image.

:::

```yaml
steps:
- uses: set-freight-alias
  config:
    name: ${{ ctx.targetFreight.name }}
    alias: ${{ imageFrom('some/repo').Tag }}
```