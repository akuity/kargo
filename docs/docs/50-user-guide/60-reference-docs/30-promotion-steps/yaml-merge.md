---
sidebar_label: yaml-merge
description: Merge multiple YAML files into a single file.
---

# `yaml-merge`

`yaml-merge` merges multiple YAML files into a single file.
YAML files are merged in order.The first file in the list
is the source, and all subsequent files are applied over it.

When `ignoreMissingFiles` is false (default), the Task will fail
if any file from `inPaths` does not exist.


:::note
Merging is done with usual constrains:
- new objects are added
- object with same name are modified
- lists are replaced by latest version (no merge)
- null values delete the object
:::

## Configuration

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `inPaths` | `[]string` | Y | Paths to a YAML files. This path is relative to the temporary workspace that Kargo provisions for use by the promotion process. |
| `outPath` | `string`   | Y | The path to the output file. This path is relative to the temporary workspace that Kargo provisions for use by the promotion process. |
| `ignoreMissingFiles` | `bool` | N | When set to true, the directive will skip input files that does not exist. Defaults to `false`. |

## Output

| Name | Type | Description |
|------|------|-------------|
| `commitMessage` | `string` | A description of the change(s) applied by this step. Typically, a subsequent [`git-commit` step](git-commit.md) will reference this output and aggregate this commit message fragment with other like it to build a comprehensive commit message that describes all changes. |

## Examples

### Common Usage

In this example, three Helm value files, one global, one more specific
for the QA environment and a last one specific to the cluster, are merged
into a new single file, that is then commited to the deployment branch.

This pattern is commonly used when you need to merge multiple value files
into a final `values.yaml` file, to be used by Helm.

```yaml
vars:
- name: gitRepo
  value: https://github.com/example/repo.git
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
- uses: yaml-merge
  config:
    inPaths:
      - ./src/charts/my-chart/values.yaml
      - ./src/charts/qa/values.yaml
      - ./src/charts/qa/cluster-a/values.yaml
    outPath: ./out/charts/my-chart/values.yaml
# Render manifests to ./out, commit, push, etc...
```


Given the sample input YAMLs:

**charts/my-chart/values.yaml**
```yaml
service:
  enabled: false
```

**charts/qa/values.yaml**
```yaml
service:
  enabled: true
```

**charts/qa/cluster-a/values.yaml**
```yaml
service:
  type: LoadBalancer
```

The step would produce the following output:

**charts/my-chart/values.yaml**
```yaml
service:
  enabled: true
  type: LoadBalancer
```
