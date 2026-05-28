---
sidebar_label: Custom steps
description: Execute command in user-provided image
---

<span class="tag professional"></span>
<span class="tag beta"></span>

# Custom steps

:::info

This promotion step is only available in Kargo on the
[Akuity Platform](https://akuity.io/akuity-platform), versions v1.10 and above.

Additionally, it requires enabling of the Promotion Controller and use of self-hosted agent
to allow for Pod-based promotions.

:::

:::warning

This feature is alpha version, configurations and specifications may change in the future.
Performance and stability have not been tested yet, using the feature for memory-heavy processes may cause issues.

:::

Custom steps allow operators to configure their own promotion step logic using OCI images and scripting.
This can be used for tasks currently not provided by built-in steps to extend Kargo capabilites
with bespoke or proprietary functionality.

## Registering a custom step

Custom steps have to be registered in a Kargo cluster in order to be used in promotion templates.
To register a new step, a Kargo cluster admin needs to create a cluster-scoped `CustomPromotionStep` resource:

```yaml
apiVersion: ee.kargo.akuity.io/v1alpha1
kind: CustomPromotionStep
metadata:
  name: hello-world
spec:
  ## REQUIRED: image to execute the command in
  image: ubuntu
  ## REQUIRED: command to run
  command: ["sh", "-c", "sleep 5; echo 'hello=${HELLO}-${{ config.world }}' > $KARGO_OUTPUT"]
  ## OPTIONAL: additional environment to provide to the command
  env:
    - name: HELLO
      value: ${{ config.hello }}
## OPTIONAL: output capture configuration
  output:
    source:
## One of Pipe | Stdout | Stderr | File, default is Pipe
      type: Pipe
## required when type is File, should be omitted if it's not
      path: ""
## One of JSON | YAML | KeyValue | Text
## inferred from file extension when omitted, default is KeyValue
      format: KeyValue
## Apply expressions to output to reformat with different keys
## If omitted returns the parsed output as is
    transform:
      message: output.hello
## OPTIONAL: error handling metadata
  defaultTimeout: 5m
  defaultErrorThreshold: 3
## OPTIONAL: capabilities to provision into step container
## Options: access-control-plane, access-argocd
  capabilities: []
## OPTIONAL: container resources configuration
  resources:
    requests:
      memory: "64Mi"
      cpu: "250m"
    limits:
      memory: "128Mi"
      cpu: "500m"
## OPTIONAL: secrets to pull the image from private repos
  imagePullSecrets: []
```

:::warning

The `command` field does not specify the `command` of the step container in promotion pod definition.
Kargo is running an executor binary which coordinates step execution (start, retry, abort) and will run the `command`.

:::

:::warning

If retry policy is set for the step, the `command` could be executed multiple times on failure.
It's recommended to design commands to have some idempotency.

:::


## Using a custom step

After a custom step is registered in the cluster, it can be used in promotion template:

```yaml
vars:
- name: exampleVar
  value: example
steps:
- as: my-custom-step
  uses: hello-world
  config:
    world: ${{ vars.exampleVar }}
    hello: bonjour
    something:
      else:
        - entirely
```

### Passing input to steps

Command execution will run in the same workdir as other steps.

To access step config or step execution context, `command` and values in `env` can use templates like:

```yaml
command:
  - "echo"
  - "Step ${{ ctx.meta.step.alias }} in promotion $PROM_VAR with ${{ config.my_config }} and ${{ config.nested.values[0] }}"
env:
  - name: PROM_VAR
    value: ${{ ctx.promotion }}
```
And config:
```yaml
config:
  my_config: configvalue
  nested:
    values:
      - one
```

:::info

There is no config validation at the moment and configuration structure and keys can be arbitrary.
The step will error in runtime if any expressions require a missing config value, e.g. `${{ config.something.else }}`

:::


Templates are using [expression language](../40-expressions.md), but only with `config` and `ctx` variables. The `ctx` variable is using [the ctx format](../40-expressions.md#context-ctx-object-structure).

In order to pass values from secrets, configmaps or other promotion context to the custom step, they should be used in the promotion template and passed as `config` variables.

#### Passing secrets to custom steps

Example using credential from a secret, assuming secret `db_credential` has keys `username` and `password`:

```yaml
apiVersion: ee.kargo.akuity.io/v1alpha1
kind: CustomPromotionStep
metadata:
  name: access-db
spec:
  image: my_image
  command: ["db_script.sh", "--username=${{ config.cred.username }}"]
  env:
    - name: DB_PASSWORD
      value: ${{ config.cred.password }}
``` 

```yaml
- as: custom-db-step
  uses: access-db
  config:
    cred: ${{ secret("db_credential") }}
```

### Steps output

Output from a step execution is parsed according to the `output` configuration:
```yaml
output:
  source:
    type: Pipe | Stdout | Stderr | File
    path: <string>
    format: JSON | YAML | KeyValue | Text
  transform:
    <key>: <expression_string>
```

If `output` is not specified, default configuration is:
```yaml
output:
  source:
    type: Pipe
    path: ""
    format: KeyValue
  transform: {}
```

#### Output sources

| Type | Behavior |
|------|----------|
| `Pipe` | Reads from the `KARGO_OUTPUT` temp file injected by the executor. Default when `output` is omitted. |
| `Stdout` | Reads from the command's standard output. |
| `Stderr` | Reads from the command's standard error. |
| `File` | Reads from a file at the specified `path`. Format inferred from file extension when omitted. |

#### Ouptut formats

| Format | Behavior |
|--------|----------|
| `KeyValue` | `key=value` lines (GitHub Actions–compatible). Quotes are part of the value, not stripped. Lines without `=` are ignored. Multiline values supported via heredoc syntax (`key<<EOF` / lines / `EOF`). |
| `JSON` | Parsed as JSON. Objects used directly; arrays/scalars wrapped as `{"output": <value>}`. |
| `YAML` | Same semantics as JSON. Format inferred from `.yaml`/`.yml` file extensions. |
| `Text` | Stored as `{"output": "<raw string>"}`. |

#### Transform expressions

An optional `transform` map accepts per-key [expr-lang](https://expr-lang.org) expressions to reshape the parsed output before it is stored. The variable `output` holds the parsed value (map for JSON/YAML/KeyValue, string for Text). Each expression may return any value:

```yaml
transform:
  hasCritical: output.summary.critical > 0
  total: output.summary.total
  failed: output contains "FAIL"
  message: output.message
```

If `transform` map is used, it will completely replace the output produced by the `source`.
To pass some values through they should map to their respective output key (like `message` in the example above)

#### Output size limits

- All formats: hard 256 KiB limit on the final output map (checked after transform). Returns an error if exceeded.
- The size check happens after `transform` evaluation, so a transform that extracts a small subset of a large payload will not hit the limit.
- Step result messages: stdout/stderr truncated to the last 16 KiB each to keep Kubernetes status conditions readable.

## Runtime limitations

- Supported container architectures:
    - linux-arm64
    - linux-amd64
- Aborting the step will kill the process with `SIGKILL`. There is no graceful shutdown at the moment.
- Currently step containers run with user `65532`, images requiring specific user are not supported yet.
- Avoid using too many steps in the same promotion as they utilize the same pod and may exhaust pod/node resources.

## ADVANCED: step capabilities

Kargo steps can have the following capabilities configured:

- `access-control-plane` - allow access to Kargo controlplane via k8s API. `kubeconfig` or `token` (for local in-cluster access) will be provisioned to `coordination/kubernetes/kargo` directory in the container.
- `access-argocd` - allow access to ArgoCD controlplane via k8s API. `kubeconfig` or `token` (for local in-cluster access) will be provisioned to `coordination/kubernetes/argocd` directory in the container.
