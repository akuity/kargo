---
sidebar_label: With Argo CD
---

# Installation with Argo CD

This section outlines a few generalized approaches to installing and managing Kargo with non-default configuration options using Argo CD.

:::note
This section assumes that you have already installed any dependencies or prerequisites required for running Kargo on a Kubernetes cluster. Please refer to [Basic Installation](../../40-operator-guide/10-basic-installation.md#prerequisites) for more details.
:::

## Direct Chart Installation

The most common way to deploy Kargo using Argo CD is to create an `Application` and use the Helm chart directly. Using this method, you can use the `.spec.source.helm.parameters` section to specify any parameters you may need. This is the most straightforward way to deploy Kargo using Argo CD.

:::info
If using the `api.adminAccount.passwordHash` parameter, you must escape the `$` character with `$$` to prevent Helm from interpreting it as a variable. Please see [this discussion](https://discord.com/channels/1138942074998235187/1138946346217394407/1267966083168469102) for more information.
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

Conversely, insetad of using the `parameters` field under the `.spec.source.helm` section; you can use the `values` block or `valuesObject` object to specify the values for the Kargo Helm chart.

Another method is to use `.spec.sources` and store your values files in a separate repository. This is useful if you are using GitOps to track your values configuration changes, which still use the public Helm chart repository.

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

Here, the `parametes` section isn't used and instead, the `values.yaml` file is hosted in a separate repository and is referenced using the `ref` field.

## Argo CD Kustomize Application

Another method to deploy Kargo using Argo CD is to use a Kustomize. This method is useful if you want to customize the Kargo deployment using Kustomize overlays or patching. To do this, you will need to add the Kargo Helm chart to your `kustomization.yaml` file using the `helmCharts` field.

:::info
The `valueFile` field references a `values.yaml` file in the same directory as the `kustomization.yaml` file.
:::

```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

helmCharts:
- name: kargo
  version: 1.2.0
  repo: oci://ghcr.io/akuity/kargo-charts
  releaseName: kargo
  valuesFile: values.yaml
```

In the overlay, you can then reference the Kargo Helm chart and apply any patches or customizations you need. For example, you can change the log level of the controller to `DEBUG`

```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

resources:
- ../../base

patches:
- target:
    kind: ConfigMap
    name: kargo-controller
    version: v1
  patch: |
    - op: replace
      path: /data/LOG_LEVEL
      value: "DEBUG"
```

The corresponding Argo CD `Application` would look like this (referencing the Kustomize `dev` overlay, in this example):

:::warning
Using Helm with Kustomize requires you to make an Argo CD configuration change. Please see [the offical Argo CD documentation](https://argo-cd.readthedocs.io/en/stable/user-guide/kustomize/#kustomizing-helm-charts) for more details.
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
    repoURL: https://github.com/<username>/kargo-helm-values
    path: kustomize/overlays/dev
    targetRevision: main
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
    - CreateNamespace=true
```

## What's Next?

Now that you have deployed Kargo using Argo CD, you can explore the various features and capabilities of Kargo. Please see the [Operator Guide](../../operator-guide/) or the [User Guide](../../user-guide/) for futher information.