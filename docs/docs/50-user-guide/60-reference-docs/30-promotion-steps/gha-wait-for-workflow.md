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

:::warning[Breaking Changes in v1.9]

Starting with Kargo v1.9, this promotion step has significant configuration changes:

- The `credentials` field has been removed
- `owner` and `repo` fields have been replaced with a single `repoURL` field
- Credentials are now inherited from Git repository credentials configured in Kargo
- A new optional `insecureSkipTLSVerify` field has been added

See the [Migration Guide](#migration-from-v18-to-v19) below for details.

:::

The `gha-wait-for-workflow` promotion step provides integration with GitHub Actions, allowing you to wait for workflow runs to complete and optionally validate their conclusion status. This is particularly useful for ensuring that CI/CD pipelines, tests, or deployment scripts complete successfully before proceeding with subsequent promotion steps.

## Credentials Configuration

### v1.9 and Above

Starting with Kargo v1.9, credentials are inherited from Git repository credentials configured in Kargo. The Git repository credentials must be configured with either:

- **GitHub Personal Access Token (PAT)** - Classic or fine-grained
- **GitHub App credentials** - App ID, installation ID, and private key

:::info Required Permissions

The GitHub credentials must have the following permissions:

**Fine-grained Personal Access Token:**
- `actions:read` - To read workflow run status

**Classic Personal Access Token:**
- `repo` - read/write access

**GitHub App:**
- `actions:read` - To read workflow run status

:::

### v1.8 (Deprecated)

:::warning[Removed in v1.9]

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

| Name                    | Type      | Required | Description                                                                                |
| ----------------------- | --------- | -------- | ------------------------------------------------------------------------------------------ |
| `repoURL`               | `string`  | Y        | The GitHub repository URL where the workflow resides (e.g., 'https://github.com/owner/repo'). |
| `runID`                 | `integer` | Y        | The workflow run ID to wait for. Can be a direct ID or a reference to freight metadata.   |
| `expectedConclusion`    | `string`  | N        | The expected final conclusion status. If not provided, conclusion status is not validated. |
| `insecureSkipTLSVerify` | `boolean` | N        | Skip TLS verification when communicating with the GitHub API (default: false).            |

### v1.8 (Deprecated)

:::warning Changed in v1.9

The following configuration format has been changed in v1.9.

:::

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

### v1.9 and Above

This example waits for a previously dispatched workflow to complete successfully using the new configuration format.

```yaml
steps:
- uses: gha-wait-for-workflow
  config:
    repoURL: https://github.com/myorg/my-app
    runID: "${{ outputs['dispatch-deployment'].runID }}"
    expectedConclusion: success
    insecureSkipTLSVerify: false
```

### v1.8 (Deprecated)

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

This example shows how to combine `gha-dispatch-workflow` and `gha-wait-for-workflow` steps using the new configuration format:

```yaml
steps:
# Dispatch the workflow
- uses: gha-dispatch-workflow
  as: dispatch-deployment
  config:
    repoURL: https://github.com/example/test
    workflowFile: test.yaml
    ref: master
    inputs:
      greeting: "hola"
    timeout: 120

# Wait for the workflow to complete
- uses: gha-wait-for-workflow
  config:
    repoURL: https://github.com/example/test
    runID: "${{ outputs['dispatch-deployment'].runID }}"
    expectedConclusion: success

# Continue with other promotion steps after workflow completes...
```

## Migration from v1.8 to v1.9

To migrate from v1.8 to v1.9:

1. **Remove the `credentials` field** - Credentials are now inherited from Git repository credentials
2. **Replace `owner` and `repo` fields** with a single `repoURL` field:
   - Old: `owner: example` and `repo: test`
   - New: `repoURL: https://github.com/example/test`
3. **Configure Git repository credentials** in Kargo with GitHub PAT or GitHub App credentials
4. **Optionally add `insecureSkipTLSVerify`** if you need to skip TLS verification

### Before (v1.8)

```yaml
config:
  credentials:
    secretName: github-credentials
  owner: example
  repo: test
  runID: 12345
  expectedConclusion: success
```

### After (v1.9)

```yaml
config:
  repoURL: https://github.com/example/test
  runID: 12345
  expectedConclusion: success
```
