---
sidebar_label: yaml-merge
description: Merges multiple YAML files into a single file.
---

# `yaml-merge`

`yaml-merge` merges multiple YAML files into a single file.
YAML files are merged in order. The first file in the list
is the source, and all subsequent files are applied over it.

When `ignoreMissingFiles` is false (default), the step will fail
if any file from `inFiles` does not exist.

:::note

Merging is performed as follows:
- **Scalar values:** If both documents define a scalar (string, number, boolean)
  at the same key, the value from the second document overrides the first.
- **Mapping (object) values:** If both documents define a mapping at the same
  key, the mappings are merged recursively.
- **Sequence (array) values:** If both documents define a sequence at the same
  key, the sequence from the second document replaces the first (no merging).
- **Keys present only in one document:** Keys that exist in only one document
  are included as-is in the result.
- **Null values:** If a key is set to `null` in the second document, it removes
  or overrides the value from the first document.

:::

## Configuration

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `inFiles` | `[]string` | Y | Paths to a YAML files. This path is relative to the temporary workspace that Kargo provisions for use by the promotion process. |
| `outFile` | `string`   | Y | The path to the output file. This path is relative to the temporary workspace that Kargo provisions for use by the promotion process. |
| `ignoreMissingFiles` | `bool` | N | When set to `true`, the directive will skip input files that do not exist. If all input files are missing, an empty output file will be created. Defaults to `false`. |

## Output

| Name | Type | Description |
|------|------|-------------|
| `commitMessage` | `string` | A description of the change(s) applied by this step. Typically, a subsequent [`git-commit` step](git-commit.md) will reference this output and aggregate this commit message fragment with others like it to build a comprehensive commit message that describes all changes. |

## Examples

### Common Usage

In the following example, multiple Helm values files (one "base", a second with
environment-specific overrides, and a third with cluster-specific overrides) are
merged into a new, single file, which is then committed to a `Stage`-specific
branch:

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
    inFiles:
    - ./src/charts/my-chart/values.yaml
    - ./src/charts/qa/values.yaml
    - ./src/charts/qa/cluster-a/values.yaml
    outFile: ./out/charts/my-chart/values.yaml
# Render manifests to ./out, commit, push, etc...
```

Given the following sample input files:

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
