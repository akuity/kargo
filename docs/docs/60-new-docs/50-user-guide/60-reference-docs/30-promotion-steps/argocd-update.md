---
sidebar_label: argocd-update
description: Updates one or more Argo CD `Application` resources in various ways.
---

# `argocd-update`

`argocd-update` updates one or more Argo CD `Application` resources in various
ways. Among other scenarios, this step is useful for the common one of forcing
an Argo CD `Application` to sync after previous steps have updated a remote
branch referenced by the `Application`. This step is commonly the last step in
a promotion process.

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
window. (The step's default timeout is five minutes.)_
:::

## Configuration

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `apps` | `[]object` | Y | Describes Argo CD `Application` resources to update and how to update them. At least one must be specified.  |
| `apps[].name` | `string` | Y | The name of the Argo CD `Application`. __Note:__ A small technical restriction on this field is that any [expressions](../40-expressions.md) used therein are limited to accessing `ctx` and `vars` and may not access `secrets` or any Freight. This is because templates in this field are, at times, evaluated outside the context of an actual `Promotion` for the purposes of building an index. In practice, this restriction does not prove to be especially limiting. |
| `apps[].namespace` | `string` | N | The namespace of the Argo CD `Application` resource to be updated. If left unspecified, the namespace will be the Kargo controller's configured default -- typically `argocd`. __Note:__ This field is subject to the same restrictions as the `name` field. See above. |
| `apps[].sources` | `[]object` | N | Describes Argo CD `ApplicationSource`s to update and how to update them. |
| `apps[].sources[].repoURL` | `string` | Y | The value of the target `ApplicationSource`'s  own `repoURL` field. This must match exactly. |
| `apps[].sources[].chart` | `string` | N | Applicable only when the target `ApplicationSource` references a Helm chart repository, the value of the target `ApplicationSource`'s  own `chart` field. This must match exactly. |
| `apps[].sources[].desiredRevision` | `string` | N | Specifies the desired revision for the source. i.e. The revision to which the source must be observably synced when performing a health check. This field is mutually exclusive with `desiredCommitFromStep`. Prior to v1.1.0, if both were left undefined, the desired revision was determined by Freight (if possible). Beginning with v1.1.0, if both are left undefined, Kargo will not require the source to be observably synced to any particular source to be considered healthy. Note that the source's `targetRevision` will not be updated to this revision unless `updateTargetRevision=true` is also set. |
| `apps[].sources[].desiredCommitFromStep` | `string` | N | Applicable only when `repoURL` references a Git repository, this field references the `commit` output from a previous step and uses it as the desired revision for the source. i.e. The revision to which the source must be observably synced when performing a health check. This field is mutually exclusive with `desiredRevisionFromStep`. Prior to v1.1.0, if both were left undefined, the desired revision was determined by Freight (if possible). Beginning with v1.1.0, if both are left undefined, Kargo will not require the source to be observably synced to any particular source to be considered healthy. Note that the source's `targetRevision` will not be updated to this commit unless `updateTargetRevision=true` is also set.<br/><br/>__Deprecated: Use `desiredRevision` with an expression instead. Will be removed in v1.3.0.__ |
| `apps[].sources[].updateTargetRevision` | `boolean` | Y | Indicates whether the target `ApplicationSource` should be updated such that its `targetRevision` field points directly at the desired revision. A `true` value in this field requires exactly one of `desiredCommitFromStep` or `desiredRevision` to be specified. |
| `apps[].sources[].kustomize` | `object` | N | Describes updates to an Argo CD `ApplicationSource`'s Kustomize-specific properties. |
| `apps[].sources[].kustomize.images` | `[]object` | Y | Describes how to update an Argo CD `ApplicationSource`'s Kustomize-specific properties to reference specific versions of container images. |
| `apps[].sources[].kustomize.images[].repoURL` | `string` | Y | URL of the image being updated. |
| `apps[].sources[].kustomize.images[].tag` | `string` | N | A tag naming a specific revision of the image specified by `repoURL`. Mutually exclusive with `digest` and `useDigest=true`. One of `digest`, `tag`, or `useDigest=true` must be specified. |
| `apps[].sources[].kustomize.images[].digest` | `string` | N | A digest naming a specific revision of the image specified by `repoURL`. Mutually exclusive with `tag` and `useDigest=true`. One of `digest`, `tag`, or `useDigest=true` must be specified. |
| `apps[].sources[].kustomize.images[].useDigest` | `boolean` | N | Whether to use the container image's digest instead of its tag. |
| `apps[].sources[].kustomize.images[].newName` | `string` | N | A substitution for the name/URL of the image being updated. This is useful when different Stages have access to different container image repositories (assuming those different repositories contain equivalent images that are tagged identically). This may be a frequent consideration for users of Amazon's Elastic Container Registry. |
| `apps[].sources[].kustomize.images[].fromOrigin` | `object` | N | See [specifying origins](#specifying-origins). If not specified, may inherit a value from `apps[].sources[].kustomize.fromOrigin`. <br/><br/>__Deprecated: Use `digest` or `tag` with an expression instead. Will be removed in v1.3.0.__ |
| `apps[].sources[].kustomize.fromOrigin` | `object` | N | See [specifying origins](#specifying-origins). If not specified, may inherit a value from `apps[].sources[].fromOrigin`. <br/><br/>__Deprecated: Will be removed in v1.3.0.__ |
| `apps[].sources[].helm` | `object` | N | Describes updates to an Argo CD `ApplicationSource`'s Helm parameters. |
| `apps[].sources[].helm.images` | `[]object` | Y | Describes how to update  an Argo CD `ApplicationSource`'s Helm parameters to reference specific versions of container images. |
| `apps[].sources[].helm.images[].repoURL` | `string` | N | URL of the image being updated. __Deprecated: Use `value` with an expression instead. Will be removed in v1.3.0.__ |
| `apps[].sources[].helm.images[].key` | `string` | Y | The key to update within the target `ApplicationSource`'s `helm.parameters` map. See Helm documentation on the [format and limitations](https://helm.sh/docs/intro/using_helm/#the-format-and-limitations-of---set) of the notation used in this field. |
| `apps[].sources[].helm.images[].value` | `string` | Y | Specifies how the value of `key` is to be updated. When `repoURL` is non-empty, possible values for this field are limited to:<ul><li>`ImageAndTag`: Replaces the value of `key` with a string in form `<image url>:<tag>`</li><li>`Tag`: Replaces the value of `key` with the image's tag</li><li>`ImageAndDigest`: Replaces the value of `key` with a string in form `<image url>@<digest>`</li><li>`Digest`: Replaces the value of `key` with the image's digest</li></ul> When `repoURL` is empty, use an expression in this field to describe the new value.  |
| `apps[].sources[].helm.images[].fromOrigin` | `object` | N | See [specifying origins](#specifying-origins). If not specified, may inherit a value from `apps[].sources[].helm.fromOrigin`. <br/><br/>__Deprecated: Use `value` with an expression instead. Will be removed in v1.3.0.__ |
| `apps[].sources[].helm.fromOrigin` | `object` | N | See [specifying origins].(#specifying-origins). If not specified, may inherit a value from `apps[].sources[]`. <br/><br/>__Deprecated: Will be removed in v1.3.0.__ |
| `apps[].sources[].fromOrigin` | `object` | N | See [specifying origins](#specifying-origins). If not specified, may inherit a value from `apps[].fromOrigin`. <br/><br/>__Deprecated: Will be removed in v1.3.0.__ |
| `apps[].fromOrigin` | `object` | N | See [specifying origins](#specifying-origins). If not specified, may inherit a value from `fromOrigin`.  <br/><br/>__Deprecated: Will be removed in v1.3.0.__ |
| `fromOrigin` | `object` | N | See [specifying origins](#specifying-origins). <br/><br/>__Deprecated: Will be removed in v1.3.0.__  |

## Health Checks

The `argocd-update` step is unique among all other built-in promotion steps in
that, on successful completion, it will register health checks to be performed
upon the target Stage on an ongoing basis. This health check configuration is
_opaque_ to the rest of Kargo and is understood only by health check
functionality built into the step. This permits Kargo to factor the health and
sync state of Argo CD `Application` resources into the overall health of a Stage
without requiring Kargo to understand `Application` health directly.

:::info
Although the `argocd-update` step is the only promotion step to currently
utilize this health check framework, we anticipate that future built-in and
third-party promotion steps will take advantage of it as well.
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
    messageFromSteps:
    - update-image
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
        targetRevision: ${{ chartFrom(chartRepo, "my-chart").Version }}
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
