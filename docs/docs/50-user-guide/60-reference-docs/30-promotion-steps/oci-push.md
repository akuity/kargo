---
sidebar_label: oci-push
description: Pushes OCI artifacts to a registry, either by copying/retagging between registries or by uploading a local archive.
---

# `oci-push`

`oci-push` pushes OCI artifacts to a registry. It operates in one of two modes:

- **Copy/retag** (`srcRef`): copies or retags an existing artifact between
  registries or within the same registry. This supports container images and
  Helm charts stored in OCI registries — for example, retagging an image with a
  release version or copying it to a production registry. Multi-arch image
  indexes are copied in full.
- **Push a local archive** (`srcPath`): uploads a local file from the workspace
  (such as a tarball of rendered manifests) as a single-layer OCI artifact, with
  configurable media types. This is useful for publishing rendered manifests as
  an artifact for consumption by Argo CD, Flux, or other OCI-aware tooling.

Registry authentication is supported for both source and destination.

Exactly one of `srcRef` or `srcPath` must be specified.

## Configuration

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `srcRef` | `string` | Y* | Reference to the source OCI artifact. Supports both tag format `registry/repository:tag` and digest format `registry/repository@sha256:digest`. For Helm OCI artifacts, the `oci://` prefix is supported (e.g., `oci://registry/repository:tag`) and will use Helm-specific credential lookup. Mutually exclusive with `srcPath`. |
| `srcPath` | `string` | Y* | Path, relative to the workspace, of a local file to push as a single-layer OCI artifact (e.g., a tarball produced by an earlier step). Mutually exclusive with `srcRef`. |
| `destRef` | `string` | Y | Destination reference including tag (e.g., `registry/repository:tag`). For Helm OCI artifacts, the `oci://` prefix is supported. For retag-in-place, use the same repository as `srcRef` with the new tag. |
| `mediaType` | `string` | N | Media type of the artifact layer when pushing a local file via `srcPath`. Defaults to `application/vnd.oci.image.layer.v1.tar+gzip`. Ignored when using `srcRef`. |
| `artifactType` | `string` | N | Declares the type of artifact being pushed via `srcPath`. It is applied as the manifest's config media type (following the convention used by Helm, Flux, and ORAS). Ignored when using `srcRef`. |
| `annotations` | `object` | N | Annotations to set on the destination artifact. Keys may be prefixed with `index:` or `manifest:` to scope them to the index or image manifest respectively. Unprefixed keys default to the image manifest. For single images (including local-archive pushes), `index:`-prefixed keys are ignored. Values support expressions. When copying with `srcRef`, existing annotations on the source artifact are preserved; specified annotations are added or overwritten. |
| `insecureSkipTLSVerify` | `boolean` | N | Whether to skip TLS verification for both source and destination registries. Defaults to `false`. |

`*` Exactly one of `srcRef` or `srcPath` is required.

:::note

`srcPath` is content-agnostic: it pushes any file as a single layer, so it is
not limited to tarballs. A compressed tarball is the common case, but single
non-tar files (e.g., a WASM module or an SBOM) work as well — set `mediaType`
and `artifactType` to describe the content.

Producing canonical Helm charts via `srcPath` is not supported, because a Helm
chart's manifest config blob must carry the chart's `Chart.yaml` metadata. Use
the copy/retag mode (`srcRef` with the `oci://` prefix) to promote existing Helm
charts between registries.

:::

## Outputs

| Name | Type | Description |
|------|------|-------------|
| `image` | `string` | Full destination reference with tag (e.g., `prod.example.com/myapp:v1.2.3`). |
| `digest` | `string` | Digest of the pushed artifact (e.g., `sha256:abc123...`). |
| `tag` | `string` | Tag that was applied, parsed from `destRef`. |

## Limits

The total compressed size of the artifact (config blob and all layers) must not
exceed 1 GiB (or as configured by your administrator). For multi-arch image indexes, this includes the sum across all
child images. Exceeding this limit causes a terminal (non-retryable) error.

This limit is not enforced when `srcRef` and `destRef` refer to the same
repository (i.e. retagging within the same registry and path), since no blob
transfer occurs in that case. It is always enforced for local-archive pushes
(`srcPath`), based on the file's size.

## Examples

### Pushing a Local Archive

In this example, a directory of rendered manifests is packaged into a tarball
and pushed to an OCI registry as an artifact — the pattern used to publish
manifests for consumption by Argo CD, Flux, or other OCI-aware tooling. The
`mediaType` and `artifactType` follow Flux's OCI conventions, and provenance is
recorded via standard OCI annotations.

:::note

The `tar` step used below to produce the archive is planned but not yet
available. Until then, produce the archive with another step (for example, by
invoking `tar` from a script) and point `srcPath` at the result.

:::

