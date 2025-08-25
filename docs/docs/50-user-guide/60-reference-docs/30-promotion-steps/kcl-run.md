---
sidebar_label: kcl-run
description: Executes KCL (Kushion Configuration Language) configuration files to generate YAML output.
---

# `kcl-run`

`kcl-run` executes KCL (Kushion Configuration Language) configuration files to
generate YAML output. This step uses the `kcl-go` SDK to run KCL programs
without requiring a system dependency. This step is useful for generating
configuration files, manifests, or other YAML/JSON output from KCL programs.

## Configuration

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `inputPath` | `[]string` | Y | List of paths to directories containing KCL files (*.k) to execute, or paths to individual KCL files. These paths are relative to the temporary workspace that Kargo provisions for use by the promotion process. When a directory is specified, all .k files in the directory will be processed. |
| `outputPath` | `string` | N | Path where the KCL output should be written. If not specified, output will be returned in the step result. This path is relative to the temporary workspace that Kargo provisions for use by the promotion process. |
| `settings` | `object` | N | Key-value pairs to pass to the KCL execution as options. |
| `args` | `[]string` | N | Additional arguments to pass to the KCL execution as key-value pairs. |
| `valueFiles` | `[]string` | N | Paths to YAML/JSON files containing values to be used during KCL execution. These files will be made available to KCL as external data sources. |
| `oci` | `object` | N | OCI configuration for fetching schema definitions from upstream registries. |
| `oci.registry` | `string` | N | Registry URL for OCI schema definitions. Defaults to `ghcr.io`. |
| `oci.repo` | `string` | N | Repository name for OCI schema definitions. Defaults to `kcl-lang`. |

## Examples

### Basic Usage with Directory

```yaml
apiVersion: kargo.akuity.io/v1alpha1
kind: Promotion
metadata:
  name: my-promotion
spec:
  steps:
  - uses: kcl-run
    config:
      inputPath: 
        - config/kcl-files
      outputPath: manifests/app.yaml
```

### Basic Usage with Single File

```yaml
apiVersion: kargo.akuity.io/v1alpha1
kind: Promotion
metadata:
  name: my-promotion
spec:
  steps:
  - uses: kcl-run
    config:
      inputPath: 
        - config/app.k
      outputPath: manifests/app.yaml
```

### With Multiple Input Paths

```yaml
apiVersion: kargo.akuity.io/v1alpha1
kind: Promotion
metadata:
  name: my-promotion
spec:
  steps:
  - uses: kcl-run
    config:
      inputPath: 
        - config/base
        - config/overlays/production
        - config/additional.k
      outputPath: manifests/app.yaml
```

### With Settings

```yaml
apiVersion: kargo.akuity.io/v1alpha1
kind: Promotion
metadata:
  name: my-promotion
spec:
  steps:
  - uses: kcl-run
    config:
      inputPath: 
        - config/kcl-files
      outputPath: manifests/app.yaml
      settings:
        environment: production
        replicas: "3"
        debug: "false"
```

### With Additional Args

```yaml
apiVersion: kargo.akuity.io/v1alpha1
kind: Promotion
metadata:
  name: my-promotion
spec:
  steps:
  - uses: kcl-run
    config:
      inputPath: 
        - config/kcl-files
      outputPath: manifests/app.yaml
      args:
        - "--strict"
        - "true"
        - "--verbose"
        - "true"
```

### With Value Files

```yaml
apiVersion: kargo.akuity.io/v1alpha1
kind: Promotion
metadata:
  name: my-promotion
spec:
  steps:
  - uses: kcl-run
    config:
      inputPath: 
        - config/kcl-files
      outputPath: manifests/app.yaml
      valueFiles:
        - config/values.yaml
        - config/environment/prod.yaml
```

### With OCI Configuration

```yaml
apiVersion: kargo.akuity.io/v1alpha1
kind: Promotion
metadata:
  name: my-promotion
spec:
  steps:
  - uses: kcl-run
    config:
      inputPath: 
        - config/kcl-files
      outputPath: manifests/app.yaml
      oci:
        registry: ghcr.io
        repo: kcl-lang
```

## Example KCL File

```kcl
# app.k
apiVersion = "apps/v1"
kind = "Deployment"
metadata = {
    name = "nginx"
    labels.app = "nginx"
}
spec = {
    replicas = 3
    selector.matchLabels = metadata.labels
    template.metadata.labels = metadata.labels
    template.spec.containers = [
        {
            name = metadata.name
            image = "${metadata.name}:1.14.2"
            ports = [{ containerPort = 80 }]
        }
    ]
}
```

## Output

The `kcl-run` step will:
1. Find and execute all .k files in the specified directory (or the single file if a file path is provided)
2. Execute the KCL files using the kcl-go SDK
3. Generate YAML output from the KCL configuration
4. Either write the output to the specified file path or return it in the step result

If `outputPath` is specified, the step result will contain:
```json
{
  "outputPath": "path/to/output.yaml"
}
```

If `outputPath` is not specified, the step result will contain:
```json
{
  "output": "generated YAML content"
}
```

## Common Use Cases

- Generating Kubernetes manifests from KCL configuration
- Creating configuration files for different environments
- Transforming data structures using KCL's powerful features
- Generating complex YAML configurations with validation and constraints
