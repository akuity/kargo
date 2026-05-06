---
description: Learn how to set up workload identity for promotion pods
sidebar_label: Promotion Pod Identity
---

<span class="tag professional"></span>
<span class="tag beta"></span>

:::info

This document describes how to set workload identity for promotions run by
Promotion Controller in Kargo versions v1.11 and above.  

Promotion Controller is only available in Kargo on the [Akuity Platform](https://akuity.io/akuity-platform)
and requires use of self-hosted agent.

:::

# Promotion Pod identity

## Background

Most workload identity tools are using pod serviceaccounts to provision access for tools running in the pod.

When using Promotion Controller, each promotion is run as a pod with steps running as containers in this pod.
This makes it possible for steps to access workload identity for the promotion pod and have required permissions.

Promotion pods are assigned `kargo-promotion-orchestrator` serviceaccount by default.
While it is technically possible to assign federated identity (e.g. IRSA) to `kargo-promotion-orchestrator`
serviceaccount, Akuity Platform may override these setting for example when upgrading or configuring Kargo.

Also using the same serviceaccount to access multiple different roles for different promotions/stages/steps is not the
best approach from security or configuration point of view.

## Promotion decorator annotation for Stages

Since version `1.11`, Kargo supports decorator annotations for `Stage` resources, which will change how
Promotion Controller runs promotion pods.

Supported decorator annotations are:

- `ee.kargo.akuity.io/promotion-sa` - **overrides** the serviceaccount used to run the promotion pod
- `ee.kargo.akuity.io/promotion-labels` - **extends** the labels used in the promotion pod
- `ee.kargo.akuity.io/promotion-annotations` - **extends** the annotations used in the promotion pod

Each annotation can be set separately. Empty values have no effect.

:::warning

If not empty, `promotion-sa` must be a valid serviceaccount in the agent namespace.
`promotion-labels` must be a JSON string with valid k8s labels.
`promotion-annotations` must be a JSON string with valid k8s annotations.

Promotion will error if values in these annotations are invalid.

:::

## Example using decorators for a stage

For example if we have a custom step accessing S3:

```
apiVersion: ee.kargo.akuity.io/v1alpha1
kind: CustomPromotionStep
metadata:
  name: aws-s3-ls
spec:
  image: amazon/aws-cli
  command: [
    "sh", 
    "-c", 
    "mkdir -p $HOME; aws s3 ls s3://{{ config.bucket }}"]
  env:
    - name: HOME ## HOME is required to run as non-root 
      value: /tmp/home
```

And we have a serviceaccount configured to access a role with S3 access:
```
apiVersion: v1
kind: ServiceAccount
metadata:
  annotations:
    eks.amazonaws.com/role-arn: arn:aws:iam::111122223333:role/access-s3
  labels:
    app.kubernetes.io/managed-by: eksctl
  name: s3-access
  namespace: akuity
```

We can configure a stage to use this serviceaccount and access the role:

```
apiVersion: kargo.akuity.io/v1alpha1
kind: Stage
metadata:
  name: my-stage
  namespace: kargo-demo
  annotations:
  	ee.kargo.akuity.io/promotion-sa: s3-access
spec:
  promotionTemplate:
    spec:
      steps:
        - as: list s3
          config:
            bucket: my_bucket_name
          uses: aws-s3-ls
  requestedFreight:
  ...
```

## Known limitations

Because entire promotion is running in a single pod, all steps in the pod will have the same identity and will get the same
access provisioned into them.

This means steps within the same promotion cannot use different serviceaccounts. If they need different permissions, those
need to be attached to the same serviceaccount.

Some builtin steps require serviceaccount to have specific k8s role bindings.
Namely when running `argocd-update` with incluster ArgoCD (on the same cluster as promotion controller agent),
the serviceaccount used in promotion must be bound to `kargo-promotion-token-manager` role.

You can create a binding for serviceaccount like this:
```
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: kargo-promotion-token-manager-orchestrator
  namespace: akuity ## Promotion namespace
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: "kargo-promotion-token-manager"
subjects:
  - kind: ServiceAccount
    namespace: akuity ## Promotion namespace
    name: <my-service-account>
```

**NOTE** this only applies to a setup where ArgoCD runs in the same cluster as promotion pods.