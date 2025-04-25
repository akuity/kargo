---
sidebar_label: untar
description: Extracts the contents of a tar (or gzipped tar) file to a specified location.
---

# `untar`

`untar` extracts the contents of a tar file (including gzipped tar files) to a specified location. It automatically detects if the file is gzipped and handles it accordingly.

## Configuration

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `inPath` | `string` | Y | Path to the tar file to extract. This path is relative to the temporary workspace that Kargo provisions for use by the promotion process. |
| `outPath` | `string` | Y | Path to the destination directory where contents will be extracted. This path is relative to the temporary workspace that Kargo provisions for use by the promotion process. |
| `stripComponents` | `integer` | N | Number of leading path components to strip from file names in the archive. Similar to the `--strip-components` option in the tar command. |
| `ignore` | `string` | N | A multiline string of glob patterns to ignore when extracting files. It accepts the same syntax as `.gitignore` files. |

## Examples

### Basic Usage

Extract a tarball to a specific directory:

```yaml
steps:
- uses: untar
  config:
    inPath: ./artifacts/bundle.tar.gz
    outPath: ./extracted
```

### Strip Path Components

Extract a tarball while removing leading directory components:

```yaml
steps:
- uses: untar
  config:
    inPath: ./artifacts/bundle.tar.gz
    outPath: ./extracted
    stripComponents: 1
```

### Ignore Patterns

Extract a tarball while ignoring specific files:

```yaml
steps:
- uses: untar
  config:
    inPath: ./artifacts/bundle.tar.gz
    outPath: ./extracted
    ignore: |
      *.log
      temp/
      .DS_Store
```

### Extract helm chart dependency

This example shows how to update dependencies for a given helm chart and then extract the downloaded archive to have the raw manifests as dependencies instead of the tgz.

```yaml
vars:
- name: gitRepo
  value: https://github.com/example/repo.git
- name: chartRepo
  value: https://example.github.io/helm-charts
- name: chartName
  value: myChart
steps:
- uses: git-clone
  config:
    checkout:
      - branch: main
        path: ./target
    repoURL: ${{ vars.gitRepo }}
- uses: helm-update-chart
  config:
    path: ./target/pathToMyUmbrellaChart
    charts:
    - repository: ${{ vars.chartRepo }}
      name: ${{ vars.chartName }}
      version: ${{ chartFrom(vars.chartRepo, vars.chartName).Version }}
- as: deleteFormerChart
  uses: delete
  config:
    path: ./target/pathToMyUmbrellaChart/charts/${{ vars.chartName }}
- as: untar
  uses: untar
  config:
    inPath: ./target/pathToMyUmbrellaChart/charts/${{ vars.chartName }}-${{ chartFrom(vars.chartRepo, "kube-prometheus-stack").Version }}.tgz
    outPath: ./target/pathToMyUmbrellaChart/charts/${{ vars.chartName }}
- as: deleteTgz
  uses: delete
  config:
    path: ./target/pathToMyUmbrellaChart/charts/${{ vars.chartName }}-${{ chartFrom(vars.chartRepo, "kube-prometheus-stack").Version }}.tgz
# Then commit, push ...
```
