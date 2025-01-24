---
sidebar_label: With Argo CD
---

# Installation with Argo CD

This section outlines a few generalized approaches to installing and managing Kargo with non-default configuration options using Argo CD.

:::note
This section assumes that you have already installed any dependencies or prerequisites required for running Kargo on a Kubernetes cluster. Please refer to [Basic Installation](../../operator-guide/basic-installation#prerequisites) for more details.
:::

## Direct Argo CD Application 

The most common way to deploy Kargo using Argo CD is to create an `Application` and use the Helm chart directly. Using this method, you can use the `.spec.source.helm.parameters` section to specify any parameters you may need. This is the most straightforward way to deploy Kargo using Argo CD.

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
          value: "$$2a$$10$$Zrhhie4vLz5ygtVSaif6o.qN36jgs6vjtMBdM6yrU1FOeiAAMMxOm"
        - name: controller.logLevel
          value: "DEBUG"
        - name: api.adminAccount.tokenTTL
          value: "24h"
        - name: api.adminAccount.tokenSigningKey
          value: "iwishtowashmyirishwristwatch"
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
    - CreateNamespace=true
```

:::info
If using the `api.adminAccount.passwordHash` parameter in this method, you must escape the `$` character with `$$` to prevent Helm from interpreting it as a variable. Please see [this discussion](https://discord.com/channels/1138942074998235187/1138946346217394407/1267966083168469102) for more information.
:::

Conversely, insetad of using the `parameters` field under the `.spec.source.helm` section; you can use the `values` block or `valuesObject` object to specify the values for the Kargo Helm chart. Below is an example of how to use `valuesObject` to specify the values.

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
      valuesObject:
        api:
          adminAccount:
            passwordHash: $2a$10$Zrhhie4vLz5ygtVSaif6o.qN36jgs6vjtMBdM6yrU1FOeiAAMMxOm
            tokenSigningKey: iwishtowashmyirishwristwatch
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

Another method is to use a Multi-Source Argo CD Application. Here, you'd use the `.spec.sources` field and store your values files in a separate repository. This is useful if you are using GitOps to track your values configuration changes, but will still use the public Helm chart repository.

:::info
We recommend using this method as it more closely aligns with GitOps principles and best practices.
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

The `parametes` section isn't used in this method, instead the `values.yaml` file is hosted in a separate repository and is referenced using the `ref` field.

## What's Next?

Now that you have deployed Kargo using Argo CD, you can explore the various features and capabilities of Kargo. Please see the [Operator Guide](../../operator-guide/) or the [User Guide](../../user-guide/) for futher information.