---
description: Installing Kargo
---

# Installing Kargo

## Basic installation

Installing Kargo with default configuration is quick and easy.

You will need:

* [Helm](https://helm.sh/docs/): These instructions were tested with v3.11.2.
* A Kubernetes cluster with [cert-manager](https://cert-manager.io/) and
  [Argo CD](https://argo-cd.readthedocs.io) pre-installed. These instructions
  were tested with:
    * Kubernetes: v1.25.3
    * cert-manager: v1.11.5
    * Argo CD: v2.8.3

:::info
cert-manager is used for self-signing a certificate used to identify Kargo's
webhook server to the Kubernetes API server. If you do not wish to use
cert-manager for this purpose, you may provision your own certificate through
other means. Refer to the advanced installation section for more information.
:::

:::info
We are working toward transitioning Argo CD from a required dependency to a
_suggested_ dependency.
:::

:::note
If your Argo CD control plane manages multiple Kubernetes clusters, be advised
that Kargo is intended to be installed into the same cluster as the Argo CD
control plane and _not_ into the individual clusters that Argo CD is managing.
:::

The following command will install Kargo with default configuration:

```shell
helm install kargo \
  oci://ghcr.io/akuity/kargo-charts/kargo \
  --version 0.1.0-rc.21 \
  --namespace kargo \
  --create-namespace \
  --wait
```

:::note
The `--version` flag is required for installing unstable releases, such as this
release candidate.
:::

## Advanced installation

1. Extract the default values from the Helm chart and save it to a convenient
   location. In the example below, we save it to `~/kargo-values.yaml`

   ```shell
   helm inspect values \
     oci://ghcr.io/akuity/kargo-charts/kargo \
     --version 0.1.0-rc.21 > ~/kargo-values.yaml
   ```

1. Edit and save the values.

   :::info
   You will find this configuration file contains helpful comments for every
   option, so specific options are not covered in detail here.
   :::

1. Proceed with installation, using your modified values:

   ```shell
   helm install kargo \
     oci://ghcr.io/akuity/kargo-charts/kargo \
     --version 0.1.0-rc.21 \
     --namespace kargo \
     --create-namespace \
     --values ~/kargo-values.yaml \
     --wait
   ```
