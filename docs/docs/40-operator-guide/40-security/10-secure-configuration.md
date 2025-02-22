---
sidebar_label: Secure Configuration
---

# Secure Configuration

The purpose of this document is to direct operators' attention to specific
security considerations that should be taken into account at installation time,
as well as calling out the specific configuration options that address them.

:::info
If you're only installing Kargo into a local Kubernetes cluster for testing
purposes, installing Kargo with the default configuration should be sufficient.
:::

:::info
Whether you're installing Kargo
[using Helm](../20-advanced-installation/10-advanced-with-helm.md) or
[via Argo CD](../20-advanced-installation/20-advanced-with-argocd.md), this
document assumes familiarity with procedures for configuring that installation.

Refer to the
[Kargo Chart's README](https://github.com/akuity/kargo/tree/main/charts/kargo)
for detailed documentation of all configuration options.
:::

## Securing the API Server

Since users interact with Kargo's API server (indirectly via the UI or CLI), the
API server has the distinction of being Kargo's most attackable component. This
makes it especially important to ensure that it is configured with secure
options. This is doubly important if your Kargo API server is exposed to the
internet -- which is possibly the case for some highly distributed
organizations.

The following sections will enumerate specific considerations and how to
address them.

### Disabling the Admin Account

Kargo's default configuration enables an admin account, primarily for the
convenience of new users who are likely to be installing Kargo into a local
Kubernetes cluster and are not likely to configure SSO via OpenID Connect.
The admin account is highly privileged, which makes it an appealing target.

__We strongly recommend disabling the admin account in any environment other
than a local Kubernetes cluster.__

Disabling the admin account can be done at installation time by setting
`api.adminAccount.enabled` to `false`.

:::note
The admin account is Kargo's _only_ built-in account. When disabling it, as we
recommend, this effectively _requires_ that you configure SSO via OpenID
Connect.

Refer to [SSO with OpenID Connect](20-openid-connect.md) for in-depth coverage
of this topic.
:::

### Securing the Admin Account

If, for some reason, you must leave the admin account enabled, __you must
provide a bcrypt-hashed password for the account and a signing key__ that the
API server will use to sign tokens (JWTs) for the account.

1. Generate and base64 encoded a bcrypt-hashed password and a signing key:

    ```console
    pass=$(openssl rand -base64 48 | tr -d "=+/" | head -c 32)
    echo "Password: $pass"
    hashed_pass=$(htpasswd -bnBC 10 "" $pass | tr -d ':\n')
    echo "Password Hash: $hashed_pass"
    echo "Encoded Password Hash: $(echo -n "$hashed_pass" | base64)"
    echo "Encoded Signing Key: $(openssl rand -base64 48 | tr -d "=+/" | head -c 32 | base64)"
    ```

1. Create a `Secret` resource in the same namespace in which Kargo is installed
with the following format:

    ```yaml
    apiVersion: v1
    kind: Secret
    type: Opaque
    metadata:
      name: <secret name>
      namespace: <kargo namespace>
    data:
      ADMIN_ACCOUNT_TOKEN_SIGNING_KEY: <base64 encoded signing key>
      ADMIN_ACCOUNT_PASSWORD_HASH: <base64 encoded bcrypt-hashed password>
    ```

1. At installation time, set `api.secret.name` to the name of the `Secret`
   resource you created.

### Disabling Secret Management

First, note that the Kargo API server _never_ has cluster-scoped access to
`Secret` resources. Instead, the management controller dynamically expands and
contracts API server access to `Secret` resources in individual namespaces as
`Project` resources are created and deleted. This effectively limits the API
server to accessing `Secret` resources within project namespaces only.

__There are two compelling reasons to disallow even this limited access:__

1. Your threat model will not abide even the limited `Secret` access granted
   to the API server.

1. You GitOps your Kargo projects (which we recommend) or use some other means
   to manage `Secret` resources. With users not "click-opsing" `Secret`s, the
   need for the API server to access any `Secret`s _at all_ is entirely
   obviated.

If either of the above scenarios is applicable, all `Secret` access by the API
server can be disabled at installation time by setting
`api.secretManagementEnabled` to `false`.

### Configuring TLS

__There is no good reason not to secure inbound requests to the API server with
TLS.__ Failing to do so is a significant security risk, thus TLS is enabled by
default. The default configuration, however, makes use of a self-signed
certificate, which will not be trusted by users' browsers.

Assuming you've _not_ opted into using `Ingress` (disabled by default), __you
should provide your own certificate.__ This can be done by setting
`api.tls.selfSignedCert` to `false` and creating a `Secret` resource named
`kargo-api-cert` in the same namespace in which Kargo is installed. The `Secret`
must be formatted in the conventional manner for a certificate described
[here](https://kubernetes.io/docs/concepts/configuration/secret/#tls-secrets).

:::info
We strongly recommend using [cert-manager](https://cert-manager.io/) to manage
`Secret` resources containing TLS certificates.
:::

If you _have_ opted into using `Ingress` (`api.ingress.enabled` set to `true`),
it will default to using TLS with a self-signed certificate. To __provide
your own certificate__, you should set `api.ingress.tls.selfSignedCert` to
`false` and create a `Secret` resource named `kargo-api-ingress-cert` in the
same namespace in which Kargo is installed. Once again, such a `Secret` must be
formatted in the conventional manner for a certificate described
[here](https://kubernetes.io/docs/concepts/configuration/secret/#tls-secrets).

:::info
With `Ingress` enabled and terminating TLS, it may be tempting to disable TLS
on the API server itself. Your threat model may or may not permit this, but
we recommend against it.
:::

## Securing the Controller

### Secret Access

By default, the Kargo controller does _not_ have cluster-scoped access to
`Secret` resources. Instead, the _management controller_ dynamically expands and
contracts the controller's read-only access to `Secret` resources in individual
namespaces as `Project` resources are created and deleted. This effectively
limits the controller to accessing `Secret` resources within project namespaces
only.

:::info
It is not uncommon for Kargo controllers to be installed in clusters other than
the Kargo control plane. Restricting controller access to only `Secret`
resources within project namespaces is a measure designed to prevent exposure
of non-Kargo-project `Secret` resources in the control plane's cluster in the
event that a remote controller's credentials are compromised.
:::

It is possible to opt-in to cluster-scoped `Secret` access by setting
`controller.serviceAccount.clusterWideSecretReadingEnabled` to `true` at
installation time, __although we strongly recommend keeping the default
configuration for this option (`false`).__

:::info
The likely impetus for enabling cluster-scoped `Secret` access is to eliminate
the need for manually managing `RoleBinding`s that grant the controller
read-only access to `Secret` resources in specially designated "global
credential namespaces" as described [here](40-managing-credentials.md). __We
still do not recommend this.__
:::
