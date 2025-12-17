---
sidebar_label: set-freight-alias
description: Updates the alias of a Freight Resource in a project`
---

# `set-freight-alias`

`set-freight-alias` updates the alias of a Freight Resource in a project.
Unlike most promotion steps, which operate on the Freight currently being promoted,
this step explicitly targets a Freight via its `freightID`. This allows updating aliases 
for other Freights in the same project, even if they are not part of the current promotion.

:::note

Always ensure the new alias is unique within the project. If the alias is already used by another Freight, this step will fail.

:::

## Configuration

| Name        | Type     | Required | Description                                                     |
|-------------|----------|----------|-----------------------------------------------------------------|
| `freightID` | `string` | Y        | The ID of the Freight resource to update. Must not be empty.    |
| `newAlias`  | `string` | Y        | The desired new alias to set on the Freight. Must not be empty. |

## Examples

### Common Usage

Sometimes, you may want to rename a Freight alias for a resource that is not currently being promoted,
but exists in the same project. This is useful for housekeeping, environment reassignments, or preparing resources
for future promotions without affecting the currently promoted Freight.

For example, suppose you have a Freight `freight-id-456` that represents a previous deployment in your project.
Its alias is `staging-old`, and you want to rename it to `archived` to reflect its status:

```yaml
steps:
- uses: set-freight-alias
  config:
    freightID: "freight-id-456"
    newAlias: "archived
```

After this step runs, the freight `freight-id-456` will have its alias updated to `archived`,
while the Freight currently being promoted remains unaffected. The corresponding label 
`kargo.akuity.io/alias` will also be updated to reflect the new alias.
