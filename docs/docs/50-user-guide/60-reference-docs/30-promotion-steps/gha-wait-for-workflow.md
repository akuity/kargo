---
sidebar_label: gha-wait-for-workflow
description: Waits for GitHub Actions workflow runs to complete with optional status validation.
---

# `gha-wait-for-workflow`

<span class="tag professional"></span>
<span class="tag beta"></span>

:::info
This promotion step is only available in Kargo on the [Akuity Platform](https://akuity.io/akuity-platform), versions v1.8 and above.
:::

The `gha-wait-for-workflow` promotion step provides integration with GitHub Actions, allowing you to wait for workflow runs to complete and optionally validate their conclusion status. This is particularly useful for ensuring that CI/CD pipelines, tests, or deployment scripts complete successfully before proceeding with subsequent promotion steps.

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
- `actions:read` - To read workflow run status

**Classic Personal Access Token:**
- `repo` - read/write access
:::

## Configuration

| Name                 | Type      | Required | Description                                                                                |
| -------------------- | --------- | -------- | ------------------------------------------------------------------------------------------ |
| `owner`              | `string`  | Y        | The owner of the repository (user or organization).                                        |
| `repo`               | `string`  | Y        | The name of the repository.                                                                |
| `runID`              | `integer` | Y        | The workflow run ID to wait for.                                                           |
| `expectedConclusion` | `string`  | N        | The expected final conclusion status. If not provided, conclusion status is not validated. |

Valid values for `expectedConclusion`:
- `success` - The workflow completed successfully
- `failure` - The workflow failed
- `cancelled` - The workflow was cancelled
- `skipped` - The workflow was skipped
- `timed_out` - The workflow timed out
- `action_required` - The workflow requires manual action
- `neutral` - The workflow completed with neutral status
- `stale` - The workflow is stale

## Output

| Name         | Type     | Description                                                       |
| ------------ | -------- | ----------------------------------------------------------------- |
| `conclusion` | `string` | The final conclusion status of the workflow run after completion. |

## Example

This example waits for a previously dispatched workflow to complete successfully.

```yaml
steps:
- uses: gha-wait-for-workflow
  config:
    credentials:
      secretName: github-credentials
    owner: myorg
    repo: my-app
    runID: "${{ outputs['dispatch-deployment'].runID }}"
    expectedConclusion: success
```

## Multi-Step Workflow Example

This example shows how to combine `gha-dispatch-workflow` and `gha-wait-for-workflow` steps:

```yaml
steps:
# Dispatch the workflow
- uses: gha-dispatch-workflow
  as: dispatch-deployment
  config:
    credentials:
      secretName: github-credentials
    owner: gdsoumya
    repo: git-test
    workflowFile: test.yaml
    ref: master
    inputs:
      greeting: "hola"
    timeout: 120

# Wait for the workflow to complete
- uses: gha-wait-for-workflow
  config:
    credentials:
      secretName: github-credentials
    owner: gdsoumya
    repo: git-test
    runID: "${{ outputs['dispatch-deployment'].runID }}"
    expectedConclusion: success

# Continue with other promotion steps after workflow completes...
```
