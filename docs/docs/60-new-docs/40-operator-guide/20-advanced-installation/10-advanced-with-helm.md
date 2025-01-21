---
sidebar_label: With Helm
description: Learn how to perform an advanced installation of Kargo using Helm
---

# Advanced Installation with Helm

For a customized and secure Kargo setup, follow these advanced installation steps.

## Installation Steps

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

For more information on the available configuration options, refer to the [Kargo Helm Chart README](https://github.com/akuity/kargo/tree/main/charts/kargo).
