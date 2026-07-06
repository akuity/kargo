---
sidebar_label: gh-create-issue
description: Creates a new GitHub issue and stores the issue number in Freight metadata for use across promotion stages.
---

# `gh-create-issue`

<span class="tag professional"></span>
<span class="tag beta"></span>

:::info
This promotion step is only available in Kargo on the
[Akuity Platform](https://akuity.io/akuity-platform), versions v1.11.0 and above.
:::

The `gh-create-issue` step creates a new issue in a GitHub repository and
records the issue number in Freight metadata so that downstream stages can
reference it.

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

| Name                    | Type       | Required | Description                                                                                                                                             |
| ----------------------- | ---------- | -------- | ------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `repoURL`               | `string`   | Y        | The URL of the GitHub repository (e.g. `https://github.com/owner/repo`).                                                                               |
| `insecureSkipTLSVerify` | `boolean`  | N        | If `true`, TLS verification of the GitHub server certificate is skipped. Use only for GitHub Enterprise Server instances with self-signed certificates. |
| `title`                 | `string`   | Y        | The title of the issue.                                                                                                                                 |
| `body`                  | `string`   | N        | The body text of the issue. Supports GitHub Flavored Markdown.                                                                                          |
| `labels`                | `[]string` | N        | Labels to apply to the issue. Labels must already exist in the repository.                                                                              |
| `assignees`             | `[]string` | N        | GitHub usernames to assign to the issue. GitHub allows a maximum of 10 assignees per issue; entries beyond 10 are silently ignored.                     |
| `issueAlias`            | `string`   | N        | Override for the Freight metadata key used to store the created issue number. Defaults to `github-issue-number`. See [Issue Alias](#issue-alias).       |

## Output

| Name     | Type       | Description                                                                                             |
| -------- | ---------- | ------------------------------------------------------------------------------------------------------- |
| `number` | `integer`  | The number of the created issue (e.g. `42`).                                                            |
| `url`    | `string`   | The HTML URL of the created issue.                                                                      |

## Example

This example creates a tracking issue at the start of a promotion and closes
it when the promotion completes:

```yaml
steps:
- as: create-tracking-issue
  uses: gh-create-issue
  config:
    repoURL: https://github.com/myorg/myrepo
    title: "Promote ${{ imageFrom(vars.imageRepo).Tag }} to ${{ ctx.stage }}"
    body: |
      Automated promotion triggered by Kargo.

      - **Stage:** ${{ ctx.stage }}
      - **Image:** ${{ imageFrom(vars.imageRepo).RepoURL }}:${{ imageFrom(vars.imageRepo).Tag }}
      - **Promotion:** ${{ ctx.promotion }}
    labels:
    - promotion
    - "${{ ctx.stage }}"

# ... your promotion steps (git-clone, argocd-update, etc.) ...

- uses: gh-update-issue
  config:
    repoURL: https://github.com/myorg/myrepo
    issueNumber: ${{ outputs['create-tracking-issue'].number }}
    state: closed
```

## Issue Alias

The `issueAlias` field controls which Freight metadata key Kargo uses to store
the issue number. The default key is `github-issue-number`.

Use `issueAlias` when you need to track more than one GitHub issue across a
pipeline, or when a more descriptive key helps with readability:

```yaml
- uses: gh-create-issue
  config:
    repoURL: https://github.com/myorg/myrepo
    title: "Change request for ${{ ctx.stage }}"
    issueAlias: change-request-number
```

## Multi-Stage Workflow

When `gh-create-issue` runs in an early stage, Kargo stores the issue number in
Freight metadata. Later stages retrieve it with
[`freightMetadata()`](../40-expressions.md#freightmetadatafreightname):

```yaml
# Access the issue number stored under the default key
issueNumber: ${{ freightMetadata(ctx.targetFreight.name)['github-issue-number'] }}

# Access an issue number stored under a custom alias
issueNumber: ${{ freightMetadata(ctx.targetFreight.name)['change-request-number'] }}
```

### Example: Multi-Stage Pipeline

This example creates an issue in the `test` stage, waits for a human to label
it `approved` before promoting to `staging`, and closes it in `prod`.

```yaml
---
apiVersion: kargo.akuity.io/v1alpha1
kind: Stage
metadata:
  name: test
  namespace: my-project
spec:
  promotionTemplate:
    spec:
      vars:
      - name: imageRepo
        value: public.ecr.aws/nginx/nginx
      - name: gitRepo
        value: https://github.com/myorg/myrepo
      steps:
      - as: create-issue
        uses: gh-create-issue
        config:
          repoURL: ${{ vars.gitRepo }}
          title: "Release ${{ imageFrom(vars.imageRepo).Tag }} — ${{ ctx.stage }}"
          labels:
          - release
          - "env-${{ ctx.stage }}"

      - uses: gh-issue-add-comment
        config:
          repoURL: ${{ vars.gitRepo }}
          issueNumber: ${{ outputs['create-issue'].number }}
          body: "Promotion to **${{ ctx.stage }}** started. Freight: `${{ ctx.targetFreight.name }}`."

      - uses: argocd-update
        config:
          apps:
          - name: test-app
            namespace: argocd
            sources:
            - repoURL: ${{ vars.gitRepo }}
              kustomize:
                images:
                - repoURL: ${{ vars.imageRepo }}
                  tag: ${{ imageFrom(vars.imageRepo).Tag }}

---
apiVersion: kargo.akuity.io/v1alpha1
kind: Stage
metadata:
  name: staging
  namespace: my-project
spec:
  promotionTemplate:
    spec:
      vars:
      - name: imageRepo
        value: public.ecr.aws/nginx/nginx
      - name: gitRepo
        value: https://github.com/myorg/myrepo
      steps:
      # Block until a human adds the "approved" label to the issue
      - uses: gh-wait-for-issue-state
        config:
          repoURL: ${{ vars.gitRepo }}
          issueNumber: ${{ freightMetadata(ctx.targetFreight.name)['github-issue-number'] }}
          label: approved

      - uses: argocd-update
        config:
          apps:
          - name: staging-app
            namespace: argocd
            sources:
            - repoURL: ${{ vars.gitRepo }}
              kustomize:
                images:
                - repoURL: ${{ vars.imageRepo }}
                  tag: ${{ imageFrom(vars.imageRepo).Tag }}

      - uses: gh-update-issue
        config:
          repoURL: ${{ vars.gitRepo }}
          issueNumber: ${{ freightMetadata(ctx.targetFreight.name)['github-issue-number'] }}
          addLabels:
          - "env-staging"
          removeLabels:
          - "env-test"

---
apiVersion: kargo.akuity.io/v1alpha1
kind: Stage
metadata:
  name: prod
  namespace: my-project
spec:
  promotionTemplate:
    spec:
      vars:
      - name: imageRepo
        value: public.ecr.aws/nginx/nginx
      - name: gitRepo
        value: https://github.com/myorg/myrepo
      steps:
      - uses: argocd-update
        config:
          apps:
          - name: prod-app
            namespace: argocd
            sources:
            - repoURL: ${{ vars.gitRepo }}
              kustomize:
                images:
                - repoURL: ${{ vars.imageRepo }}
                  tag: ${{ imageFrom(vars.imageRepo).Tag }}

      - uses: gh-update-issue
        config:
          repoURL: ${{ vars.gitRepo }}
          issueNumber: ${{ freightMetadata(ctx.targetFreight.name)['github-issue-number'] }}
          state: closed
          addLabels:
          - released
          removeLabels:
          - "env-staging"
```
