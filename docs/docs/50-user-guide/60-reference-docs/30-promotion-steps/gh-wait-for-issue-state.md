---
sidebar_label: gh-wait-for-issue-state
description: Blocks a promotion stage until an expression evaluated against a GitHub issue succeeds, or fails the step immediately if a failure expression matches.
---

# `gh-wait-for-issue-state`

<span class="tag professional"></span>
<span class="tag beta"></span>

:::info
This promotion step is only available in Kargo on the
[Akuity Platform](https://akuity.io/akuity-platform), versions v1.11.0 and above.
:::

The `gh-wait-for-issue-state` step polls a GitHub issue and holds the
promotion until a `successExpression` evaluates to `true`. An optional
`failureExpression`, checked on every poll before `successExpression`, fails
the step immediately instead of continuing to wait.

This is useful for human-in-the-loop approval workflows where a reviewer
signals readiness — or rejection — by applying a label or closing the issue.

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

## The `issue` variable

Both `successExpression` and `failureExpression` are evaluated with the issue
available as `issue`, with the following fields:

| Field       | Type       | Description                                    |
| ----------- | ---------- | ----------------------------------------------- |
| `number`    | `integer`  | The issue number.                              |
| `title`     | `string`   | The issue title.                               |
| `body`      | `string`   | The issue body.                                |
| `state`     | `string`   | `open` or `closed`.                            |
| `labels`    | `[]string` | Label names currently applied to the issue.    |
| `assignees` | `[]string` | Login names of assigned users.                 |
| `url`       | `string`   | The issue's HTML URL.                          |

`labels` and `assignees` are arrays of plain strings, not objects — check for
a label with the `in` operator (`"approved" in issue.labels`), not by
filtering on a `.name` field.

## Configuration

| Name                    | Type      | Required | Description                                                                                                                                                                         |
| ----------------------- | --------- | -------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `repoURL`               | `string`  | Y        | The URL of the GitHub repository (e.g. `https://github.com/owner/repo`).                                                                                                           |
| `insecureSkipTLSVerify` | `boolean` | N        | If `true`, TLS verification of the GitHub server certificate is skipped. Use only for GitHub Enterprise Server instances with self-signed certificates.                             |
| `issueNumber`           | `integer` | Y        | The number of the issue to watch.                                                                                                                                                   |
| `successExpression`     | `string`  | Y        | An expression evaluated against `issue` on every poll. The step succeeds once this evaluates to `true`.                                                                            |
| `failureExpression`     | `string`  | N        | An expression evaluated against `issue` on every poll, before `successExpression`. If it evaluates to `true`, the step fails immediately instead of continuing to poll.            |
| `pollInterval`          | `string`  | N        | How often to check the issue state, specified as a [Go duration string](https://pkg.go.dev/time#ParseDuration) (e.g., `30s`, `5m`, `1.5h`). Overrides the default controller reconciliation interval when set. |

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
    successExpression: '"approved" in issue.labels'
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
    successExpression: 'issue.state == "closed"'
```

### Wait for both label and closed state

Combine conditions with `and`:

```yaml
steps:
- uses: gh-wait-for-issue-state
  config:
    repoURL: https://github.com/myorg/myrepo
    issueNumber: ${{ freightMetadata(ctx.targetFreight.name)['github-issue-number'] }}
    successExpression: 'issue.state == "closed" and "approved" in issue.labels'
```

### Fail fast on rejection

Use `failureExpression` so a reviewer can reject immediately instead of the
promotion waiting for `pollInterval`/timeout to elapse:

```yaml
steps:
- uses: gh-wait-for-issue-state
  config:
    repoURL: https://github.com/myorg/myrepo
    issueNumber: ${{ freightMetadata(ctx.targetFreight.name)['github-issue-number'] }}
    successExpression: '"approved" in issue.labels'
    failureExpression: '"rejected" in issue.labels'
    pollInterval: 30s
```

### Match against multiple labels

`successExpression`/`failureExpression` accept any expr-lang expression, so
matching against more than one label doesn't require a different shape:

```yaml
steps:
- uses: gh-wait-for-issue-state
  config:
    repoURL: https://github.com/myorg/myrepo
    issueNumber: ${{ freightMetadata(ctx.targetFreight.name)['github-issue-number'] }}
    successExpression: 'any(issue.labels, {# in ["approved", "lgtm"]})'
    failureExpression: 'any(issue.labels, {# in ["rejected", "blocked"]})'
```