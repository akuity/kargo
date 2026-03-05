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
This can be used for tasks currently not provided by built-in steps to extend kargo capabilites
with bespoke or proprietary functionality.

## Registering custom step

Custom steps have to be registered in Kargo cluster in order to be used in promotion templates.
To register a new step, kargo cluster admin needs to create a cluster-scoped `CustomPromotionStep` resource:

```
apiVersion: ee.kargo.akuity.io/v1alpha1
kind: CustomPromotionStep
metadata:
  name: hello-world
spec:
  ## REQUIRED: image to execute the command in
  image: ubuntu
  ## REQUIRED: command to run
  command: ["sh", "-c", "sleep 5; echo ::kargo::out::hello::${HELLO}-${CONFIG_WORLD}"]
  ## OPTIONAL: additional environment to provide to the command
  env:
    - name: HELLO
      value: bonjour
## OPTIONAL: error handling metadata
#  defaultTimeout: 5m
#  defaultErrorThreshold: 3
## OPTIONAL: capabilities to provision into step container
## Options: access-credentials, access-control-plane, access-argocd
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

**NOTE** the `command` field does not specify the `command` of the executor container, but a script which will be run by
the step executor.

## Using custom step

After custom step is registered in the cluster, it can be used in promotion templates:

```
vars:
- name: exampleVar
  value: example
steps:
- as: my-cusotm-step
  uses: hello-world
  config:
    world: ${{ vars.exampleVar }}
```

There is no config validation at the moment and configuration keys can be arbitrary.
Configuration values **must be strings**, there is no support for structured config yet.

### Passing input to steps

Command execution will run in the same workdir as other steps and will have the following environment variables set:

Step context variables:

- `KARGO_UI_BASE_URL` - baseURL of kargo control plane
- `KARGO_WORKDIR` - path to workdir (also will be the workdir of the script execution)
- `KARGO_ALIAS` - alias of the step
- `KARGO_PROJECT` - project reference
- `KARGO_STAGE` - stage reference
- `KARGO_PROMOTION` - promotion reference
- `KARGO_PROMOTIONACTOR` - actor triggering the promotion

Step configuration variables:

- `CONFIG_<KEY>` - configuration value for each config set in the template. Keys are in `UPPER_SNAKE_CASE`

For example in the step above `world: ${{ vars.exampleVar }}` will be evaluated to `CONFIG_WORLD=example` variable.

### Steps output

Output from step execution is parsed from STDOUT of the command execution.

For example in step above `echo ::kargo::out::hello::${HELLO}-${CONFIG_WORLD}` would print `::kargo::out::hello::bonjour-example`.
This will set the step output to `{"hello":"bonjour-example"}`.

- Each STDOUT line starting with `::kargo::out::` is treated as an output
- If there is no value part, e.g. `::kargo::out::key::` or `::kargo::out::key`, the value of the output will be empty
- Output ends at line break. **Multiline output is not supported at the moment**

## Runtime limitations

- Currently only linux containers are supported due to coordination and step execution control logic.
- Container architecture must be compatible with the executor binary from `quay.io/akuity/kargo-promotion-executor` image.
- Aborting step execution is not implemented yet, avoid long-running steps which might require aborting.
- Currently step containers run with user `65532`, images requiring specific user are not supported yet.
- Avoid using too many steps in the same promotion as they utilize the same pod and may exhaust pod/node resources.

