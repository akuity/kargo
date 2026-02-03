---
sidebar_label: set-freight-alias
description: Updates the alias of a `Freight` resource in a project.
---

# `set-freight-alias`

`set-freight-alias` updates the alias of a `Freight` resource in a project.

## Configuration

| Name          | Type     | Required | Description                                                 |
|---------------|----------|----------|-------------------------------------------------------------|
| `freightName` | `string` | Y | The immutable name (or ID) of a `Freight` resource to update. |
| `newAlias`    | `string` | Y | The desired new alias to set on the `Freight`.                |

:::note

Alias uniqueness is enforced by the Kargo API. If the requested alias is already
in use by another `Freight` in the project, this step will fail.

:::

## Examples

### Common Usage

In Kargo, the name of a `Freight` resource is a hash derived from a canonical
representation of that `Freight`, making it a stable fingerprint. This provides
immutability and uniqueness, but the resulting names are not human-friendly.
To improve usability, Kargo assigns each `Freight` a human-friendly alias, which
is initially generated with a random value.

In many cases, teams want to replace this randomly assigned alias with one that
has clear semantic meaning within their organization. Assigning an
organization-specific alias can make a `Freight` easier to reason about across
Stages and environments, improving clarity in promotion pipelines, dashboards,
and operational workflows—without sacrificing the safety guarantees provided by
immutable identifiers.

For example, a team might want to label a `Freight` with an alias that reflects its
intended purpose or lifecycle state within the organization:

```yaml
steps:
- uses: set-freight-alias
  config:
    freightName: ${{ ctx.targetFreight.name }}
    newAlias: "candidate-for-staging"
```

In this example, the `Freight` currently being promoted is identified using its
immutable name, ensuring the correct resource is targeted even in the presence
of concurrent promotions or alias mutations. The alias is then updated to a
human-readable value that is meaningful within the organization.
