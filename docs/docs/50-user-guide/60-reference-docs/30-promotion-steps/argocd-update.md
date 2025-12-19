---
sidebar_label: argocd-update
description: Updates one or more Argo CD `Application` resources in various ways.
---

# `argocd-update`

`argocd-update` updates one or more Argo CD `Application` resources in various
ways. `Application`s can be selected either by exact name or by using label
selectors to match multiple `Application`s at once.

Among other scenarios, this step is useful for the common one of forcing an 
Argo CD `Application` to sync after previous steps have updated a remote branch
referenced by the `Application`. This step is commonly the last step in a
promotion process.

:::note

For an Argo CD `Application` resource to be managed by a Kargo `Stage`,
the `Application` _must_ have an annotation of the following form:

```yaml
kargo.akuity.io/authorized-stage: "<project-name>:<stage-name>"
```

Such an annotation offers proof that a user who is themselves authorized
to update the `Application` in question has consented to a specific
`Stage` updating the `Application` as well.

The following example shows how to configure an Argo CD `Application`
manifest to authorize the `test` `Stage` of the `kargo-demo` `Project`:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: kargo-demo-test
  namespace: argocd
  annotations:
    kargo.akuity.io/authorized-stage: kargo-demo:test
spec:
  # Application specifications go here
```

:::

:::info

Enforcement of Argo CD
[sync windows](https://argo-cd.readthedocs.io/en/stable/user-guide/sync_windows/)
was improved substantially in Argo CD v2.11.0. If you wish for the `argocd-update`
step to honor sync windows, you must use Argo CD v2.11.0 or later.

_Additionally, it is recommended that if a promotion process is expected to
sometimes encounter an active deny window, the `argocd-update` step should be
configured with a timeout that is at least as long as the longest expected deny
window. The step's default timeout of five minutes can be overridden using the
[`retry.timeout`](../15-promotion-templates.md#step-retries) field.

:::

## Application Selection

The `argocd-update` step supports two methods for selecting Argo CD `Application`
resources:

1. **By Name**: Specify an exact application name using the `name` field
2. **By Label Selector**: Match one or more `Application`s using the `selector`
   field with label-based criteria

These two methods are mutually exclusive — you must specify either `name` or
`selector`, but not both.

### Selecting by Name

When using the `name` field, you select a single `Application` by its exact name.
This is the traditional method and is useful when you need to update a specific
`Application`.

### Selecting by Label Selector

Label selectors allow you to match multiple `Application` resources based on
their labels. This is useful for:

- Updating multiple `Application`s across different environments simultaneously.
- Managing groups of related `Application`s (e.g., all `Application`s with label
  `team: platform`).
- Performing bulk operations like hard refreshes across a set of `Application`s.

[Label selectors](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#set-references-in-api-objects)
support two types of matching criteria:

- **`matchLabels`**: Simple key-value equality matching. All specified labels 
  must match (AND logic).
- **`matchExpressions`**: Advanced matching with operators (`In`, `NotIn`,
  `Exists`, `DoesNotExist`) for more complex selection logic.

Both criteria types can be combined in a single selector, and all criteria must
be satisfied for an `Application` to be selected.

### Validation for Multi-Application Updates

When using selectors with source updates (e.g., updating `targetRevision` or
image versions), the step performs homogeneity validation before making any
changes:

1. All selected `Application`s must have compatible source configurations.
2. The specified source updates must be applicable to all matched `Application`s.
3. If validation fails for any `Application`, the step fails immediately without 
   updating any of the selected `Application`s.

This ensures that bulk updates are applied consistently and prevents partial
updates that could leave `Application`s in inconsistent states.

:::info

When using selectors without source updates (for hard refreshes), validation is
not performed since no source changes are being made. This allows you to safely
refresh groups of `Application`s with heterogeneous configurations.

:::

## Configuration

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `apps` | `[]object` | Y | Describes Argo CD `Application` resources to update and how to update them. At least one must be specified.  |
| `apps[].name` | `string` | N | The name of the Argo CD `Application`. Mutually exclusive with `selector`. Either `name` or `selector` must be specified. __Note:__ A small technical restriction on this field is that any [expressions](../40-expressions.md) used therein are limited to accessing `ctx` and `vars` and may not access `secrets` or any Freight. This is because templates in this field are, at times, evaluated outside the context of an actual `Promotion` for the purposes of building an index. In practice, this restriction does not prove to be especially limiting. |
| `apps[].namespace` | `string` | N | The namespace of the Argo CD `Application` resource(s) to be updated. If left unspecified, the namespace will be the Kargo controller's configured default -- typically `argocd`. __Note:__ This field is subject to the same restrictions as the `name` field. See above. |
| `apps[].selector` | `object` | N | Label selector for matching one or more Argo CD `Application` resources. Mutually exclusive with `name`. Either `name` or `selector` must be specified. __Note:__ This field is subject to the same restrictions as the `name` field regarding expression usage. See above. |
| `apps[].selector.matchLabels` | `map[string]string` | N | A map of label key-value pairs. All specified labels must match for an `Application` to be selected (AND logic). At least one of `matchLabels` or `matchExpressions` must be specified. |
| `apps[].selector.matchExpressions` | `[]object` | N | A list of label selector requirements. All expressions must be satisfied for an `Application` to be selected. At least one of `matchLabels` or `matchExpressions` must be specified. |
| `apps[].selector.matchExpressions[].key` | `string` | Y | The label key that the selector applies to. |
| `apps[].selector.matchExpressions[].operator` | `string` | Y | The operator to use for matching. Valid values: `In`, `NotIn`, `Exists`, `DoesNotExist`. |
| `apps[].selector.matchExpressions[].values` | `[]string` | N | An array of string values. Required when `operator` is `In` or `NotIn`. Must be empty when `operator` is `Exists` or `DoesNotExist`. |
| `apps[].sources` | `[]object` | N | Describes Argo CD `ApplicationSource`s to update and how to update them. |
| `apps[].sources[].repoURL` | `string` | Y | The value of the target `ApplicationSource`'s  own `repoURL` field. This must match exactly. |
| `apps[].sources[].chart` | `string` | N | Applicable only when the target `ApplicationSource` references a Helm chart repository, the value of the target `ApplicationSource`'s  own `chart` field. This must match exactly. |
| `apps[].sources[].desiredRevision` | `string` | N | Specifies the desired revision for the source. i.e. The revision to which the source must be observably synced when performing a health check. Prior to v1.1.0, if left undefined, the desired revision was determined by Freight (if possible). Beginning with v1.1.0, if left undefined, Kargo will not require the source to be observably synced to any particular source to be considered healthy. Note that the source's `targetRevision` will not be updated to this revision unless `updateTargetRevision=true` is also set. |
| `apps[].sources[].updateTargetRevision` | `boolean` | Y | Indicates whether the target `ApplicationSource` should be updated such that its `targetRevision` field points directly at the desired revision. A `true` value in this field requires `desiredRevision` to be specified. |
| `apps[].sources[].kustomize` | `object` | N | Describes updates to an Argo CD `ApplicationSource`'s Kustomize-specific properties. |
| `apps[].sources[].kustomize.images` | `[]object` | Y | Describes how to update an Argo CD `ApplicationSource`'s Kustomize-specific properties to reference specific versions of container images. |
| `apps[].sources[].kustomize.images[].repoURL` | `string` | Y | URL of the image being updated. |
| `apps[].sources[].kustomize.images[].tag` | `string` | N | A tag naming a specific revision of the image specified by `repoURL`. Mutually exclusive with `digest`. One of `digest` or `tag` must be specified. |
| `apps[].sources[].kustomize.images[].digest` | `string` | N | A digest naming a specific revision of the image specified by `repoURL`. Mutually exclusive with `tag`. One of `digest` or `tag` must be specified. |
| `apps[].sources[].kustomize.images[].newName` | `string` | N | A substitution for the name/URL of the image being updated. This is useful when different Stages have access to different container image repositories (assuming those different repositories contain equivalent images that are tagged identically). This may be a frequent consideration for users of Amazon's Elastic Container Registry. |
| `apps[].sources[].helm` | `object` | N | Describes updates to an Argo CD `ApplicationSource`'s Helm parameters. |
| `apps[].sources[].helm.images` | `[]object` | Y | Describes how to update  an Argo CD `ApplicationSource`'s Helm parameters to reference specific versions of container images. |
| `apps[].sources[].helm.images[].key` | `string` | Y | The key to update within the target `ApplicationSource`'s `helm.parameters` map. See Helm documentation on the [format and limitations](https://helm.sh/docs/intro/using_helm/#the-format-and-limitations-of---set) of the notation used in this field. |
| `apps[].sources[].helm.images[].value` | `string` | Y | Specifies the new value for the key. Typically, a value from [`chartFrom()`](../40-expressions.md#chartfromrepourl-chartname-freightorigin) is used here. |

## Health Checks

The `argocd-update` step is unique among all other built-in promotion steps in
that, on successful completion, it will register health checks to be performed
upon the target `Stage` on an ongoing basis. This health check configuration is
_opaque_ to the rest of Kargo and is understood only by health check
functionality built into the step. This permits Kargo to factor the health and
sync state of Argo CD `Application` resources into the overall health of a
`Stage` without requiring Kargo to understand `Application` health directly.

:::info

Although the `argocd-update` step is the only promotion step to currently
utilize this health check framework, we anticipate that future built-in and
third-party promotion steps will take advantage of it as well.

Because of this, the health of a `Stage` is not necessarily a simple
reflection of the `Application` resource it manages. It can also be influenced
by other `Application` resources that are updated by other promotion steps,
or by a `Promotion` which failed to complete successfully.

:::

## Examples

### Common Usage

```yaml
steps:
# Clone, render manifests, commit, push, etc...
- uses: git-commit
  as: commit
  config:
    path: ./out
    message: ${{ outputs['update-image'].commitMessage }}
