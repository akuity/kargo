---
sidebar_label: hcl-update
description: Updates attribute values in HCL files to modify OpenTofu configuration.
---

<span class="tag professional"></span>
<span class="tag beta"></span>

# `hcl-update`

:::info

This promotion step is only available in Kargo on the
[Akuity Platform](https://akuity.io/akuity-platform), versions v1.9 and above.

Additionally, it requires enabling of the Promotion Controller to allow for
Pod-based promotions.

:::

`hcl-update` modifies attribute values in HCL (HashiCorp Configuration Language)
files. This step is typically used to update OpenTofu configuration files before
running [`tf-plan`](tf-plan.md) and [`tf-apply`](tf-apply.md) steps, allowing
you to dynamically set values such as image tags and other configuration 
parameters as part of the promotion process.

## Configuration

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `path` | `string` | Y | Path to an HCL file. This path is relative to the temporary workspace that Kargo provisions for use by the promotion process. |
| `updates` | `[]object` | Y | A list of updates to apply to the HCL file. At least one update must be specified. |
| `updates[].key` | `string` | Y | The key whose value needs to be updated. Supports dot notation for nested values (e.g., `resource.aws_instance.example.tags.version`). |
| `updates[].value` | `string`, `number`, or `boolean` | Y | The new value to set. Strings are quoted, booleans are lowercase (`true`/`false`), and numbers are written as-is. |

## Examples

### Common Usage

The most common usage of this step is to update an OpenTofu variables file with
values from the Freight being promoted. In this example, a container image URI
is updated in a Stage-specific `env.auto.tfvars` file before planning and
applying infrastructure changes.

```yaml
vars:
- name: repoURL
  value: https://github.com/example/infra.git
- name: image
  value: 123456789.dkr.ecr.us-west-2.amazonaws.com/my-app
steps:
- uses: git-clone
  config:
    repoURL: ${{ vars.repoURL }}
    checkout:
    - branch: main
      path: ./src
- uses: hcl-update
  config:
    path: ./src/opentofu/${{ ctx.stage }}/env.auto.tfvars
    updates:
    - key: image_uri
      value: ${{ vars.image }}:${{ imageFrom(vars.image).Tag }}
- uses: tf-apply
  config:
    dir: ./src/opentofu/${{ ctx.stage }}
# Commit and push state changes...
```

### Updating Multiple Values

This example demonstrates updating multiple attributes in a single step. This is
useful when several configuration values need to change together, such as when
deploying a new version with associated settings.

```yaml
steps:
# Clone, prepare configuration, etc...
- uses: hcl-update
  config:
    path: ./src/opentofu/${{ ctx.stage }}/env.auto.tfvars
    updates:
    - key: image_uri
      value: ${{ vars.image }}:${{ imageFrom(vars.image).Tag }}
    - key: replica_count
      value: 3
    - key: enable_monitoring
      value: true
# Plan, apply, etc...
```

### Updating Nested Resource Attributes

This example shows how to use dot notation to update deeply nested attributes
within OpenTofu resource definitions. The key path follows the HCL structure
of the configuration file.

```yaml
steps:
# Clone, prepare configuration, etc...
- uses: hcl-update
  config:
    path: ./src/opentofu/main.tf
    updates:
    - key: resource.aws_lambda_function.app.image_uri
      value: ${{ vars.image }}:${{ imageFrom(vars.image).Tag }}
# Plan, apply, etc...
```
