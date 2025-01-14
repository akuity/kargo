---
sidebar_label: Promotion Tasks Reference
description: Learn about Kargo's promotion tasks that can define reusable promotion steps.
---

# Promotion Tasks Reference

`PromotionTask`s allow you to define a set of
[Promotion Steps](./10-promotion-steps/index.md) on a project or global
(`ClusterPromotionTask`) level that can be reused across multiple
[Promotion Templates](../30-how-to-guides/14-working-with-stages.md#promotion-templates).

## Defining a Promotion Task

A simple `PromotionTask` is defined as follows:

```yaml
apiVersion: kargo.akuity.io/v1alpha1
kind: PromotionTask
metadata:
  name: open-pr-and-wait
  namespace: kargo-demo
spec:
  vars:
  - name: repoURL
  - name: sourceBranch
  - name: targetBranch
  steps:
  - uses: git-open-pr
    as: open-pr
    config:
      repoURL: ${{ vars.repoURL }}
      createTargetBranch: true
      sourceBranch: ${{ vars.sourceBranch }}
      targetBranch: ${{ vars.targetBranch }}
  - uses: git-wait-for-pr
    as: wait-for-pr
    config:
      repoURL: ${{ vars.repoURL }}
      prNumber: ${{ task.outputs['open-pr'].prNumber }}
  - uses: compose-output
    as: output
    config:
      mergeCommit: ${{ task.outputs['wait-for-pr'].commit }}
```

### Promotion Task Variables

The `spec.vars` section of a `PromotionTask` defines the input variables that
it expects when it is used in a Promotion Template. Each variable requires a
`name` and optionally a default `value`. When the `value` is not provided, the
variable is considered required and must be provided when the `PromotionTask`
is used in a Promotion Template.

```yaml
apiVersion: kargo.akuity.io/v1alpha1
kind: PromotionTask
# ...omitted for brevity
spec:
  vars:
  # This variable is required
  - name: repoURL
  # This variable is optional and defaults to "main"
  - name: targetBranch
    value: main
```

Variables can be referenced in the `PromotionTask` steps' configuration using
the `${{ vars.<variable-name> }}` syntax. For example, the `repoURL` variable
is referenced as `${{ vars.repoURL }}`.

```yaml
apiVersion: kargo.akuity.io/v1alpha1
kind: PromotionTask
# ...omitted for brevity
spec:
  vars:
  - name: repoURL
  - name: sourceBranch
  - name: targetBranch
    value: main
  steps:
  - uses: git-open-pr
    as: open-pr
    config:
      repoURL: ${{ vars.repoURL }}
      createTargetBranch: true
      sourceBranch: ${{ vars.sourceBranch }}
      targetBranch: ${{ vars.targetBranch }}
```

When the `PromotionTask` is used in a Promotion Template, the input variables
must be provided as `vars` of the step referencing the `PromotionTask`.

```yaml
apiVersion: kargo.akuity.io/v1alpha1
kind: Stage
# ...omitted for brevity
spec:
  promotionTemplate:
    spec:
      steps:
      - task:
          name: my-promotion-task
          vars:
          - name: repoURL
            value: https://github.com/example/repository.git
          - name: sourceBranch
            value: feature-branch
```

When the Promotion Template defines a
[`vars` section](../30-how-to-guides/14-working-with-stages.md#promotion-templates)
the variables are inherited by the `PromotionTask` and do not require redefinition
unless they need to be overridden.

```yaml
apiVersion: kargo.akuity.io/v1alpha1
kind: Stage
# ...omitted for brevity
spec:
  promotionTemplate:
    spec:
      vars:
      - name: repoURL
        value: https://github.com/example/repository.git
      steps:
        - task:
            name: my-promotion-task
            vars:
            - name: sourceBranch
              value: feature-branch
```

### Promotion Task Steps

The `spec.steps` section of a `PromotionTask` define the sequence of steps that
are inflated when a `Promotion` is created from a Promotion Template that
references the task. Each step works the as a [regular step](10-promotion-steps/index.md)
in a Promotion Template, except that references to other tasks are not allowed.

#### Promotion Task Context

The steps of a `PromotionTask` have access to an additional `task`
[pre-defined variable](20-expression-language.md#pre-defined-variables) that
provides access to the `outputs` of the previous steps. The `task.outputs`
property is a map of step aliases to their outputs.

```yaml
apiVersion: kargo.akuity.io/v1alpha1
kind: PromotionTask
# ...omitted for brevity
spec:
  steps:
  - uses: git-open-pr
    as: open-pr
    config:
      repoURL: ${{ vars.repoURL }}
      createTargetBranch: true
      sourceBranch: ${{ vars.sourceBranch }}
      targetBranch: ${{ vars.targetBranch }}
  - uses: git-wait-for-pr
    as: wait-for-pr
    config:
      repoURL: ${{ vars.repoURL }}
      prNumber: ${{ task.outputs['open-pr'].prNumber }}
```

### Promotion Task Outputs

Outputs of a `PromotionTask` can be made more accessible by defining them using
a [`compose-output` step](10-promotion-steps/70-compose-output.md). The outputs
are then made available under the alias defined in the
[`as` field](10-promotion-steps/index.md#step-aliases) of the step referencing the
`PromotionTask`.

```yaml
---
apiVersion: kargo.akuity.io/v1alpha1
kind: PromotionTask
metadata:
  name: open-pr-and-wait
  namespace: kargo-demo
spec:
    # ...omitted for brevity
    - uses: compose-output
      as: output
      config:
        mergeCommit: ${{ task.outputs['wait-for-pr'].commit }}
---
apiVersion: kargo.akuity.io/v1alpha1
kind: Stage
# ...omitted for brevity
spec:
  promotionTemplate:
    spec:
      steps:
      - task:
          name: open-pr-and-wait
          as: pr
          # ...additional configuration
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
                    "text": "A new commit was merged: ${{ outputs.pr.mergeCommit }}"
                  }
                }
              ]
            }) }}
```

### Cluster Promotion Task

A `ClusterPromotionTask` is a `PromotionTask` that is available to all
[projects](../30-how-to-guides/11-working-with-projects.md)
in the cluster. The `ClusterPromotionTask` is defined the same way as a
`PromotionTask`, but without the `namespace` field in the metadata.

```yaml
apiVersion: kargo.akuity.io/v1alpha1
kind: ClusterPromotionTask
metadata:
  name: open-pr-and-wait
spec:
  # ...equivalent to a PromotionTask
```

A `ClusterPromotionTask` can be used in a Promotion Template the same way as a
`PromotionTask`, but requires the additional specification of the `kind` field
in the step referencing the task.

```yaml
apiVersion: kargo.akuity.io/v1alpha1
kind: Stage
# ...omitted for brevity
spec:
  promotionTemplate:
    spec:
      steps:
      - kind: ClusterPromotionTask
        name: open-pr-and-wait
      # ...additional configuration
```
