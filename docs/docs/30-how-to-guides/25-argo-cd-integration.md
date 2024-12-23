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

:::note
This page is a work in progress. Thank you for your patience as as we work to add more details.
:::

## Updating Argo CD Applications

In the course of orchestrating the transition of an application instance
from one state to another, it is common for Kargo to updated Argo CD
`Application` resources in some way. Such updates are enabled through the
use of the
[`argocd-update` promotion step](./35-references/10-promotion-steps.md#argocd-update).
Often, these updates entail little more than modifying an `Application`'s 
`operation` field to force the `Application` to be synced to recently
updated desired state.

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
