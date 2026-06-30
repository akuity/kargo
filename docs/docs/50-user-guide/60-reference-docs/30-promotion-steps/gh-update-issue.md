---
sidebar_label: gh-update-issue
description: Updates the title, body, state, labels, or assignees of an existing GitHub issue.
---

# `gh-update-issue`

<span class="tag professional"></span>
<span class="tag beta"></span>

:::info
This promotion step is only available in Kargo on the
[Akuity Platform](https://akuity.io/akuity-platform), versions v1.11.0 and above.
:::

The `gh-update-issue` step modifies an existing GitHub issue. At least one
optional field (`title`, `body`, `state`, `addLabels`, `removeLabels`, or
`assignees`) must be set — the step will fail validation if only `repoURL` and
`issueNumber` are provided.

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

The GitHub token must have **Issues: Read and write** access for the repository
(or the `repo` scope for a classic personal access token).

## Configuration

| Name                    | Type       | Required | Description                                                                                                                                             |
| ----------------------- | ---------- | -------- | ------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `repoURL`               | `string`   | Y        | The URL of the GitHub repository (e.g. `https://github.com/owner/repo`).                                                                               |
| `insecureSkipTLSVerify` | `boolean`  | N        | If `true`, TLS verification of the GitHub server certificate is skipped. Use only for GitHub Enterprise Server instances with self-signed certificates. |
| `issueNumber`           | `integer`  | Y        | The number of the issue to update.                                                                                                                      |
| `title`                 | `string`   | N        | New title for the issue.                                                                                                                                |
| `body`                  | `string`   | N        | New body text for the issue. Supports GitHub Flavored Markdown.                                                                                         |
| `state`                 | `string`   | N        | Set the issue state to `open` or `closed`.                                                                                                              |
| `addLabels`             | `[]string` | N        | Labels to add to the issue. Labels must already exist in the repository.                                                                                |
| `removeLabels`          | `[]string` | N        | Labels to remove from the issue.                                                                                                                        |
| `assignees`             | `[]string` | N        | Replace the issue's assignee list with these GitHub usernames. Pass an empty list to remove all assignees.                                              |

:::info

`addLabels` and `removeLabels` are computed against the issue's current label
set, so they can be used together safely. If the same label appears in both
lists, the remove takes precedence.

:::

## Output

This step does not produce any output.

## Example

This example closes an issue and adds a `released` label when promotion to
production succeeds:

```yaml
steps:
# ... your promotion steps ...

- uses: gh-update-issue
  config:
    repoURL: https://github.com/myorg/myrepo
    issueNumber: ${{ freightMetadata(ctx.targetFreight.name)['github-issue-number'] }}
    state: closed
    addLabels:
    - released
    - "env-prod"
    removeLabels:
    - "env-staging"
```
