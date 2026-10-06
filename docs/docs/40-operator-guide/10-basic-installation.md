---
sidebar_label: Basic Installation
description: Learn how to do a basic installation of Kargo using Helm
---

# Basic Installation

Installing Kargo's __cluster-side components__ (e.g. controllers and API server)
with __default configuration__ is quick and easy.

:::info[Not what you were looking for?]

If you're looking for a more customized installation of cluster-side components,
refer to
[Advanced Installation](./20-advanced-installation/10-advanced-with-helm.md)
for steps, and
[Common Configurations](./20-advanced-installation/30-common-configurations.md)
for guidance on configuring Kargo to address common operational concerns.

If you're a developer looking for instructions for installing the Kargo CLI, see
[Installing the CLI](../50-user-guide/05-cli/10-installation.md).

:::

:::caution

The default configuration is suitable _only_ for trying Kargo in a local cluster
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

1. Generate a password, a signing key, and a database password.

    There are no default values for these three fields, so you _must_ provide
    your own.

    Recommended commands for generating a complex password and signing key, for
    hashing the password as required, and for generating a password for the
    bundled database are:

    ```console
    pass=$(openssl rand -base64 48 | tr -d "=+/" | head -c 32)
    echo "Password: $pass"
    hashed_pass=$(htpasswd -bnBC 10 "" $pass | tr -d ':\n')
    signing_key=$(openssl rand -base64 48 | tr -d "=+/" | head -c 32)
    db_pass=$(openssl rand -base64 48 | tr -d "=+/" | head -c 32)
    ```

    The above commands will leave you with values assigned to `$hashed_pass`,
    `$signing_key`, and `$db_pass`. These will be used in the next step.

1. Install Kargo with default configuration and your chosen admin account
   password:

    ```shell
    helm install kargo \
      oci://ghcr.io/akuity/kargo-charts/kargo \
      --namespace kargo \
      --create-namespace \
      --set api.adminAccount.passwordHash=$hashed_pass \
      --set api.adminAccount.tokenSigningKey=$signing_key \
      --set database.postgres.password=$db_pass \
      --wait
    ```

## Database

The chart installs a minimal PostgreSQL instance alongside Kargo. It runs as
a single, unreplicated instance with no backups or tuning and is meant for
evaluation and development. To use your own database instead, set
`database.postgres.enabled` to `false` and point `database.external.secretName`
at a Secret holding its connection string.

The bundled instance's `kargo` user is protected by the password you provide
as `database.postgres.password`, as in the installation steps above. Like the
admin account's token signing key, it is stored in plaintext. To keep it out of
your values, create a Secret in Kargo's namespace holding the password under
the key `password` and set `database.postgres.existingSecret` to its name; the
chart then stores no password itself. This is the recommended approach when
installing with a GitOps tool such as
[Argo CD](./20-advanced-installation/20-advanced-with-argocd.md#database-password).

### Schema migrations

On every install and upgrade, the chart runs a Job that applies any pending
database schema migrations. The migrations are embedded in the Kargo image,
and the Job applies them with that image's `migrate` subcommand, so the
schema it produces is the one the installed version expects. It waits for the
database to accept connections, serializes with any other runner on a
database-level lock, and keeps its Pod for a day after finishing so that its
logs can be inspected.

The Job is an ordinary resource rather than a Helm hook, so it also works
when the chart is rendered with `helm template`, as GitOps tools do. Ordering
does not matter: the Job waits for the database, and Kargo's components
refuse to serve against a schema that is behind the version they expect.

To apply migrations yourself, for example from a pipeline that manages schema
changes with its own approvals, set `database.migrations.enabled` to `false`
and run the same subcommand from the Kargo image against your database:

```shell
docker run --rm -e DATABASE_URL='postgres://...' ghcr.io/akuity/kargo:<version> migrate
```

:::note
The `migrate` subcommand belongs to the control plane binary in the Kargo
image, not to the `kargo` CLI you install locally.
:::

Migrations are forward-only and each release's schema remains compatible with
the previous release's code, so a rollback of Kargo does not require a
rollback of the schema.

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
