---
sidebar_label: With Argo CD
---

# Installation with Argo CD

This section outlines a few generalized approaches to installing and managing Kargo with non-default configuration options using Argo CD.

:::note
This section assumes that you have already installed any dependencies or prerequisites required for running Kargo on a Kubernetes cluster. Please refer to [Basic Installation](../../operator-guide/basic-installation#prerequisites) for more details.
:::

All methods described here will involve deploying Kargo using an Argo CD
`Application` resource that is configured to obtain Kargo's Helm chart directly
from its official repo. We will demonstrate a variety of ways to specify
your own configuration values using `api.adminAccount.passwordHash` and `api.adminAccount.tokenSigningKey` as examples since you are _required_ to
provide values for these anyway (unless
[the admin account is disabled instead](../40-security/10-secure-configuration.md#disabling-the-admin-account)).

Recommended commands for generating a complex password and signing key, and for hashing the password as required are:

```console
pass=$(openssl rand -base64 48 | tr -d "=+/" | head -c 32)
echo "Password: $pass"
echo "Password Hash: $(htpasswd -bnBC 10 "" $pass | tr -d ':\n')"
echo "Signing Key: $(openssl rand -base64 48 | tr -d "=+/" | head -c 32)"
```

:::note
Methods of securing the admin account are explored in greater detail [here](../40-security/10-secure-configuration.md#securing-the-admin-account).
:::

## `spec.source.helm.parameters`

The most straightforward way to specify chart configuration options is by using the
`Application`'s `spec.source.helm.parameters` field:

:::info
The parameters used are just examples, and you should use the values that are appropriate for your environment. Detailed information about available options can also be found in the [Kargo Helm Chart's README.md](https://github.com/akuity/kargo/tree/main/charts/kargo).
:::

```yaml
---
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
    targetRevision: 1.2.0
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

Alternatively, instead of using `spec.source.helm`'s `parameters` field, you can use the either of the `values` or `valuesObject` fields to specify configuration options for the chart:

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
    targetRevision: 1.2.0
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

__Our recommended method__ is to use an `Application` with
[multiple sources](https://argo-cd.readthedocs.io/en/stable/user-guide/multiple_sources/) to reference _both_ the Kargo Helm chart repository a `values.yaml`
of your own from your own Git repository.

__This is our recommendation because it aligns best with with GitOps principles.__


In the configuration below, the second source (the one with `repoURL` pointed at your own Git repository) is assigned a `ref` of `values`. This permits content from that
repository (in particular, a `values.yaml` file) to be referenced by the _other_ source:

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
      targetRevision: 1.2.0
      helm:
        valueFiles:
          - $values/kargo/values.yaml
    - repoURL: https://github.com/<username>/kargo-helm-values
      targetRevision: main
      ref: values
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
    - CreateNamespace=true
```