- uses: git-push
  config:
    path: ./out
- uses: argocd-update
  config:
    apps:
    - name: my-app
      sources:
      - repoURL: https://github.com/example/repo.git
        desiredRevision: ${{ outputs.commit.commit }}
```

### Updating a Target Revision

:::caution

Without making any modifications to a Git repository, this example simply
updates a "live" Argo CD `Application` resource to point its `targetRevision`
field at a specific version of a Helm chart, which Argo CD will pull directly
from the chart repository.

While this can be a useful technique, it should be used with caution. This is
not "real GitOps" since the state of the `Application` resource is not backed
up in a Git repository. If the `Application` resource were deleted, there would
be no remaining record of its desired state.

:::

```yaml
vars:
- name: gitRepo
  value: https://github.com/example/repo.git
steps:
- uses: argocd-update
  config:
    apps:
    - name: my-app
      sources:
      - repoURL: ${{ chartRepo }}
        chart: my-chart
        desiredRevision: ${{ chartFrom(chartRepo, "my-chart").Version }}
        updateTargetRevision: true
```

### Updating an Image with Kustomize

:::caution

Without making any modifications to a Git repository, this example simply
updates Kustomize-specific properties of a "live" Argo CD `Application`
resource.

While this can be a useful technique, it should be used with caution. This is
not "real GitOps" since the state of the `Application` resource is not backed up
in a Git repository. If the `Application` resource were deleted, there would be
no remaining record of its desired state.

:::

```yaml
vars:
- name: gitRepo
  value: https://github.com/example/repo.git
