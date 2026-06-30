---
sidebar_label: gh-delete-comment
description: Deletes a comment from a GitHub issue or pull request, typically used in failure cleanup steps.
---

# `gh-delete-comment`

<span class="tag professional"></span>
<span class="tag beta"></span>

:::info
This promotion step is only available in Kargo on the
[Akuity Platform](https://akuity.io/akuity-platform), versions v1.11.0 and above.
:::

The `gh-delete-comment` step removes a comment from a GitHub issue or pull
request. It is typically used in `if: ${{ failure() }}` cleanup steps to remove
progress comments that were posted by an earlier
[`gh-add-comment`](./gh-add-comment.md) step in the same stage.

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

| Name                    | Type      | Required | Description                                                                                                                                             |
| ----------------------- | --------- | -------- | ------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `repoURL`               | `string`  | Y        | The URL of the GitHub repository (e.g. `https://github.com/owner/repo`).                                                                               |
| `insecureSkipTLSVerify` | `boolean` | N        | If `true`, TLS verification of the GitHub server certificate is skipped. Use only for GitHub Enterprise Server instances with self-signed certificates. |
| `commentID`             | `integer` | Y        | The ID of the comment to delete, as returned by `gh-add-comment`.                                                                                      |

## Output

This step does not produce any output.

## Example

This example deletes a progress comment if the promotion fails:

```yaml
steps:
- as: post-comment
  uses: gh-add-comment
  config:
    repoURL: https://github.com/myorg/myrepo
    issueNumber: ${{ freightMetadata(ctx.targetFreight.name)['github-issue-number'] }}
    body: "Promotion to **${{ ctx.stage }}** is in progress..."

# ... your promotion steps ...

- uses: gh-delete-comment
  if: ${{ failure() && status('post-comment') == 'Succeeded' }}
  config:
    repoURL: https://github.com/myorg/myrepo
    commentID: ${{ outputs['post-comment'].commentID }}
```
