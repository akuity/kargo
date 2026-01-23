---
sidebar_label: oci-download
description: Downloads OCI artifacts from a registry and extracts layer content to a specified file.
---

# `oci-download`

`oci-download` downloads OCI artifacts from a registry and extracts layer
content to a specified file. This step is useful for downloading artifacts like
Helm charts, configuration files, or other resources packaged as OCI artifacts.
The step supports authentication and can target specific layers by media type.

:::note

Downloads are limited to 100MB to prevent resource exhaustion.

:::

## Configuration

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `imageRef` | `string` | Y | Reference to the OCI artifact to download. Supports both tag format `registry/repository:tag` and digest format `registry/repository@sha256:digest`. For Helm OCI artifacts, the `oci://` prefix is supported (e.g., `oci://registry/repository:tag`) and will use Helm-specific credential lookup. |
| `outPath` | `string` | Y | Path to the destination file where the extracted artifact will be saved. This path is relative to the temporary workspace that Kargo provisions for use by the promotion process. |
| `allowOverwrite` | `boolean` | N | Whether to allow overwriting an existing file at the specified path. If `false` and the file exists, the download will fail. Defaults to `false`. |
| `mediaType` | `string` | N | Media type of the layer to download. Selects the first layer matching this type. If not specified, selects the first layer available. |
| `insecureSkipTLSVerify` | `boolean` | N | Whether to skip TLS verification when downloading the artifact. Defaults to `false`. |

## Examples

### Downloading a Helm Chart

In this example, an OCI-packaged Helm chart is downloaded from a registry and
saved to a local file. This is useful when you need to download charts that are
distributed as OCI artifacts.

```yaml
steps:
- uses: oci-download
  config:
    imageRef: registry.example.com/charts/my-app:1.0.0
    outPath: ./charts/my-app-1.0.0.tgz
```

### Downloading a Helm Chart with OCI Protocol

This example shows downloading a Helm chart using the `oci://` prefix, which
ensures that [Helm-specific credentials](../../50-security/30-managing-secrets.md)
are used for authentication.

```yaml
steps:
- uses: oci-download
  config:
    imageRef: oci://registry.example.com/charts/my-app:1.0.0
    outPath: ./charts/my-app-1.0.0.tgz
```

### Downloading Configuration Files

In this example, configuration files packaged as an OCI artifact are downloaded
and extracted. The step downloads the first available layer since no specific
media type is specified.

```yaml
steps:
- uses: oci-download
  config:
    imageRef: registry.example.com/configs/app-config@sha256:abc123def456789
    outPath: ./config/app-config.yaml
```

### Downloading with Digest Reference

In this example, an artifact is downloaded using a digest reference for
immutable content addressing. This ensures you get exactly the same content
every time, regardless of tag mutations.

```yaml
steps:
- uses: oci-download
  config:
    imageRef: ghcr.io/example/artifacts@sha256:1234567890abcdef
    outPath: ./artifacts/data.tar.gz
```

### Downloading with a Specific Media Type

In this example, an artifact is downloaded by specifying a media type. This is
useful when the OCI artifact contains multiple layers, and you want to target a
specific one, such as a configuration file attached to a container image.

```yaml
steps:
- uses: oci-download
  config:
    imageRef: registry.example.com/artifacts/my-app:v1.2.3
    outPath: ./artifacts/config.json
    mediaType: application/vnd.example.config.v1+json
```

### Downloading with TLS Verification Disabled

In this example, an artifact is downloaded from a registry with self-signed
certificates by disabling TLS verification. This should only be used in
development or testing environments where the registry is trusted.

```yaml
steps:
- uses: oci-download
  config:
    imageRef: internal-registry.local/artifacts/data:latest
    outPath: ./data/artifact.tar.gz
    insecureSkipTLSVerify: true
```

### Downloading and Rendering Helm Charts

This example shows how `oci-download` can be combined with
[`helm-template`](helm-template.md) to download Helm charts from OCI registries
and render them to manifests. After downloading the chart archive, it's rendered
directly with Stage-specific values before being committed to a Git repository.

```yaml
vars:
- name: gitRepo
  value: https://github.com/example/manifests.git
- name: chart
  value: oci://registry.example.com/charts/my-app
steps:
- uses: git-clone
  config:
    repoURL: ${{ vars.gitRepo }}
    checkout:
    - branch: stage/${{ ctx.stage }}
      create: true
      path: ./out
- uses: git-clear
  config:
    path: ./out
- uses: oci-download
  config:
    imageRef: ${{ vars.chart }}:1.0.0
    outPath: ./chart.tgz
- uses: helm-template
  config:
    path: ./chart.tgz
    releaseName: my-app
    namespace: ${{ ctx.stage }}
    outPath: ./out
- uses: git-commit
  config:
    path: ./out
    message: "Update manifests for ${{ ctx.stage }} stage"
- uses: git-push
  config:
    path: ./out
```
