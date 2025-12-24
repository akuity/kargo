---
sidebar_label: tf-output
description: Retrieves outputs from OpenTofu state for use in subsequent steps.
---

<span class="tag professional"></span>
<span class="tag beta"></span>

# `tf-output`

:::info

This promotion step is only available in Kargo on the
[Akuity Platform](https://akuity.io/akuity-platform), versions v1.9 and above.

Additionally, it requires enabling of the Promotion Controller to allow for
Pod-based promotions.

:::

`tf-output` retrieves output values from OpenTofu state. These outputs can be
used in subsequent promotion steps or written to a JSON file. This step is
typically used after [`tf-apply`](tf-apply.md) to access values such as resource
IDs, endpoints, or other computed attributes.

:::note

By default, sensitive outputs are filtered from the results when retrieving all
outputs. Set `sensitive: true` to include them, or retrieve a specific output by
name to bypass filtering.

:::

## Configuration

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `dir` | `string` | Y | Directory containing OpenTofu configuration files. This path is relative to the temporary workspace that Kargo provisions for use by the promotion process. |
| `name` | `string` | N | Name of a specific output to retrieve. When specified, only this output is returned. When omitted, all outputs are retrieved. |
| `out` | `string` | N | Path where outputs will be written as JSON, relative to the temporary workspace. When specified, outputs are written to this file instead of being returned in the step output. |
| `state` | `string` | N | Path to a custom state file, relative to the `dir` directory. When omitted, the default state file is used. |
| `sensitive` | `boolean` | N | Whether to include sensitive outputs in the results. Defaults to `false`. Only applies when retrieving all outputs (when `name` is not specified). |
| `vars` | `[]object` | N | Variables to pass to OpenTofu. |
| `vars[].file` | `string` | N | Path to a variables file (`.tfvars`), relative to the `dir` directory. Mutually exclusive with `name`/`value`. |
| `vars[].name` | `string` | N | Variable name. Required when not using `file`. Mutually exclusive with `file`. |
| `vars[].value` | `string` | N | Variable value. Required when not using `file`. Mutually exclusive with `file`. |
| `env` | `[]object` | N | Environment variables to set during OpenTofu execution. |
| `env[].name` | `string` | Y | Environment variable name. Must match the pattern `^[a-zA-Z_][a-zA-Z0-9_]*$`. |
| `env[].value` | `string` | Y | Environment variable value. |

## Output

The output format depends on the configuration:

**When `out` is specified:**

The step writes outputs to the specified file as JSON and returns an empty map.

**When `out` is not specified and `name` is specified:**

| Name | Type | Description |
|------|------|-------------|
| `{name}` | `any` | The value of the specified output. The key matches the `name` parameter. |

**When `out` is not specified and `name` is not specified:**

| Name | Type | Description |
|------|------|-------------|
| `{output_name}` | `object` | Each output is returned as an object containing `value`, `type`, and `sensitive` fields. Sensitive outputs are excluded unless `sensitive: true` is set. |

## Examples

### Common Usage

The most common usage of this step is to retrieve outputs from the OpenTofu
state after applying configuration. This example retrieves the function URL
from an AWS Lambda deployment for use in subsequent steps.

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
    env:
    - name: AWS_REGION
      value: us-west-2
    - name: AWS_ACCESS_KEY_ID
      value: ${{ secret('aws-creds').awsAccessKeyID }}
    - name: AWS_SECRET_ACCESS_KEY
      value: ${{ secret('aws-creds').awsSecretAccessKey }}
- uses: tf-output
  as: infra
  config:
    dir: ./src/opentofu/${{ ctx.stage }}
    env:
    - name: AWS_REGION
      value: us-west-2
    - name: AWS_ACCESS_KEY_ID
      value: ${{ secret('aws-creds').awsAccessKeyID }}
    - name: AWS_SECRET_ACCESS_KEY
      value: ${{ secret('aws-creds').awsSecretAccessKey }}
# Commit and push state changes...
```

The outputs can then be referenced in subsequent steps:

```yaml
- uses: http
  config:
    url: ${{ outputs.infra.function_url.value }}
```

### Retrieving a Specific Output

This example retrieves a single output by name. When retrieving by name, the
step returns only the value without the metadata wrapper, making it easier to
use in subsequent steps.

```yaml
steps:
# Clone, plan, apply, etc...
- uses: tf-output
  as: endpoint
  config:
    dir: ./src/opentofu/${{ ctx.stage }}
    name: function_url
    env:
    - name: AWS_REGION
      value: us-west-2
    - name: AWS_ACCESS_KEY_ID
      value: ${{ secret('aws-creds').awsAccessKeyID }}
    - name: AWS_SECRET_ACCESS_KEY
      value: ${{ secret('aws-creds').awsSecretAccessKey }}
```

The output value can be referenced directly:

```yaml
- uses: http
  config:
    url: ${{ outputs.endpoint.function_url }}
```

### Writing Outputs to a File

This example writes all outputs to a JSON file. This is useful when outputs need
to be consumed by external tools or processes outside of Kargo's promotion
workflow.

```yaml
steps:
# Clone, plan, apply, etc...
- uses: tf-output
  config:
    dir: ./src/opentofu/${{ ctx.stage }}
    out: ./outputs.json
    env:
    - name: AWS_REGION
      value: us-west-2
    - name: AWS_ACCESS_KEY_ID
      value: ${{ secret('aws-creds').awsAccessKeyID }}
    - name: AWS_SECRET_ACCESS_KEY
      value: ${{ secret('aws-creds').awsSecretAccessKey }}
```

### Including Sensitive Outputs

This example demonstrates how to include sensitive outputs in the results. This
is useful when you need access to values that OpenTofu marks as sensitive, such
as generated passwords or API keys.

:::warning

Exercise caution when including sensitive outputs, as they may contain secrets
or other confidential information.

:::

```yaml
steps:
# Clone, plan, apply, etc...
- uses: tf-output
  as: outputs
  config:
    dir: ./src/opentofu/${{ ctx.stage }}
    sensitive: true
    env:
    - name: AWS_REGION
      value: us-west-2
    - name: AWS_ACCESS_KEY_ID
      value: ${{ secret('aws-creds').awsAccessKeyID }}
    - name: AWS_SECRET_ACCESS_KEY
      value: ${{ secret('aws-creds').awsSecretAccessKey }}
```
