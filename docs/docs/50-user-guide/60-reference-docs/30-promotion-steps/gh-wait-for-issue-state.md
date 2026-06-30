---
sidebar_label: gh-wait-for-issue-state
description: Blocks a promotion stage until a GitHub issue reaches a specified state or has a specific label applied.
---

# `gh-wait-for-issue-state`

<span class="tag professional"></span>
<span class="tag beta"></span>

:::info
This promotion step is only available in Kargo on the
[Akuity Platform](https://akuity.io/akuity-platform), versions v1.11.0 and above.
:::

The `gh-wait-for-issue-state` step polls a GitHub issue and holds the
promotion until the issue meets a specified condition: a target state
(`open` or `closed`) and/or the presence of a named label. At least one of
`state` or `label` must be provided.

This is useful for human-in-the-loop approval workflows where a reviewer
signals readiness by closing an issue or applying a label.

GitHub Issues integration for Kargo is a group of promotion steps:

1. [gh-add-comment](./gh-add-comment.md)
2. [gh-create-issue](./gh-create-issue.md)
3. [gh-delete-comment](./gh-delete-comment.md)
4. [gh-search-issues](./gh-search-issues.md)
5. [gh-update-comment](./gh-update-comment.md)
6. [gh-update-issue](./gh-update-issue.md)
7. [gh-wait-for-issue-state](./gh-wait-for-issue-state.md)

## Credentials

These steps use the same
[repository credentials](../../50-security/30-managing-secrets.md#repository-credentials)
that [`git-clone`](./git-clone.md) and [`git-open-pr`](./git-open-pr.md) use
for the same repository. If you have already configured a Git credential for
the `repoURL`, no additional setup is required.

The GitHub token must have **Issues: Read** access for the repository (or the
`repo` scope for a classic personal access token).

## Configuration

| Name                    | Type      | Required | Description                                                                                                                                                                         |
| ----------------------- | --------- | -------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `repoURL`               | `string`  | Y        | The URL of the GitHub repository (e.g. `https://github.com/owner/repo`).                                                                                                           |
| `insecureSkipTLSVerify` | `boolean` | N        | If `true`, TLS verification of the GitHub server certificate is skipped. Use only for GitHub Enterprise Server instances with self-signed certificates.                             |
| `issueNumber`           | `integer` | Y        | The number of the issue to watch.                                                                                                                                                   |
| `state`                 | `string`  | N        | Wait until the issue is in this state. Must be `open` or `closed`. At least one of `state` or `label` is required.                                                                 |
| `label`                 | `string`  | N        | Wait until the issue has this label. At least one of `state` or `label` is required.                                                                                               |
| `pollInterval`          | `string`  | N        | How often to check the issue state, specified as a [Go duration string](https://pkg.go.dev/time#ParseDuration) (e.g., `30s`, `5m`). Overrides the default controller reconciliation interval when set. |

## Output

This step does not produce any output.

## Examples

### Wait for label

Block promotion until a reviewer applies the `approved` label to signal
readiness:

```yaml
steps:
- uses: gh-wait-for-issue-state
  config:
    repoURL: https://github.com/myorg/myrepo
    issueNumber: ${{ freightMetadata(ctx.targetFreight.name)['github-issue-number'] }}
    label: approved
    pollInterval: 2m

# Promotion continues once the label is present
- uses: argocd-update
  config:
    apps:
    - name: prod-app
      namespace: argocd
```

### Wait for closed state

Block promotion until the issue is closed:

```yaml
steps:
- uses: gh-wait-for-issue-state
  config:
    repoURL: https://github.com/myorg/myrepo
    issueNumber: ${{ freightMetadata(ctx.targetFreight.name)['github-issue-number'] }}
    state: closed
```

### Wait for both label and closed state

Both conditions must be true simultaneously:

```yaml
steps:
- uses: gh-wait-for-issue-state
  config:
    repoURL: https://github.com/myorg/myrepo
    issueNumber: ${{ freightMetadata(ctx.targetFreight.name)['github-issue-number'] }}
    state: closed
    label: approved
```
