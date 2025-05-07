---
sidebar_label: With Helm
description: Learn how to perform an advanced installation of Kargo using Helm
---

# Advanced Installation with Helm

This section outlines the general procedure for customizing configuration of
Kargo's cluster-side components when installing them via Helm.

1. Extract the default values from the Helm chart and save it to a convenient
location. In the example below, we save it to `kargo-values.yaml`

    ```shell
    helm inspect values \
      oci://ghcr.io/akuity/kargo-charts/kargo > kargo-values.yaml
    ```

1. Edit and save the values.

    :::info
    You will find this configuration file contains helpful comments for every
    option, so specific options are not covered in detail here.

    Detailed information about available options can also be found in the
    [Kargo Helm Chart's README.md](https://github.com/akuity/kargo/tree/main/charts/kargo).

    Additionally, for important security-related configuration, check the [Secure Configuration Guide](../40-security/10-secure-configuration.md).
    :::

1. Proceed with installation, using your modified values:

    ```shell
    helm install kargo \
      oci://ghcr.io/akuity/kargo-charts/kargo \
      --version 1.4.4 \
      --namespace kargo \
      --create-namespace \
      --values kargo-values.yaml \
      --wait
    ```
