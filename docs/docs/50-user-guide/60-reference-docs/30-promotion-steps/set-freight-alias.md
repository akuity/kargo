---
sidebar_label: set-freight-alias
description: Updates the alias of a Freight resource in a project.
---

# `set-freight-alias`

`set-freight-alias` updates the alias of a Freight resource in a project.

## Configuration

| Name          | Type     | Required | Description                                                   |
|---------------|----------|----------|---------------------------------------------------------------|
| `freightName` | `string` | Y | The immutable name (or ID) of the Freight resource to update. |
| `newAlias`    | `string` | Y | The desired new alias to set on the Freight.                  |

:::note

Alias uniqueness is enforced by the Kargo API. If the requested alias is already
in use by another Freight in the project, this step will fail.

:::

## Examples

### Renaming the Freight being actively promoted

The most common use case for `set-freight-alias` is to rename the Freight that is
currently being promoted to reflect its new lifecycle state.

For example, after successfully promoting a Freight to the `staging` Stage, you
may want to update its alias to `now-in-staging`:

```yaml
steps:
- uses: set-freight-alias
  config:
    freightName: ${{ ctx.targetFreight.name }}
    newAlias: now-in-staging
```