steps:
- uses: argocd-update
  config:
    apps:
    - name: my-app
      sources:
      - repoURL: https://github.com/example/repo.git
        kustomize:
          images:
          - repoURL: ${{ vars.imageRepo }}
            tag: ${{ imageFrom(vars.imageRepo).Tag }}
```

### Updating an Image with Helm

:::caution

Without making any modifications to a Git repository, this example simply
updates Helm-specific properties of a "live" Argo CD `Application` resource.

While this can be a useful technique, it should be used with caution. This is
not "real GitOps" since the state of the `Application` resource is not backed
up in a Git repository. If the `Application` resource were deleted, there would
be no remaining record of its desired state.

:::

```yaml
steps:
- uses: argocd-update
  config:
    apps:
    - name: my-app
      sources:
      - repoURL: https://github.com/example/repo.git
        helm:
          images:
          - key: image.tag
            value: ${{ imageFrom("my/image").Tag }}
```

### Selecting Applications with matchLabels

This example shows how to use label selectors to match `Application`s based on
simple key-value label matching. All specified labels must match for an
application to be selected.

```yaml
steps:
- uses: argocd-update
  config:
    apps:
    - selector:
        matchLabels:
          environment: production
          team: platform
      sources:
      - repoURL: https://github.com/example/repo.git
        desiredRevision: ${{ outputs.commit.commit }}