```yaml
steps:
- uses: tar
  config:
    path: ./manifests
    outPath: ./manifests.tar.gz
- uses: oci-push
  config:
    srcPath: ./manifests.tar.gz
    destRef: oci://ghcr.io/example/config/app:${{ ctx.promotion }}
    mediaType: application/vnd.cncf.flux.content.v1.tar+gzip
    artifactType: application/vnd.cncf.flux.config.v1+json
    annotations:
      org.opencontainers.image.source: ${{ commitFrom("https://github.com/example/app.git").repoURL }}
      org.opencontainers.image.revision: ${{ commitFrom("https://github.com/example/app.git").id }}
```

### Retagging an Image with a Release Version

In this example, a dedicated "release" Stage sits downstream from a testing
Stage. When verified Freight is promoted into this Stage, its single step retags
the image with a semver release version in the same repository. Because the
source and destination are in the same repository, this is a lightweight
metadata operation with no blob transfer.

```yaml
steps:
- uses: oci-push
  config:
    srcRef: registry.example.com/myapp@${{ imageFrom("registry.example.com/myapp").digest }}
    destRef: registry.example.com/myapp:v1.2.3
```

### Copying to a Production Registry

In this example, a verified image is copied from a sandbox registry to a
production registry, preserving its original tag. Credentials for each registry
are resolved independently.

```yaml
steps:
- uses: oci-push
  config:
    srcRef: sandbox.example.com/myapp@${{ imageFrom("sandbox.example.com/myapp").digest }}
    destRef: prod.example.com/myapp:${{ imageFrom("sandbox.example.com/myapp").tag }}
```

### Copying to a Per-Stage Repository

In this example, images are copied to stage-specific repositories (e.g., for
garbage collection policies that limit images per repository). The step output
is then used by `kustomize-set-image` to update the deployment manifest.

```yaml
apiVersion: kargo.akuity.io/v1alpha1
kind: Stage
# ...
spec:
  # ...
  promotionTemplate:
    spec:
      steps:
      - uses: oci-push
        as: push-to-stage
        config:
          srcRef: registry.example.com/widget-service@${{ imageFrom("registry.example.com/widget-service").digest }}
          destRef: registry.example.com/widget-service/${{ ctx.stage }}:${{ imageFrom("registry.example.com/widget-service").tag }}
      - uses: kustomize-set-image
        config:
          path: ./out
          images:
          - image: registry.example.com/widget-service
            newName: registry.example.com/widget-service/${{ ctx.stage }}
            digest: ${{ outputs["push-to-stage"].digest }} # Or task.outputs in a (Cluster)PromotionTask
```

### Promoting an OCI Helm Chart

In this example, a Helm chart stored in an OCI registry is copied to a
production registry. The `oci://` prefix ensures Helm-specific
[credentials](../../50-security/30-managing-secrets.md) are used for
authentication.

```yaml
steps:
- uses: oci-push
  config:
    srcRef: oci://registry.example.com/charts/my-app:${{ chartFrom("oci://registry.example.com/charts/my-app").version }}
    destRef: oci://prod-registry.example.com/charts/my-app:${{ chartFrom("oci://registry.example.com/charts/my-app").version }}
```

### Adding Annotations

In this example, OCI annotations are stamped onto the destination manifest
during the push. This can be used to record provenance metadata such as the
source repository or Kargo promotion name.

```yaml
steps:
- uses: oci-push
  config:
    srcRef: registry.example.com/myapp@${{ imageFrom("registry.example.com/myapp").digest }}
    destRef: registry.example.com/myapp:v1.2.3
    annotations:
      org.opencontainers.image.source: "https://github.com/example/myapp"
      io.kargo.promotion: ${{ ctx.promotion }}
```

### Scoped Annotations for Multi-Arch Images

When pushing image indexes (multi-arch), annotation keys can be prefixed with
`index:` or `manifest:` to control where they are applied. Unprefixed keys
default to the image manifest.

```yaml
steps:
- uses: oci-push
  config:
    srcRef: registry.example.com/myapp@${{ imageFrom("registry.example.com/myapp").digest }}
    destRef: registry.example.com/myapp:v1.2.3
    annotations:
      org.opencontainers.image.source: "https://github.com/example/myapp"
      index:org.opencontainers.image.revision: ${{ commitFrom("https://github.com/example/myapp").id }}
      manifest:org.opencontainers.image.description: "my app image"
```

### Copying with TLS Verification Disabled

In this example, an artifact is copied between registries with self-signed
certificates by disabling TLS verification. This should only be used in
development or testing environments where the registries are trusted.

```yaml
steps:
- uses: oci-push
  config:
    srcRef: internal-registry.local/myapp:latest
    destRef: staging-registry.local/myapp:latest
    insecureSkipTLSVerify: true
```
