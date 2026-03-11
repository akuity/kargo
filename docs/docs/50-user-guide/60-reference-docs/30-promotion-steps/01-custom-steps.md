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

## Registering custom step

Custom steps have to be registered in Kargo cluster in order to be used in promotion templates.
To register a new step, Kargo cluster admin needs to create a cluster-scoped `CustomPromotionStep` resource:

```
apiVersion: ee.kargo.akuity.io/v1alpha1
kind: CustomPromotionStep
metadata:
  name: hello-world
spec:
  ## REQUIRED: image to execute the command in
  image: ubuntu
  ## REQUIRED: command to run
  command: ["sh", "-c", "sleep 5; echo ::kargo::out::hello::${HELLO}-${{ config.world }}"]
  ## OPTIONAL: additional environment to provide to the command
  env:
    - name: HELLO
      value: ${{ config.hello }}
## OPTIONAL: error handling metadata
#  defaultTimeout: 5m
#  defaultErrorThreshold: 3
## OPTIONAL: capabilities to provision into step container
## Options: access-control-plane, access-argocd
#  capabilities: []
## OPTIONAL: container resources configuration
#  resources:
#    requests:
#      memory: "64Mi"
#      cpu: "250m"
#    limits:
#      memory: "128Mi"
#      cpu: "500m"
## OPTIONAL: secrets to pull the image from private repos
#  imagePullSecrets: []
```

:::warning

The `command` field is does not specify the `command` of the step container in promotion pod definition.
Kargo is running an executor binary which coordinate step execution (start, retry, abort) and will run the `command`.
If retry policy is set for the step, the `command` could be executed multiple times on failure.

:::


## Using custom step

After custom step is registered in the cluster, it can be used in promotion templates:

```
vars:
- name: exampleVar
  value: example
steps:
- as: my-custom-step
  uses: hello-world
  config:
    world: ${{ vars.exampleVar }}
    hello: bonjour
```

There is no config validation at the moment and configuration keys can be arbitrary.

### Passing input to steps

Command execution will run in the same workdir as other steps.

To access step config or step execution context, `command` and values in `env` can use templates like:

```
command:
  - "echo"
  - "Step ${{ ctx.meta.step.alias }} in promotion $PROM_VAR with ${{ config.my_config }}"
env:
  - name: PROM_VAR
    value: ${{ ctx.promotion }}
```
And config:
```
config:
  my_config: configvalue
```

Templates are using [expression language](../40-expressions.md), but only with `config` and `ctx` variables. The `ctx` variable is using [the ctx format](../40-expressions.md#context-ctx-object-structure).

In order to pass values from secrets, configmaps or othe promotion context to the custom step, they should be used in the promotion template and passed as `config` variables.

#### Passing secrets to custom steps

Example using credential from a secret, assuming secret `db_credential` has keys `username` and `password`:

```
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

```
- as: custom-db-step
  uses: access-db
  config:
    cred: ${{ secret("db_credential") }}
```

### Steps output

Output from step execution is parsed from STDOUT of the command execution.

For example, a script `echo ::kargo::out::hello::world` would print `::kargo::out::hello::world`.
This will set the step output to `{"hello":"world"}`.

- Each STDOUT line starting with `::kargo::out::` is treated as an output
- If there is no value part, e.g. `::kargo::out::key::` or `::kargo::out::key`, the value of the output will be empty
- Output ends at line break. **Multiline output is not supported at the moment**

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
