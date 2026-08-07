---
sidebar_label: gh-search-issues
description: Fetches a single GitHub issue by number, or searches issues in a repository using a query string.
---

# `gh-search-issues`

<span class="tag professional"></span>
<span class="tag beta"></span>

:::info
This promotion step is only available in Kargo on the
[Akuity Platform](https://akuity.io/akuity-platform), versions v1.11.0 and above.
:::

The `gh-search-issues` step has two mutually exclusive modes, selected by
providing either `issueNumber` or `query`:

- **Fetch by number** — retrieves a single known issue.
- **Search by query** — searches issues in the repository and returns all
  matching results.

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

The GitHub token must have **Issues: Read** access for the repository (or the
`repo` scope for a classic personal access token).

## Output

Both modes return the same output shape:

| Name     | Type           | Description                                                                                                                    |
| -------- | -------------- | ------------------------------------------------------------------------------------------------------------------------------ |
| `issues` | `[]object`     | Array of matching issues. Each object contains `number` (integer), `title`, `body`, `state`, `labels` ([]string), `assignees` ([]string), and `url`. |

---

## Fetch by Number

Retrieves a specific issue by its number. `issueNumber` and `query` are
mutually exclusive — only one may be set.

### Configuration

| Name                    | Type      | Required | Description                                                                                                                                             |
| ----------------------- | --------- | -------- | ------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `repoURL`               | `string`  | Y        | The URL of the GitHub repository (e.g. `https://github.com/owner/repo`).                                                                               |
| `insecureSkipTLSVerify` | `boolean` | N        | If `true`, TLS verification of the GitHub server certificate is skipped. Use only for GitHub Enterprise Server instances with self-signed certificates. |
| `issueNumber`           | `integer` | Y        | The number of the issue to fetch.                                                                                                                       |

### Example

```yaml
steps:
- as: fetch-issue
  uses: gh-search-issues
  config:
    repoURL: https://github.com/myorg/myrepo
    issueNumber: ${{ freightMetadata(ctx.targetFreight.name)['github-issue-number'] }}

- uses: gh-issue-add-comment
  config:
    repoURL: https://github.com/myorg/myrepo
    issueNumber: ${{ outputs['fetch-issue'].issues[0].number }}
    body: "Current state: **${{ outputs['fetch-issue'].issues[0].state }}**"
```

---

## Search by Query

Searches issues in the repository using a
[GitHub issue search query](https://docs.github.com/en/search-github/searching-on-github/searching-issues-and-pull-requests).
The search is automatically scoped to the repository specified in `repoURL` —
do not include a `repo:` qualifier in the query.

`issueNumber` and `query` are mutually exclusive — only one may be set.

:::info

Query mode uses the [GitHub Search API](https://docs.github.com/en/rest/search/search#search-issues-and-pull-requests),
which is backed by an ElasticSearch index that is updated asynchronously.
Issues created or modified within the last ~60 seconds may not appear in
results yet. For steps in the same promotion stage that just created or
modified an issue, use `issueNumber` mode instead — it reads directly from
the GitHub REST API and is immediately consistent. Query mode is well-suited
for downstream stages, where the issue was created in an earlier stage run
and the index has had time to catch up.

:::

:::note

Query mode returns at most **30 results**. 

:::

### Configuration

| Name                    | Type      | Required | Description                                                                                                                                                                                                                                               |
| ----------------------- | --------- | -------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `repoURL`               | `string`  | Y        | The URL of the GitHub repository (e.g. `https://github.com/owner/repo`). Determines both which repository to search and which credential to use.                                                                                                          |
| `insecureSkipTLSVerify` | `boolean` | N        | If `true`, TLS verification of the GitHub server certificate is skipped. Use only for GitHub Enterprise Server instances with self-signed certificates.                                                                                                    |
| `query`                 | `string`  | Y        | GitHub issue search query (e.g. `is:open label:bug`). The query is automatically scoped to the repository in `repoURL`. Do not include a `repo:` qualifier. See the [GitHub search syntax](https://docs.github.com/en/search-github/searching-on-github/searching-issues-and-pull-requests) for supported filters. |

### Example

This example searches for open issues labeled `release` to find all active
release tracking issues in the repository:

```yaml
steps:
- as: find-release-issues
  uses: gh-search-issues
  config:
    repoURL: https://github.com/myorg/myrepo
    query: "is:open label:release"

- uses: gh-issue-add-comment
  if: ${{ len(outputs['find-release-issues'].issues) > 0 }}
  config:
    repoURL: https://github.com/myorg/myrepo
    issueNumber: ${{ outputs['find-release-issues'].issues[0].number }}
    body: "Promotion to **${{ ctx.stage }}** completed."
```
