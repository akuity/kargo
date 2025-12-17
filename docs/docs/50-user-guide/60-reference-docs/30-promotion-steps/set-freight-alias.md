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

### Common Usage

In many organizations, Freight identifiers are opaque and not meaningful to humans
(e.g. commit SHAs or generated IDs). Teams often want to assign human-friendly,
organization-specific aliases to Freights to make them easier to reason about
across Stages and environments.

A common pattern is to update a Freight’s alias early in the promotion lifecycle,
such as in a pre-processing Stage, once the Freight has been selected but before
it is promoted further.

For example, a team might want to label a Freight with an alias that reflects its
intended purpose or lifecycle state within the organization:

```yaml
steps:
- uses: set-freight-alias
  config:
  freightName: ${{ ctx.targetFreight.name }}
  newAlias: "candidate-for-staging"
```

In this example, the Freight currently being promoted is identified using its immutable name,
ensuring the correct resource is targeted even in the presence of concurrent promotions or alias
mutations. The alias is then updated to a human-readable value that is meaningful within the
organization. 
