---
sidebar_label: gha-dispatch-workflow
description: Dispatches GitHub Actions workflows using the workflow_dispatch event.
---

# `gha-dispatch-workflow`

<span class="tag professional"></span>
<span class="tag beta"></span>

:::info
This promotion step is only available in Kargo on the [Akuity Platform](https://akuity.io/akuity-platform), versions v1.8 and above.
:::

The `gha-dispatch-workflow` promotion step provides integration with GitHub Actions, allowing you to dispatch workflows using the `workflow_dispatch` event. This is particularly useful for triggering CI/CD pipelines, running tests, or executing deployment scripts as part of your promotion workflows.

## Credentials Configuration

All GitHub Actions operations require proper authentication credentials stored in a Kubernetes `Secret`.

| Name                     | Type     | Required | Description                                                                      |
| ------------------------ | -------- | -------- | -------------------------------------------------------------------------------- |
| `credentials.secretName` | `string` | Y        | Name of the `Secret` containing the GitHub credentials in the project namespace. |

The referenced `Secret` should contain the following keys:

- `accessToken`: GitHub personal access token or GitHub App token with appropriate permissions
- `baseURL`: (Optional) GitHub base URL for GitHub Enterprise Server
- `uploadURL`: (Optional) GitHub upload URL for GitHub Enterprise Server. Only required for GitHub Enterprise Server installations

:::info Required Permissions
The GitHub token must have the following permissions:

**Fine-grained Personal Access Token:**
- `actions:write` - To dispatch workflows
- `actions:read` - To read workflow run status

**Classic Personal Access Token:**
- `repo` - read/write access
:::

## Configuration

| Name           | Type      | Required | Description                                                                                           |
| -------------- | --------- | -------- | ----------------------------------------------------------------------------------------------------- |
| `owner`        | `string`  | Y        | The owner of the repository (user or organization).                                                   |
| `repo`         | `string`  | Y        | The name of the repository.                                                                           |
| `workflowFile` | `string`  | Y        | The workflow filename in .github/workflows (e.g., 'deploy.yml').                                      |
| `ref`          | `string`  | Y        | The git reference (branch or tag) to run the workflow on.                                             |
| `inputs`       | `object`  | N        | Input parameters to pass to the workflow as defined in the workflow's `workflow_dispatch` inputs.     |
| `timeout`      | `integer` | N        | Timeout in seconds to wait for the workflow run to be created after dispatch (default: 60, max: 300). |

## Output

| Name    | Type      | Description                                                                 |
| ------- | --------- | --------------------------------------------------------------------------- |
| `runID` | `integer` | The ID of the dispatched workflow run that can be used for status tracking. |

## Example

This example dispatches a deployment workflow with custom inputs.

```yaml
steps:
- uses: gha-dispatch-workflow
  as: dispatch-deployment
  config:
    credentials:
      secretName: github-credentials
    owner: myorg
    repo: my-app
    workflowFile: deploy.yml
    ref: main
    inputs:
      environment: "${{ ctx.stage }}"
      image_tag: "${{ imageFrom(vars.imageRepo).Tag }}"
      promotion_id: "${{ ctx.promotion }}"
      deploy_version: "${{ imageFrom(vars.imageRepo).Tag }}"
    timeout: 120
```
