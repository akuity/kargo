---
sidebar_label: tf-plan
description: Executes an OpenTofu plan operation to preview infrastructure changes.
---

<span class="tag professional"></span>
<span class="tag beta"></span>

# `tf-plan`

:::info

This promotion step is only available in Kargo on the
[Akuity Platform](https://akuity.io/akuity-platform), versions v1.9 and above.

Additionally, it requires enabling of the Promotion Controller to allow for
Pod-based promotions.

:::

`tf-plan` executes an OpenTofu plan operation to preview what infrastructure
changes would be made. This step initializes the OpenTofu working directory and
generates an execution plan, which can optionally be saved to a file for use
with the [`tf-apply`](tf-apply.md) step.

:::note

The step returns `Succeeded` when changes are detected and `Skipped` when no
changes are detected. This allows you to conditionally execute subsequent steps
based on whether infrastructure changes are needed.

:::

## Configuration

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `dir` | `string` | Y | Directory containing OpenTofu configuration files. This path is relative to the temporary workspace that Kargo provisions for use by the promotion process. |
| `out` | `string` | N | Path where the generated plan file will be saved, relative to the `dir` directory. When specified, the plan can be applied using the [`tf-apply`](tf-apply.md) step. |
| `vars` | `[]object` | N | Variables to pass to OpenTofu. |
| `vars[].file` | `string` | N | Path to a variables file (`.tfvars`), relative to the `dir` directory. Mutually exclusive with `name`/`value`. |
| `vars[].name` | `string` | N | Variable name. Required when not using `file`. Mutually exclusive with `file`. |
| `vars[].value` | `string` | N | Variable value. Required when not using `file`. Mutually exclusive with `file`. |
| `env` | `[]object` | N | Environment variables to set during OpenTofu execution. |
| `env[].name` | `string` | Y | Environment variable name. Must match the pattern `^[a-zA-Z_][a-zA-Z0-9_]*$`. |
| `env[].value` | `string` | Y | Environment variable value. |

## Output

| Name | Type | Description |
|------|------|-------------|
| `plan` | `string` | The plan output as text, showing what changes would be made. |

## Examples

### Common Usage

The most common usage of this step is to run a plan operation on Stage-specific
OpenTofu configuration. This example shows planning infrastructure changes for
an AWS Lambda deployment, with credentials provided via Kargo secrets.

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
- uses: tf-plan
  as: plan
  config:
    dir: ./src/opentofu/${{ ctx.stage }}
    env:
    - name: AWS_REGION
      value: us-west-2
    - name: AWS_ACCESS_KEY_ID
      value: ${{ secret('aws-creds').awsAccessKeyID }}
    - name: AWS_SECRET_ACCESS_KEY
      value: ${{ secret('aws-creds').awsSecretAccessKey }}
```

### Plan Output in Pull Request

This example demonstrates a review-based workflow where the plan output is
included in a pull request description. This allows reviewers to see exactly
what infrastructure changes will be made before approving the PR. The step
alias (`as: tf-plan`) allows referencing the plan output in subsequent steps.

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
- uses: tf-plan
  as: tf-plan
  config:
    dir: ./src/opentofu/${{ ctx.stage }}
    env:
    - name: AWS_REGION
      value: us-west-2
    - name: AWS_ACCESS_KEY_ID
      value: ${{ secret('aws-creds').awsAccessKeyID }}
    - name: AWS_SECRET_ACCESS_KEY
      value: ${{ secret('aws-creds').awsSecretAccessKey }}
- uses: git-commit
  config:
    path: ./src
    message: Update ${{ ctx.stage }} to ${{ imageFrom(vars.image).Tag }}
- uses: git-push
  as: push
  config:
    generateTargetBranch: true
    path: ./src
- uses: git-open-pr
  as: open-pr
  config:
    repoURL: ${{ vars.repoURL }}
    sourceBranch: ${{ task.outputs.push.branch }}
    targetBranch: main
    description: |-
      ## OpenTofu Plan Output

      ${{ task.outputs['tf-plan'].plan }}
- uses: git-wait-for-pr
  config:
    prNumber: ${{ task.outputs['open-pr'].prNumber }}
    repoURL: ${{ vars.repoURL }}
```

### Saving Plan for Apply

This example saves the plan to a file that can be used with the
[`tf-apply`](tf-apply.md) step. This ensures that exactly the changes shown in 
the plan are applied, with no possibility of drift between planning and applying.

```yaml
steps:
# Clone, update configuration, etc...
- uses: tf-plan
  as: plan
  config:
    dir: ./src/opentofu/${{ ctx.stage }}
    out: tfplan
    env:
    - name: AWS_REGION
      value: us-west-2
    - name: AWS_ACCESS_KEY_ID
      value: ${{ secret('aws-creds').awsAccessKeyID }}
    - name: AWS_SECRET_ACCESS_KEY
      value: ${{ secret('aws-creds').awsSecretAccessKey }}
- uses: tf-apply
  config:
    dir: ./src/opentofu/${{ ctx.stage }}
    plan: tfplan
    env:
    - name: AWS_REGION
      value: us-west-2
    - name: AWS_ACCESS_KEY_ID
      value: ${{ secret('aws-creds').awsAccessKeyID }}
    - name: AWS_SECRET_ACCESS_KEY
      value: ${{ secret('aws-creds').awsSecretAccessKey }}
```
