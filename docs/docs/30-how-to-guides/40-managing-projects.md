---
description: Learn how to manage Projects
sidebar_label: Managing Projects
---

# Managing Projects

### Namespace Creation and Adoption in Kargo Projects

When a new Kargo `Project` is created, it automatically generates a corresponding Kubernetes `namespace`.
In scenarios where specific configuration requirements are needed, Kargo offers an adoption feature for
pre-existing namespaces.

Kargo can adopt namespaces that you create beforehand, if they are labeled
with `kargo.akuity.io/project: "true"`.
This allows you to pre-configure namespaces with necessary labels and resources.

* Define the `namespace` in your YAML manifest, adding any required labels 
and resources. Ensure the label `kargo.akuity.io/project: "true"` is applied to the `namespace`.
* When using a YAML file, list the `namespace` definition above the Kargo `Project` resource to ensure it is created first.

In the following example, the `namespace` is labeled with `eso.example.com.au: cluster-secret-store-alpha` previously.
When the Kargo `Project` is created, it automatically adopts this pre-existing `namespace`.

```
apiVersion: v1
kind: Namespace
metadata:
  name: kargo-example
  labels:
    kargo.akuity.io/project: "true"
    eso.example.com: cluster-secret-store
---
apiVersion: kargo.akuity.io/v1alpha1
kind: Project
metadata:
    name: kargo-example
spec:
    # Project specifications go here
```

This process allows the Kargo `Project` to recognize and use your pre-configured `namespace` without needing further updates or intervention.
