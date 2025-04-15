---
sidebar_label: compose-output
description: Composes output from one or more steps into new output.
---

# `compose-output`

`compose-output` is a step that composes a new output from one or more existing
outputs. This step can be useful when subsequent steps need to reference a
combination of outputs from previous steps, or to allow a
[`PromotionTask`](../20-promotion-tasks.md) to provide easy access to outputs from
the steps it contains.

## Configuration

The `compose-output` step accepts an arbitrary set of key-value pairs, where the
key is the name of the output to be created and the value is arbitrary and can
be an [Expression Language](../40-expressions.md) expression.

## Output

The dynamic outputs of the `compose-output` step are the outputs that it composes
according to its configuration.

## Examples

### Compose a Pull Request Link

In this example, a pull request link is constructed and sent to Slack. First, a
pull request is opened using the [`git-open-pr` step](git-open-pr.md). Then, the
`compose-output` step creates a new output named URL by combining the repository
URL with the PR number to form a complete PR link. Finally, the
[`http` step](http.md) uses this composed URL to send a formatted message to a
Slack channel using Slack's API.

```yaml
vars:
- name: repoURL
  value: https://github.com/example/repo
steps:
- uses: git-open-pr
  as: open-pr
  config:
    repoURL: ${{ vars.repoURL }}
    createTargetBranch: true
    sourceBranch: ${{ outputs.push.branch }}
    targetBranch: stage/${{ ctx.stage }}
- uses: compose-output
  as: pr-link
  config:
    url: ${{ vars.repoURL }}/pull/${{ outputs['open-pr'].prNumber }}
- uses: http
  config:
    method: POST
    url: https://slack.com/api/chat.postMessage
    headers:
    - name: Authorization
      value: Bearer ${{ secrets.slack.token }}
    - name: Content-Type
      value: application/json
    body: |
      ${{ quote({
        "channel": "C123456",
        "blocks": [
          {
            "type": "section",
            "text": {
              "type": "mrkdwn",
              "text": "A new PR has been opened: ${{ task.outputs['pr-link'].url }}"
            }
          }
        ]
      }) }}
```
