---
description: Learn how to integrate Kargo Stages with Argo CD Applications.
sidebar_label: Argo CD Integration
---

# Argo CD Integration

Argo CD excels at syncing Kubernetes clusters to desired state stored in
a Git repository, but lacks any concept of a "promotion", i.e. updating
the desired state of some application instance to reflect the validated
state of some other application instance. Kargo was created to orchestrate
such processes. Because such orchestration naturally entails some direct
and indirect integration with Argo CD, this page details certain key
elements of the interactions between the two systems.

## Updating Argo CD Applications

In the course of orchestrating the transition of an application instance
from one state to another, it is common for Kargo to updated Argo CD
`Application` resources in some way. Such updates are enabled through the
use of the
[`argocd-update` promotion step](../60-reference-docs/30-promotion-steps/argocd-update.md).
Often, these updates entail little more than modifying an `Application`'s 
`operation` field to force the `Application` to be synced to recently
updated desired state.

```yaml
steps:
- uses: argocd-update
  config:
    apps:
    - name: my-app
      sources:
      - repoURL: https://github.com/example/repo.git
        desiredRevision: <commit-hash>
```

:::info
For in-depth information on the usage of the `argocd-update` step, see the
[examples](../60-reference-docs/30-promotion-steps/argocd-update.md#examples).
:::

### Authorizing Updates

Performing updates of any kind to an `Application` resource naturally
requires Kargo to be _authorized_ to do so. Kargo controllers have the
requisite RBAC permissions to perform such updates, but being a
multi-tenant system, Kargo must also understand, internally, when it
is acceptable to utilize those broad permissions to update a specific 
`Application` resource _on behalf of_ a given `Stage`.

To enable Kargo controllers to update an Argo CD `Application` on behalf of
a given `Stage`, that `Application` must be explicitly annotated as follows:

```yaml
kargo.akuity.io/authorized-stage: "<project-name>:<stage-name>"
```

Because an annotation such as the one above could only be added to
an `Application` by a user who, themselves, is authorized to update
that `Application`, Kargo interprets the presence of such an annotation
as delegation of that user's authority to do so.

In the following example, an `Application` has been annotated to
authorize Kargo to update it on behalf of a `Stage` named `test`
in the `kargo-demo` `Project`:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: kargo-demo-test
  namespace: argocd
  annotations:
    kargo.akuity.io/authorized-stage: kargo-demo:test
spec:
  # Application Specifications
```

## Health Checks

When a `Promotion` uses an `argocd-update` step to update an `Application`, a
[health check](../60-reference-docs/30-promotion-steps/argocd-update.md#health-checks)
is registered for the `Stage` that the `Promotion` is targeting. This health
check is used to continuously monitor the
[health of the `Application`](https://argo-cd.readthedocs.io/en/stable/operator-manual/health/)
that was updated by the `argocd-update` step as part of the `Stage` health.

:::info
It is important to note that `Stage` health is not determined solely by the
health of the `Application` that the `Stage` is managing. The health of the
`Stage` is determined by the health of all `Application` resources that the
`Stage` is managing, as well as any other indicators of health that are
part of the `Stage`'s definition. For example, a `Stage` may be considered
unhealthy if the latest `Promotion` to that `Stage` failed.
:::