```

This configuration will select all Argo CD `Application` resources in the
default namespace that have both the `environment: production` and
`team: platform` labels.

### Selecting Applications with matchExpressions

This example demonstrates using `matchExpressions` with the `In` operator to
select `Application`s that match one of several possible values for a label.

```yaml
steps:
- uses: argocd-update
  config:
    apps:
    - selector:
        matchExpressions:
        - key: environment
          operator: In
          values:
          - staging
          - production
      sources:
      - repoURL: https://github.com/example/repo.git
        desiredRevision: ${{ outputs.commit.commit }}
```

This configuration will select all Argo CD `Application` resources that have
an `environment` label with a value of either `staging` or `production`.

### Combining matchLabels and matchExpressions

This example shows how to combine both `matchLabels` and `matchExpressions` for
more precise selection criteria. All criteria must be satisfied for an
application to be selected.

```yaml
steps:
- uses: argocd-update
  config:
    apps:
    - selector:
        matchLabels:
          team: platform
        matchExpressions:
        - key: environment
          operator: In
          values:
          - staging
          - production
        - key: managed-by
          operator: NotIn
          values:
          - legacy-system
      sources:
      - repoURL: https://github.com/example/repo.git
        desiredRevision: ${{ outputs.commit.commit }}
```

This configuration will select all Argo CD `Application` resources that:

- Have the label `team: platform`
- Have an `environment` label with value `staging` or `production`
- Do NOT have a `managed-by` label with value `legacy-system`

### Hard Refresh with Label Selectors

This example shows how to use a label selector to perform a hard refresh of
multiple `Application`s without updating any sources. This is useful for forcing
Argo CD to re-sync `Application`s, for example after external changes have been
made to the cluster or source repositories.

```yaml
steps:
- uses: argocd-update
  config:
    apps:
    - selector:
        matchLabels:
          auto-refresh: enabled
```

This configuration will trigger a hard refresh on all Argo CD `Application`
resources that have the label `auto-refresh: enabled`. Since no `sources` are
specified, Kargo will simply ensure the `Application`s are synced to their
current target revisions without making any changes to the application
specifications.

### Updating Multiple Environments Simultaneously

This example demonstrates updating multiple `Application`s across different
environments in a single step. This is useful for rolling out changes to
multiple stages simultaneously, such as in a blue/green deployment scenario.

```yaml
vars:
- name: gitRepo
  value: https://github.com/example/repo.git
steps:
# Clone, render manifests, commit, push, etc...
- uses: git-commit
  as: commit
  config:
    path: ./out
    message: Update to new version
- uses: git-push
  config:
    path: ./out
- uses: argocd-update
  config:
    apps:
    - selector:
        matchLabels:
          app: my-microservice
          deployment-group: blue
      sources:
      - repoURL: ${{ vars.gitRepo }}
        desiredRevision: ${{ outputs.commit.commit }}
```

This configuration will update all Argo CD `Application` resources that have
both the `app: my-microservice` and `deployment-group: blue` labels, pointing
them all to the same new revision. Kargo will validate that all matched
`Application`s have compatible source configurations (i.e., the source exists
in each `Application`) before applying any updates.
