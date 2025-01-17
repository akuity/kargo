---
sidebar_label: helm-update-chart
description: Updates the `dependencies` section of a specified Helm chart's `Chart.yaml` file.
---

# `helm-update-chart`

`helm-update-chart` performs specified updates on the `dependencies` section of
a specified Helm chart's `Chart.yaml` file. This step is useful for the common
scenario of updating a chart's dependencies to reflect new versions of charts
referenced by the Freight being promoted. This step is commonly followed by a
[`helm-template` step](helm-template.md).

## Configuration

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `path` | `string` | Y | Path to a Helm chart (i.e. to a directory containing a `Chart.yaml` file). This path is relative to the temporary workspace that Kargo provisions for use by the promotion process. |
| `charts` | `[]string` | Y | The details of dependency (subschart) updates to be applied to the chart's `Chart.yaml` file. |
| `charts[].repository` | `string` | Y | The URL of the Helm chart repository in the `dependencies` entry whose `version` field is to be updated. Must _exactly_ match the `repository` field of that entry. |
| `charts[].name` | `string` | Y | The name of the chart in in the `dependencies` entry whose `version` field is to be updated. Must exactly match the `name` field of that entry. |
| `charts[].fromOrigin` | `object` | N | See [specifying origins](#specifying-origins). <br/><br/>__Deprecated: Use `version` with an expression instead. Will be removed in v1.3.0.__ |
| `charts[].version` | `string` | N | The version to which the dependency should be updated. If left unspecified, the version specified by a piece of Freight referencing this chart will be used. |

## Output

| Name | Type | Description |
|------|------|-------------|
| `commitMessage` | `string` | A description of the change(s) applied by this step. Typically, a subsequent [`git-commit` step](git-commit.md) will reference this output and aggregate this commit message fragment with other like it to build a comprehensive commit message that describes all changes. |

## Examples

### Classic Chart Repository

Given a `Chart.yaml` file such as the following:

```yaml
apiVersion: v2
name: example
type: application
version: 0.1.0
appVersion: 0.1.0
dependencies:
- repository: https://example-chart-repo
  name: some-chart
  version: 1.2.3
```

The `dependencies` can be updated to reflect the version of `some-chart`
referenced by the Freight being promoted like so:

```yaml
vars:
- name: gitRepo
  value: https://github.com/example/repo.git
- name: chartRepo
  value: https://example-chart-repo
steps:
- uses: git-clone
  config:
    repoURL: ${{ vars.gitRepo }}
    checkout:
    - commit: ${{ commitFrom(vars.gitRepo).ID }}
      path: ./src
    - branch: stage/${{ ctx.stage }}
      create: true
      path: ./out
- uses: git-clear
  config:
    path: ./out
- uses: helm-update-chart
  config:
    path: ./src/charts/my-chart
    charts:
    - repository: ${{ chartRepo }}
      name: some-chart
      version: ${{ chartFrom(chartRepo).Version }}
# Render manifests to ./out, commit, push, etc...
```

### OCI Chart Repository

:::caution
Classic (HTTP/HTTPS) Helm chart repositories can contain many differently named
charts. A specific chart, therefore, can be identified by a repository URL and
a chart name.

OCI repositories, on the other hand, are organizational constructs within OCI
_registries._ Each OCI repository is presumed to contain versions of only a
single chart. As such, a specific chart can be identified by a repository URL
alone.

Kargo Warehouses understand this distinction well, so a subscription to an OCI
chart repository will utilize its URL only, _without_ specifying a chart name.
For example:

```yaml
apiVersion: kargo.akuity.io/v1alpha1
kind: Warehouse
metadata:
  name: my-warehouse
  namespace: kargo-demo
spec:
  subscriptions:
  - chart:
      repoURL: oci://example-chart-registry/some-chart
      semverConstraint: ^1.0.0
```

Helm deals with this difference somewhat more awkwardly, however. When a Helm
chart references a chart in an OCI repository, it must reference the _registry_
by URL in the `repository` field and _still_ specify a chart name in the name
field. For example:

```yaml
apiVersion: v2
name: example
type: application
version: 0.1.0
appVersion: 0.1.0
dependencies:
- repository: oci://example-chart-registry
  name: some-chart
  version: 1.2.3
```

__When using `helm-update-chart` to update the dependencies in a `Chart.yaml`
file, you must play by Helm's rules and use the _registry_ URL in the
`repository` field and the _repository_ name (chart name) in the `name` field.__
:::

:::info
As a general rule, when configuring Kargo to update something, observe the
conventions of the thing being updated, even if those conventions differ from
Kargo's own. Kargo is aware of such differences and will adapt accordingly.
:::

Given a `Chart.yaml` file such as the following:

```yaml
apiVersion: v2
name: example
type: application
version: 0.1.0
appVersion: 0.1.0
dependencies:
- repository: oci://example-chart-registry
  name: some-chart
  version: 1.2.3
```

The `dependencies` can be updated to reflect the version of
`oci://example-chart-registry/some-chart` referenced by the Freight being
promoted like so:

```yaml
vars:
- name: gitRepo
  value: https://github.com/example/repo.git
- name: chartReg
  value: oci://example-chart-registry
steps:
- uses: git-clone
  config:
    repoURL: ${{ vars.gitRepo }}
    checkout:
    - commit: ${{ commitFrom(vars.gitRepo).ID }}
      path: ./src
    - branch: stage/${{ ctx.stage }}
      create: true
      path: ./out
- uses: git-clear
  config:
    path: ./out
- uses: helm-update-chart
  config:
    path: ./src/charts/my-chart
    charts:
    - repository: ${{ chartReg }}
      name: some-chart
      version: ${{ chartFrom(chartReg + "/some-chart").Version }}
# Render manifests to ./out, commit, push, etc...
```
