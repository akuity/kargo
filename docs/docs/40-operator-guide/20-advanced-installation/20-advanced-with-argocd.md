---
sidebar_label: With Argo CD
---

# Installation with Argo CD

This document outlines a few generalized approaches to installing and managing
Kargo using Argo CD.
:::note
This section assumes that you have already installed any dependencies or
prerequisites required for running Kargo on a Kubernetes cluster. Please refer
to [Basic Installation](../../operator-guide/basic-installation#prerequisites)
for more details.
:::

All methods described here will involve deploying Kargo using an Argo CD
`Application` resource that is configured to obtain Kargo's Helm chart
_directly_ from its official repository. We will demonstrate a variety of ways
to specify your own configuration values using `api.adminAccount.passwordHash`
and `api.adminAccount.tokenSigningKey` as examples since you are _required_ to
provide values for these anyway (unless
[the admin account is disabled instead](../40-security/10-secure-configuration.md#disabling-the-admin-account)),
but the techniques shown here can be applied to any configurable elements of
the Kargo Helm chart.

:::info
Detailed information about available options can be found in the
[Kargo Helm Chart's README.md](https://github.com/akuity/kargo/tree/main/charts/kargo).

For important security-related configuration, refer to the
[Secure Configuration Guide](../40-security/10-secure-configuration.md).
:::

Recommended commands for generating a complex password and signing key, and for
hashing the password as required are:

```console
pass=$(openssl rand -base64 48 | tr -d "=+/" | head -c 32)
echo "Password: $pass"
echo "Password Hash: $(htpasswd -bnBC 10 "" $pass | tr -d ':\n')"
echo "Signing Key: $(openssl rand -base64 48 | tr -d "=+/" | head -c 32)"
```

:::note
Methods of securing the admin account are explored in greater detail
[here](../40-security/10-secure-configuration.md#securing-the-admin-account).
:::

## `spec.source.helm.parameters`

The most straightforward way to specify chart configuration options is by using the
`Application`'s `spec.source.helm.parameters` field:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: kargo
  namespace: argocd
spec:
  project: default
  destination:
    namespace: kargo
    server: https://kubernetes.default.svc
  source:
    repoURL: ghcr.io/akuity/kargo-charts
    chart: kargo
    targetRevision: <desired version of Kargo>
    helm:
      parameters:
        - name: api.adminAccount.passwordHash
          # Note: A bcrypt-hashed password will contain `$` characters that
          # MUST each be escaped as `$$`
          value: <bcrypt-hashed password>
        - name: api.adminAccount.tokenSigningKey
          value: <token signing key>
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
    - CreateNamespace=true
```

## `spec.source.helm.values`

Alternatively, instead of using `spec.source.helm`'s `parameters` field, you can
use either of its `values` or `valuesObject` fields to specify configuration
options for the chart:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: kargo
  namespace: argocd
spec:
  project: default
  destination:
    namespace: kargo
    server: https://kubernetes.default.svc
  source:
    repoURL: ghcr.io/akuity/kargo-charts
    chart: kargo
    targetRevision: <desired version of Kargo>
    helm:
      valuesObject:
        api:
          adminAccount:
            passwordHash: <bcrypt-hashed password>
            tokenSigningKey: <token signing key>
            tokenTTL: 24h
        controller:
          logLevel: DEBUG
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
    - CreateNamespace=true
```

## Multi-Source Argo CD Application

The most advanced method covered here __is nevertheless our recommendation
because it aligns best with GitOps principles.__ Use an `Application`
with
[multiple sources](https://argo-cd.readthedocs.io/en/stable/user-guide/multiple_sources)
to reference _both_ the Kargo Helm chart repository and a `values.yaml` file of
your own from your own Git repository.

:::info
An added benefit to this approach is that if you have other resources to
include in the Kargo installation, such as
[`SealedSecret`s](https://github.com/bitnami-labs/sealed-secrets) or
[`ExternalSecret`s](https://external-secrets.io/latest/), they also can
be obtained from your own Git repository using the second source.
:::

In the configuration below, the second source (the one with `repoURL` pointed at
your own Git repository) is assigned a `ref` of `values`. This permits content
from that repository (in particular, a `values.yaml` file) to be referenced by
the _other_ source. We _also_ use the `path` parameter as usual to direct the
second source to the location of additional manifests to include in the `kargo`
namespace along with the chart:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: kargo
  namespace: argocd
spec:
  project: default
  destination:
    namespace: kargo
    server: https://kubernetes.default.svc
  sources:
    - repoURL: ghcr.io/akuity/kargo-charts
      chart: kargo
      targetRevision: <desired version of Kargo>
      helm:
        valueFiles:
          - $values/kargo/values.yaml
    - repoURL: https://github.com/example/repo.git
      targetRevision: main
      ref: values
      path: kargo/additional-manifests
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
    - CreateNamespace=true
```
