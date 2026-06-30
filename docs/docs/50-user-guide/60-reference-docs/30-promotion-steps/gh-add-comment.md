---
sidebar_label: gh-add-comment
description: Adds a comment to an existing GitHub issue or pull request and returns the comment ID for later updates or deletion.
---

# `gh-add-comment`

<span class="tag professional"></span>
<span class="tag beta"></span>

:::info
This promotion step is only available in Kargo on the
[Akuity Platform](https://akuity.io/akuity-platform), versions v1.11.0 and above.
:::

The `gh-add-comment` step posts a comment on an existing GitHub issue or pull
request. The returned `commentID` can be passed to
[`gh-update-comment`](./gh-update-comment.md) or
[`gh-delete-comment`](./gh-delete-comment.md) in later steps.

:::note

In GitHub, pull requests share the same number space as issues. Passing a PR
number as `issueNumber` posts a comment on the PR's conversation thread. The
`url` in the step output automatically points to the PR (`/pull/`) rather than
the issue (`/issues/`), because GitHub's API returns the correct URL based on
whether the number belongs to a PR or an issue.

:::

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
| `issueNumber`           | `integer` | Y        | The number of the issue or pull request to comment on. Pull requests and issues share the same number space in GitHub, so this field works for both.    |
| `body`                  | `string`  | Y        | The body text of the comment. Supports GitHub Flavored Markdown.                                                                                        |

## Output

| Name        | Type     | Description                                                     |
| ----------- | -------- | --------------------------------------------------------------- |
| `commentID` | `int`    | The ID of the created comment.                                  |
| `url`       | `string` | The HTML URL of the created comment.                            |

## Examples

### Comment on an issue

Posts a comment when a promotion starts and removes it if the promotion fails:

```yaml
steps:
- as: post-comment
  uses: gh-add-comment
  config:
    repoURL: https://github.com/myorg/myrepo
    issueNumber: ${{ freightMetadata(ctx.targetFreight.name)['github-issue-number'] }}
    body: |
      Promotion to **${{ ctx.stage }}** started.
      Image: `${{ imageFrom(vars.imageRepo).RepoURL }}:${{ imageFrom(vars.imageRepo).Tag }}`

# ... your promotion steps ...

- uses: gh-delete-comment
  if: ${{ failure() && status('post-comment') == 'Succeeded' }}
  config:
    repoURL: https://github.com/myorg/myrepo
    commentID: ${{ outputs['post-comment'].commentID }}
```

### Comment on a pull request

Posts a deployment status comment directly on the PR that triggered the
promotion. The PR number is stored in freight metadata by an upstream step:

```yaml
steps:
- uses: gh-add-comment
  config:
    repoURL: https://github.com/myorg/myrepo
    issueNumber: ${{ freightMetadata(ctx.targetFreight.name)['github-pr-number'] }}
    body: |
      Deployed to **${{ ctx.stage }}** successfully.
```