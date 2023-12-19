---
description: Learn how to install Kargo using this step-by-step guide
sidebar_label: Installing Kargo
---

# Installing Kargo

## Basic Installation

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
`cert-manager` is used for self-signing a certificate used to identify Kargo's
webhook server to the Kubernetes API server. If you do not wish to use
`cert-manager` for this purpose, you may provision your own certificate through
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

The following command will install Kargo with default configuration and a
user-specified admin password:

```shell
helm install kargo \
  oci://ghcr.io/akuity/kargo-charts/kargo \
  --namespace kargo \
  --create-namespace \
  --set api.adminAccount.password=admin \
  --set api.adminAccount.tokenSigningKey=iwishtowashmyirishwristwatch \
  --wait
```

:::caution
If deploying to an internet-facing cluster, be certain to do one of the
following:

* Disable the admin account with `--set api.adminAccount.enabled=false`

* Choose your own strong password and signing key. 
:::

## Advanced Installation

1. Extract the default values from the Helm chart and save it to a convenient
   location. In the example below, we save it to `~/kargo-values.yaml`

   ```shell
   helm inspect values \
     oci://ghcr.io/akuity/kargo-charts/kargo > ~/kargo-values.yaml
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
     --namespace kargo \
     --create-namespace \
     --values ~/kargo-values.yaml \
     --wait
   ```
