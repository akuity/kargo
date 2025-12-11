---
sidebar_label: http-download
description: Downloads files from HTTP/S URLs to enable integration with external file repositories and CDNs.
---

# `http-download`

`http-download` is a step that downloads files from HTTP/S URLs to enable
integration with external file repositories, CDNs, and other web-based file
sources.

:::note

Downloads are limited to 100MB to prevent resource exhaustion.

:::

## Configuration

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `url` | `string` | Y | The URL from which to download the file. |
| `outPath` | `string` | Y | The path where the downloaded file will be saved, relative to the step's working directory. |
| `allowOverwrite` | `boolean` | N | Whether to allow overwriting an existing file at the specified path. If `false` and the file exists, the download will fail. Defaults to `false`. |
| `headers` | `[]object` | N | A list of headers to include in the request. |
| `headers[].name` | `string` | Y | The name of the header. |
| `headers[].value` | `string` | Y | The value of the header. |
| `queryParams` | `[]object` | N | A list of query parameters to include in the request. |
| `queryParams[].name` | `string` | Y | The name of the query parameter. |
| `queryParams[].value` | `string` | Y | The value of the query parameter. The provided value will automatically be URL-encoded if necessary. |
| `insecureSkipTLSVerify` | `boolean` | N | Indicates whether to bypass TLS certificate verification when making the request. Setting this to `true` is highly discouraged. |
| `timeout` | `string` | N | A string representation of the maximum time interval to wait for the download to complete. See Go's [`time` package docs](https://pkg.go.dev/time#ParseDuration) for a description of the accepted format. Defaults to 5 minutes. |

## Outputs

The `http-download` step does not produce any outputs. Success is indicated by
the step completing without error and the file being present at the specified
path.

## Examples

### Basic Usage

This example configuration downloads a configuration file from a web server:

```yaml
steps:
# ...
- uses: http-download
  as: fetch-config
  config:
    url: https://example.com/config/app.yaml
    outPath: config/app.yaml
```

The step would download the file and save it to `config/app.yaml` in the working
directory. Parent directories are created automatically if they don't exist.

### Download with Authentication

This example downloads a file from a protected endpoint using authentication
headers:

```yaml
steps:
# ...
- uses: http-download
  as: fetch-artifact
  config:
    url: https://artifacts.example.com/releases/v1.2.3/app.tar.gz
    outPath: artifacts/app.tar.gz
    headers:
    - name: Authorization
      value: Bearer ${{ secret('artifacts').token }}
    allowOverwrite: true
    timeout: 10m
```

### Download with Query Parameters

This example downloads a file using query parameters and demonstrates handling
of existing files:

```yaml
steps:
# ...
- uses: http-download
  as: fetch-release
  config:
    url: https://artifacts.example.com/releases/source.zip
    outPath: releases/source.zip
    queryParams:
    - name: version
      value: ${{ vars.version }}
    headers:
    - name: Authorization
      value: token ${{ secret('artifacts').token }}
    allowOverwrite: false
    timeout: 5m
```

If `releases/source.zip` already exists, this step would fail terminally since
`allowOverwrite` is `false`.

### Download and Unpack Helm Chart

This example downloads a Helm chart archive and unpacks it into a directory:

```yaml
vars:
- name: chartRepo
  value: https://charts.example.com
- name: chartName
  value: my-helm-chart
steps:
# ...
- uses: http-download
  as: fetch-chart
  config:
    url: https://charts.example.com/${{ vars.chartName }}-${{ chartFrom(vars.chartRepo, vars.chartName).Version }}.tgz
    outPath: charts/${{ vars.chartName }}.tgz
    headers:
    - name: Accept
      value: application/gzip
    allowOverwrite: true
- as: extract-chart
  uses: untar
  config:
    inPath: charts/${{ vars.chartName }}.tgz
    outPath: charts/${{ vars.chartName }}
```
