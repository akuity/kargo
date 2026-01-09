---
description: Learn how ambient credentials work
sidebar_label: Ambient Credentials
---

# Ambient Credentials

This section provides guidance on configuring Kargo and various cloud platforms
to support "ambient" credentials — credentials automatically available based on
the execution environment rather than stored in `Secret`s.

## Amazon Elastic Container Registry (ECR)

Kargo can be configured to authenticate to ECR repositories using EKS Pod
Identity _or_ IAM Roles for Service Accounts (IRSA).

If Kargo locates no `Secret` resources matching a repository URL and is deployed
within an EKS cluster, it will attempt to use
[EKS Pod Identity](https://docs.aws.amazon.com/eks/latest/userguide/pod-identities.html)
or
[IAM Roles for Service Accounts (IRSA)](https://docs.aws.amazon.com/eks/latest/userguide/iam-roles-for-service-accounts.html)
to authenticate. Leveraging either eliminates the need to store ECR credentials
in a `Secret` resource.

Follow
[this overview](https://docs.aws.amazon.com/eks/latest/userguide/pod-identities.html#pod-id-setup-overview)
to set up EKS Pod Identity in your EKS cluster or
[this one](https://docs.aws.amazon.com/eks/latest/userguide/iam-roles-for-service-accounts.html)
to set up IRSA. For either, you will assign an IAM role to the
`kargo-controller` `ServiceAccount` within the namespace in which Kargo is (or
will be) installed.

:::note

To use IRSA, you will additionally need to specify the
[ARN](https://docs.aws.amazon.com/IAM/latest/UserGuide/reference-arns.html) of
the controller's IAM role as the value of the `controller.serviceAccount.iamRole` setting in Kargo's Helm chart at installation.

:::

At this point, an IAM role will be associated with the Kargo _controller_,
however, that controller acts on behalf of multiple Kargo Projects, each of
which may require access to _different_ ECR repositories. To account for this,
when Kargo attempts to access an ECR repository on behalf of a specific Project,
it will first attempt to
[assume an IAM role](https://docs.aws.amazon.com/STS/latest/APIReference/API_AssumeRole.html)
specific to that Project. The name of the role it attempts to assume will
_always_ be of the form `kargo-project-<project name>`. It is this role that
should be granted read-only access to applicable ECR repositories.

:::info

The name of the IAM role associated with each Kargo Project is deliberately not
configurable to prevent Project admins from attempting to coerce Kargo into
assuming arbitrary IAM roles.

:::

:::caution

For optimal adherence to the principle of least privilege, the IAM role
associated with the `kargo-controller` `ServiceAccount` should be limited only
to the ability to assume Project-specific IAM roles. Project-specific IAM roles
should be limited only to read-only access to applicable ECR repositories.

:::

:::info

If the Kargo controller is unable to assume a Project-specific IAM role, it will
fall back to using its own IAM role directly. For organizations without strict
tenancy requirements, this can eliminate the need to manage a large number of
Project-specific IAM roles. While useful, this approach is not strictly
recommended.

:::

Tokens Kargo obtains for accessing any specific ECR repository on behalf of any
specific Kargo Project are valid for 12 hours and cached for 10. A controller
restart clears the cache.

## Google Artifact Registry (GAR)

Kargo can be configured to authenticate to
[Google Artifact Registry](https://cloud.google.com/artifact-registry/docs/overview)
(GAR) repositories using
[Workload Identity Federation](https://cloud.google.com/kubernetes-engine/docs/how-to/workload-identity)
(WIF).

If Kargo locates no `Secret` resources matching a repository URL, and if Kargo
is deployed within a
[Google Kubernetes Engine](https://cloud.google.com/kubernetes-engine/docs/concepts/kubernetes-engine-overview)
(GKE) cluster with WIF enabled, it will attempt to use it to authenticate.
Leveraging this option eliminates the need to store credentials in a `Secret`
resource. WIF can be enabled when creating a
[new cluster](https://cloud.google.com/kubernetes-engine/docs/how-to/workload-identity#enable_on_cluster)
or can be added to an
[existing cluster](https://cloud.google.com/kubernetes-engine/docs/how-to/workload-identity#enable-existing-cluster).

:::note

Clusters managed by
[GKE Autopilot](https://cloud.google.com/kubernetes-engine/docs/concepts/autopilot-overview)
have WIF enabled automatically.

:::

With WIF enabled,
[GCP Identity and Access Management](https://cloud.google.com/iam/docs/overview)
(IAM) automatically understands a
[principal](https://cloud.google.com/iam/docs/overview#principals)
identifier of the following form to be a reference to the Kargo controller's
_Kubernetes_ Service Account (KSA):

```text
principal://iam.googleapis.com/projects/<gcp project number>/locations/global/workloadIdentityPools/<gcp project name>.svc.id.goog/subject/ns/<kargo namespace>/sa/kargo-controller
```

:::note

There is no need to annotate the Kargo controller's KSA in any specific way to
enable the above.

:::

Because the Kargo controller acts on behalf of multiple Kargo Projects, each of
which may require access to _different_ GAR repositories, when accessing a
repository on behalf of a given Project, it will attempt to
[impersonate](https://cloud.google.com/iam/docs/service-account-impersonation)
a Project-specific
[Google Service Account](https://cloud.google.com/iam/docs/service-accounts-create)
(GSA). The name of the GSA that the controller will attempt to impersonate will
_always_ be of the form `kargo-project-<kargo project name>@<gcp project
name>.iam.gserviceaccount.com`.

:::info

The name of the GSA associated with each Kargo Project is deliberately not
configurable to prevent Project admins from attempting to coerce Kargo into
impersonating arbitrary GSAs.

:::

To enable this, each Project-specific GSA must:

* Have an
  [IAM policy](https://cloud.google.com/iam/docs/reference/rest/v1/Policy) that
  permits the Kargo controller's KSA to impersonate the GSA by creating a token
  (`roles/iam.serviceAccountTokenCreator`).

* Be granted read-only access (`roles/artifactregistry.reader`) to the specific
  GAR repositories with which it interacts.

:::caution

Following the principle of least privilege, the IAM principal associated with
the Kargo controller's GSA should be granted no permissions beyond the ability
to impersonate Project-specific GSAs.

:::

:::note

Beginning with Kargo `v1.5.0`, if maintaining a separate GSA for every Kargo
Project is deemed too onerous and strict adherence to the principle of least
privilege is not a concern, permissions may be granted directly to the Kargo
controller's KSA. In the event that a Project-specific GSA does not exist or
cannot be impersonated, Kargo will fall back on using the controller's KSA
directly to access GAR repositories. While useful, this approach is not strictly
recommended.

:::

Tokens Kargo obtains for accessing any specific GAR repository on behalf of any
specific Kargo Project are valid for 60 minutes and cached for 40. A controller
restart clears the cache.

## Azure Container Registry (ACR)

Kargo can be configured to authenticate to ACR repositories using
[Azure Workload Identity](https://learn.microsoft.com/en-us/azure/aks/workload-identity-overview).

If Kargo locates no `Secret` resources matching a repository URL and is deployed
within an AKS cluster with workload identity enabled, it will attempt to use it
to authenticate. Leveraging this eliminates the need to store ACR credentials in
a `Secret` resource. Workload Identity can be enabled when creating a
[new cluster](https://learn.microsoft.com/en-us/azure/aks/workload-identity-deploy-cluster#create-an-aks-cluster)
or can be added to an
[existing cluster](https://learn.microsoft.com/en-us/azure/aks/workload-identity-deploy-cluster#update-an-existing-aks-cluster).

:::danger

Azure Workload Identity can be complex to configure and difficult to
troubleshoot.

Before continuing, be certain of the following:

* Your AKS cluster has the **OIDC Issuer** feature enabled.
* Your AKS cluster has the **Workload Identity** feature enabled.

:::

For Workload Identity to work, the Kargo controller's Kubernetes
`ServiceAccount` will need to be federated with a __managed identity__. Follow
[these instructions](https://learn.microsoft.com/en-us/azure/aks/workload-identity-deploy-cluster#create-a-managed-identity)
to create one and
[these](https://learn.microsoft.com/en-us/azure/aks/workload-identity-deploy-cluster#create-the-federated-identity-credential)
to federate it with the controller's `ServiceAccount`.

:::info

Federating the managed identity to the Kargo controller's `ServiceAccount`
establishes a trust relationship. In AKS clusters with Workload Identity
enabled, a
[mutating admission webhook](https://kubernetes.io/docs/reference/access-authn-authz/admission-controllers/#mutatingadmissionwebhook)
will intercept the creation of any `Pod` resource labeled with
`azure.workload.identity/use: "true"` _and_ using a `ServiceAccount` that's been
federated to a managed identity. Knowing such a `Pod` is authorized to act on
behalf of the associated managed identity, the webhook will modify the `Pod`'s
spec to inject credentials in a well-known location for discovery by any Azure
clients executing within any of its containers.

:::

To access container images or Helm charts hosted in ACR, the managed identity
[must be granted the `AcrPull` role](https://learn.microsoft.com/en-us/azure/container-registry/container-registry-authentication-managed-identity?tabs=azure-cli#grant-identity-access-to-the-container-registry)
on the registry or on individual repositories within it.

:::danger

Before continuing, be certain of the following:

* You have created a **User-Assigned Managed Identity**.

  ⚠️ This is different from an App Registration!

* You have created a **Federated Identity Credential** that associates the
  managed identity with the Kubernetes `ServiceAccount` used by the Kargo
  controller. (In a typical installation of Kargo, this is the
  `kargo-controller` `ServiceAccount` in the `kargo` namespace.)

* The managed identity has been granted the **`AcrPull` role** on your ACR
  registry or specific repositories within it.

:::

For Workload Identity to inject credentials into any `Pod`, two specific Kargo
configuration settings are required:

1. Controller `Pod`s must be labeled with `azure.workload.identity/use: "true"`.

    This label can be affixed to Kargo controller `Pod`s by using the
    `controller.podLabels` setting in Kargo's Helm chart at the time of
    installation or upgrade.

1. The controller's `ServiceAccount` must be annotated with
   `azure.workload.identity/client-id: <managed identity client id>`.

    :::warning

    Azure documentation states this annotation is optional, however, in
    practice, it often _is_ required.
    :::

    This annotation can be affixed to the Kargo controller's `ServiceAccount` by
    using the `controller.serviceAccount.annotations` setting in Kargo's Helm
    chart at the time of installation or upgrade.

Example Helm values:

```yaml
controller:
  podLabels:
    azure.workload.identity/use: "true"
  serviceAccount:
    annotations:
      azure.workload.identity/client-id: <managed identity client id>
```

:::info

For further guidance on this, refer to the advanced installation guides for
[Helm](../20-advanced-installation/10-advanced-with-helm.md)
or [Argo CD](../20-advanced-installation/20-advanced-with-argocd.md)

:::

:::warning

If the `azure.workload.identity/use: "true"` label is present on the Kargo
controller's `Pod` and the `azure.workload.identity/client-id` annotation is
also present on the Kargo controller's `ServiceAccount`, _but_ the `Pod` was
started prior to Workload Identity having been enabled in the cluster or prior
to the controller's `ServiceAccount` having been federated with a managed
identity, the `Pod` will not have been injected with necessary credentials. Such
a `Pod` should be deleted. The controller's `Deployment` will create a
replacement `Pod` which will be injected with necessary credentials.

:::

:::caution

For optimal adherence to the principle of least privilege, the managed identity
associated with the `kargo-controller` `ServiceAccount` should be limited only
to the `AcrPull` role on the specific ACR repositories required by your Kargo
Projects.

:::

Tokens Kargo obtains for accessing any specific ACR repository are valid for
approximately 3 hours and cached for 2.5 hours. A controller restart clears the
cache.

:::note

When authenticating to ECR using EKS Pod Identity or IRSA (Amazon), or when
authenticating to GAR using Workload Identity Federation (Google), the option
exists for strict adherence to the principle of least privilege by granting the
identity associated with the Kargo controller no permissions other than those
required to assume/impersonate other, Project-specific identities.
Project-specific identities can then be granted access only to the specific
registries or repositories.

Assuming/impersonating a Project-specific identity in Azure is considerably more
complex than doing so in AWS or GCP. As a result, the Kargo controller lacks the
option described above for Azure Workload Identity / ACR.

:::
