---
sidebar_label: Basic Installation
description: Learn how to do a basic installation of Kargo using Helm
---

# Basic Installation

Installing Kargo with default configuration is quick and easy.

:::caution
The default configuration is suitable only for trying Kargo in a local cluster
that is not internet-facing.

For detailed instructions for a secure installation, refer to
[Secure Configuration](./40-security/10-secure-configuration.md).
:::

## Prerequisites

You will need:

- [Helm](https://helm.sh/docs/): These instructions were tested with v3.13.1.

- A Kubernetes cluster with [cert-manager](https://cert-manager.io/)
  pre-installed.

  :::note
  cert-manager is not an absolute dependency, but _is_ required for installation
  with the default configuration.
  :::

The following dependencies are optional, but highly recommended to be
pre-installed in your Kubernetes cluster:

- [Argo CD](https://argo-cd.readthedocs.io)

  :::info
  Kargo works best when paired with Argo CD.
  :::

- [Argo Rollouts](https://argoproj.github.io/argo-rollouts/)

  :::info
  Kargo's verification feature makes use of Argo Rollouts `AnalysisTemplate` and
  `AnalysisRun` resources internally.

  **Kargo does not require that your application deployments also use Argo
  Rollouts.**
  :::

These instructions were tested with:

- Kubernetes: v1.29.3
- cert-manager: v1.16.1
- Argo CD: v2.13.0
- Argo Rollouts: v1.7.2

## Installation Steps

1. Generate a password and a signing key.

    There are no default values for these two fields, so you _must_ provide your
    own.

    Recommended commands for generating a complex password and signing key, and
    for hashing the password as required are:

    ```console
    pass=$(openssl rand -base64 48 | tr -d "=+/" | head -c 32)
    echo "Password: $pass"
    hashed_pass=$(htpasswd -bnBC 10 "" $pass | tr -d ':\n')
    signing_key=$(openssl rand -base64 48 | tr -d "=+/" | head -c 32)
    ```

    The above commands will leave you with values assigned to `$hashed_pass` and
    `$signing_key`. These will be used in the next step.

1. Install Kargo with default configuration and your chosen admin account
   password:

    ```shell
    helm install kargo \
      oci://ghcr.io/akuity/kargo-charts/kargo \
      --namespace kargo \
      --create-namespace \
      --set api.adminAccount.passwordHash=$hashed_pass \
      --set api.adminAccount.tokenSigningKey=$signing_key \
      --wait
    ```

## Troubleshooting

### Kargo installation fails with a `401`

Verify that you are using Helm v3.13.1 or greater.

### Kargo installation fails with a `403`

It is likely that Docker is configured to authenticate to `ghcr.io` with an
expired token. The Kargo Helm chart and images are publicly accessible, so this
issue can be resolved simply by logging out:

```shell
docker logout ghcr.io
```
