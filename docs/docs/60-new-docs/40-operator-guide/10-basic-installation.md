---
sidebar_label: Basic Installation
description: Learn how to do a basic installation of Kargo using Helm
---

# Basic Installation

Installing Kargo with its default configuration is straightforward and quick. Follow this guide to get Kargo up and running using Helm.

### Prerequisites

Ensure you have the following tools and environment set up before proceeding:

- **[Helm](https://helm.sh/docs/)**: These instructions are verified for Helm v3.13.1 and higher.
- A **Kubernetes cluster** with the following components installed (or higher versions):
    - [cert-manager](https://cert-manager.io/) v1.11.5+
    - [Argo CD](https://argo-cd.readthedocs.io) v2.9.3+
    - [Argo Rollouts](https://argoproj.github.io/argo-rollouts/) v1.6.4+
    - Kubernetes v1.27.4+

#### Important Notes

:::info
`cert-manager` is used to self-sign the certificate for Kargo's webhook server, allowing secure communication with
the Kubernetes API server. If you prefer not to use `cert-manager` for this purpose, you can provision your own
certificate. For details, Refer to the [Advanced Installation](./advanced-installation/advanced-with-helm) page for more information.
:::

:::info
The Argo CD and Argo Rollouts components are currently required but may become *suggested* dependencies in future releases.
:::

:::note
If Argo CD manages multiple clusters, install Kargo in the same cluster
as the Argo CD control plane, *not* in the individual clusters.
:::

### Installation Steps

To install Kargo with the default configuration and set a user-specified admin password, run the following command:

```shell
helm install kargo \
oci://ghcr.io/akuity/kargo-charts/kargo \
--namespace kargo \
--create-namespace \
--set api.adminAccount.passwordHash='$2a$10$Zrhhie4vLz5ygtVSaif6o.qN36jgs6vjtMBdM6yrU1FOeiAAMMxOm' \
--set api.adminAccount.tokenSigningKey=iwishtowashmyirishwristwatch \
--wait
```

#### Security Note

:::caution
For clusters exposed to the internet, consider the following options for securing your installation:
- Disable the admin account: `--set api.adminAccount.enabled=false`
- Use a strong, custom password and signing key.
:::

If Kargo is deployed successfully, you can access its dashboard at [localhost:31444](https://localhost:31444/), and you may need to log in using default credentials (`admin` for both username and password unless specified otherwise).

:::note
If Kargo installation fails with a `401`, verify that you are using Helm v3.13.1
or greater.

If Kargo installation fails with a `403`, it is likely that Docker is configured
to authenticate to `ghcr.io` with an expired token. The Kargo chart and images
are accessible anonymously, so this issue can be resolved simply by logging out:

```shell
docker logout ghcr.io
```

:::
