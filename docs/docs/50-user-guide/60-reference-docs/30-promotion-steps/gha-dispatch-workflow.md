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

:::warning[Breaking Changes in v1.9]

Starting with Kargo v1.9, this promotion step has significant configuration changes:

- The `credentials` field has been removed
- `owner` and `repo` fields have been replaced with a single `repoURL` field
- Credentials are now inherited from Git repository credentials configured in Kargo
- A new optional `insecureSkipTLSVerify` field has been added

See the [Migration Guide](#migration-from-v18-to-v19) below for details.

:::

The `gha-dispatch-workflow` promotion step provides integration with GitHub Actions, allowing you to dispatch workflows using the `workflow_dispatch` event. This is particularly useful for triggering CI/CD pipelines, running tests, or executing deployment scripts as part of your promotion workflows.

## Credentials Configuration

### v1.9 and Above

Starting with Kargo v1.9, credentials are inherited from Git repository credentials configured in Kargo. The Git repository credentials must be configured with either:

- **GitHub Personal Access Token (PAT)** - Classic or fine-grained
- **GitHub App credentials** - App ID, installation ID, and private key

:::info Required Permissions
The GitHub credentials must have the following permissions:

**Fine-grained Personal Access Token:**

- `actions:write` - To dispatch workflows
- `actions:read` - To read workflow run status

**Classic Personal Access Token:**

- `repo` - read/write access

**GitHub App:**

- `actions:write` - To dispatch workflows
- `actions:read` - To read workflow run status
:::

### v1.8 (Deprecated)

:::warning Removed in v1.9

The following credentials configuration has been removed in v1.9. Use the new Git repository credentials model instead.

:::

All GitHub Actions operations require proper authentication credentials stored in a Kubernetes `Secret`.

| Name                     | Type     | Required | Description                                                                      |
| ------------------------ | -------- | -------- | -------------------------------------------------------------------------------- |
| `credentials.secretName` | `string` | Y        | Name of the `Secret` containing the GitHub credentials in the project namespace. |

The referenced `Secret` should contain the following keys:

- `accessToken`: GitHub personal access token or GitHub App token with appropriate permissions
- `baseURL`: (Optional) GitHub base URL for GitHub Enterprise Server
- `uploadURL`: (Optional) GitHub upload URL for GitHub Enterprise Server. Only required for GitHub Enterprise Server installations

## Configuration

### v1.9 and Above

| Name                    | Type      | Required | Description                                                                                           |
| ----------------------- | --------- | -------- | ----------------------------------------------------------------------------------------------------- |
| `repoURL`               | `string`  | Y        | The GitHub repository URL where the workflow resides (e.g., 'https://github.com/owner/repo').        |
| `workflowFile`          | `string`  | Y        | The workflow filename in .github/workflows (e.g., 'deploy.yml').                                      |
| `ref`                   | `string`  | Y        | The git reference (branch or tag) to run the workflow on.                                             |
| `inputs`                | `object`  | N        | Input parameters to pass to the workflow as defined in the workflow's `workflow_dispatch` inputs.     |
| `timeout`               | `integer` | N        | Timeout in seconds to wait for the workflow run to be created after dispatch (default: 60, max: 300). |
| `insecureSkipTLSVerify` | `boolean` | N        | Skip TLS verification when communicating with the GitHub API (default: false).                        |

### v1.8 (Deprecated)

:::warning[Removed in v1.9]

The following configuration format has been removed in v1.9.

:::

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

### v1.9 and Above

This example dispatches a deployment workflow with custom inputs using the new configuration format.

```yaml
steps:
- uses: gha-dispatch-workflow
  as: dispatch-deployment
  config:
    repoURL: https://github.com/myorg/my-app
    workflowFile: deploy.yml
    ref: main
    inputs:
      environment: "${{ ctx.stage }}"
      image_tag: "${{ imageFrom(vars.imageRepo).Tag }}"
      promotion_id: "${{ ctx.promotion }}"
      deploy_version: "${{ imageFrom(vars.imageRepo).Tag }}"
    timeout: 120
    insecureSkipTLSVerify: false
```

### v1.8 (Deprecated)

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

## Migration from v1.8 to v1.9

To migrate from v1.8 to v1.9:

1. **Remove the `credentials` field** - Credentials are now inherited from Git repository credentials
2. **Replace `owner` and `repo` fields** with a single `repoURL` field:
   - Old: `owner: myorg` and `repo: my-app`
   - New: `repoURL: https://github.com/myorg/my-app`
3. **Configure Git repository credentials** in Kargo with GitHub PAT or GitHub App credentials
4. **Optionally add `insecureSkipTLSVerify`** if you need to skip TLS verification

### Before (v1.8)

```yaml
config:
  credentials:
    secretName: github-credentials
  owner: myorg
  repo: my-app
  workflowFile: deploy.yml
  ref: main
```

### After (v1.9)

```yaml
config:
  repoURL: https://github.com/myorg/my-app
  workflowFile: deploy.yml
  ref: main
```
