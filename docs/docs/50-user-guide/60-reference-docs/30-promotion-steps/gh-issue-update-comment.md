---
sidebar_label: gh-issue-update-comment
description: Replaces the body of an existing GitHub issue or pull request comment.
---

# `gh-issue-update-comment`

<span class="tag professional"></span>
<span class="tag beta"></span>

:::info
This promotion step is only available in Kargo on the
[Akuity Platform](https://akuity.io/akuity-platform), versions v1.11.0 and above.
:::

The `gh-issue-update-comment` step replaces the body of an existing comment on a
GitHub issue or pull request. The `commentID` must come from a previous
[`gh-issue-add-comment`](./gh-issue-add-comment.md) step in the same stage.

GitHub Issues integration for Kargo is a group of promotion steps:

1. [gh-issue-add-comment](./gh-issue-add-comment.md)
2. [gh-create-issue](./gh-create-issue.md)
3. [gh-issue-delete-comment](./gh-issue-delete-comment.md)
4. [gh-search-issues](./gh-search-issues.md)
5. [gh-issue-update-comment](./gh-issue-update-comment.md)
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

| Name                    | Type      | Required | Description                                                                                                                                             |
| ----------------------- | --------- | -------- | ------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `repoURL`               | `string`  | Y        | The URL of the GitHub repository (e.g. `https://github.com/owner/repo`).                                                                               |
| `insecureSkipTLSVerify` | `boolean` | N        | If `true`, TLS verification of the GitHub server certificate is skipped. Use only for GitHub Enterprise Server instances with self-signed certificates. |
| `commentID`             | `integer` | Y        | The ID of the comment to update, as returned by `gh-issue-add-comment`.                                                                                      |
| `body`                  | `string`  | Y        | The new body text of the comment. Replaces the existing body. Supports GitHub Flavored Markdown.                                                        |

## Output

| Name        | Type       | Description                                                                                             |
| ----------- | ---------- | ------------------------------------------------------------------------------------------------------- |
| `commentID` | `integer`  | The ID of the updated comment (same value passed as input, echoed for convenience).                     |
| `url`       | `string`   | The HTML URL of the updated comment.                                                                    |

## Example

This example posts a "promotion started" comment and then updates it with the
final outcome:

```yaml
steps:
- as: post-comment
  uses: gh-issue-add-comment
  config:
    repoURL: https://github.com/myorg/myrepo
    issueNumber: ${{ freightMetadata(ctx.targetFreight.name)['github-issue-number'] }}
    body: "Promotion to **${{ ctx.stage }}** is in progress..."

# ... your promotion steps ...

- uses: gh-issue-update-comment
  config:
    repoURL: https://github.com/myorg/myrepo
    commentID: ${{ outputs['post-comment'].commentID }}
    body: "Promotion to **${{ ctx.stage }}** completed successfully."
```
