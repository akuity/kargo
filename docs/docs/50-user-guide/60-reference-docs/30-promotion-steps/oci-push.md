---
sidebar_label: oci-push
description: Copies or retags OCI artifacts (container images, Helm charts) between registries.
---

# `oci-push`

`oci-push` copies or retags OCI artifacts between registries or within the same
registry. This step supports container images and Helm charts stored in OCI
registries, making it useful for promoting artifacts through a pipeline — for
example, retagging an image with a release version or copying it to a production
registry. Multi-arch image indexes are copied in full. Registry authentication
is supported for both source and destination.

## Configuration

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `srcRef` | `string` | Y | Reference to the source OCI artifact. Supports both tag format `registry/repository:tag` and digest format `registry/repository@sha256:digest`. For Helm OCI artifacts, the `oci://` prefix is supported (e.g., `oci://registry/repository:tag`) and will use Helm-specific credential lookup. |
| `destRef` | `string` | Y | Destination reference including tag (e.g., `registry/repository:tag`). For Helm OCI artifacts, the `oci://` prefix is supported. For retag-in-place, use the same repository as `srcRef` with the new tag. |
| `annotations` | `object` | N | Annotations to set on the destination artifact. Keys may be prefixed with `index:` or `manifest:` to scope them to the index or image manifest respectively. Unprefixed keys default to the image manifest. For single images, `index:`-prefixed keys are ignored. Values support expressions. Existing annotations on the source artifact are preserved; specified annotations are added or overwritten. |
| `insecureSkipTLSVerify` | `boolean` | N | Whether to skip TLS verification for both source and destination registries. Defaults to `false`. |

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
transfer occurs in that case.

## Examples

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
      digest: ${{ outputs["push-to-stage"].digest }}
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
