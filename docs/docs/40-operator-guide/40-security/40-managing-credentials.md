---
description: Learn to manage "global" and ambient repository credentials
sidebar_label: Managing Credentials
---

# Managing Credentials

To orchestrate the promotion of `Freight` from `Stage` to `Stage`, Kargo will
often require read/write permissions on private Git repositories and read-only
permissions on private container image or Helm chart repositories.

This section focuses on an operator's role in providing Kargo projects with
necessary credentials.

:::info
__Not what you were looking for?__

If you're user looking to learn more about managing
credentials at the project level, refer instead to the
[Managing Credentials](../../50-user-guide/50-security/30-managing-credentials.md)
section of the User's Guide.
:::

:::info
Whether you're installing Kargo
[using Helm](../20-advanced-installation/10-advanced-with-helm.md) or
[via Argo CD](../20-advanced-installation/20-advanced-with-argocd.md), the
next two sections assume familiarity with procedures for configuring that
installation.
:::

## Repository Credentials as `Secret` Resources

Kargo expects repository credentials it uses to have been stored as specially
labeled Kubernetes `Secret` resources containing specially-formatted data. These
`Secret`s generally take the following form:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: <name>
  namespace: <project namespace>
  labels:
    kargo.akuity.io/cred-type: <type>
stringData:
  repoURL: <repo url>
  username: <username>
  password: <password>
```

The names of `Secret` resources are inconsequential because Kargo matches
credentials to repositories by repository type and URL. `Secret` names may
therefore observe any naming convention preferred by the user.

The label key `kargo.akuity.io/cred-type` and its value, one of `git`, `helm`,
`image`, or `generic` is important, as it designates the `Secret` as
representing credentials for a Git, Helm chart, or container image repository,
or _something else_, respectively.

`Secret`s labeled as `git`, `image`, or `helm` credentials must generally
contain the following keys:

* `repoURL`:
  
  * The full URL of the repository the credentials are for.

  OR

  * A regular expression matching the URLs of multiple repositories for which
    the credentials may be used, with the `repoURLIsRegex` key additionally set
    to `true`.

    :::info
    This is useful if, for example, your project accesses many GitHub
    repositories, all beginning with `https://github.com/example-org`, and can
    use the same token for accessing all of them.
    :::

* Either:

  * `username`: The username to use when authenticating to the repository.

  * `password`: A password or personal access token.

    :::info
    If the value of the `password` key is a __personal access token__, the value
    of the `username` field is often inconsequential. You should consult your
    repository's documentation for more information.
    :::

  OR:

  * `sshPrivateKey`: A PEM-encoded SSH private key. Applicable only to Git
    repositories using SSH-style URLs -- for instance
    `git@github.com:example/repo.git`.

## Global Credentials

Credentials are generally managed at the project level by project admins, but
in cases where one or more sets of credentials are needed widely across many or
all Kargo projects, an operator may opt into designating one or more namespaces
as containing "global" credentials, accessible to all projects. It is then the
operator's responsibility to create and manage such credentials as well.

When Kargo searches for repository credentials, these additional namespaces are
searched only _after_ finding no matching credentials in the project's own
namespace.

:::note
__Precedence__

When Kargo searches for repository credentials in a "global" namespace, it
_first_ iterates over all appropriately labeled `Secret`s _without_
`repoIsRegex` set to `true` looking for a `repoURL` value matching the
repository URL exactly.

Only if no exact match is found does it iterate over all
appropriately labeled `Secret`s with `repoIsRegex` set to `true` looking for a
regular expression matching the repository URL.

When searching for an exact match, and then again when searching for a pattern
match, appropriately labeled `Secret`s are considered in lexical order by name.
:::

:::info
Because Kargo matches credentials to repositories by repository type and URL,
users do not need to be informed of the details (e.g. names) of any global
credentials, except possibly that they exist.
:::

### Enabling Global Credentials

To designate one or more namespaces as containing "global" credentials, list
them under the Kargo Helm chart's `controller.globalCredentials.namespaces`
option at installation time.

Operators must also manually ensure Kargo controllers receive read-only access
to `Secret`s in the designated namespaces. For example, if `kargo-global-creds`
is designated as a global credentials namespace, the following `RoleBinding`
should be created within that namespace:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
    name: kargo-controller-read-secrets
    namespace: kargo-global-creds
subjects:
    - kind: ServiceAccount
      name: kargo-controller
      namespace: kargo
roleRef:
    kind: ClusterRole
    name: kargo-controller-read-secrets
    apiGroup: rbac.authorization.k8s.io
```

:::note
The `kargo-controller-read-secrets` `ClusterRole` is predefined by the Kargo
Helm chart and grants read-only access to `Secret` resources.
:::

:::info
By default, Kargo controllers _lack_ cluster-wide permissions on `Secret`
resources. Instead, the Kargo _management controller_ dynamically expands and
contracts controller access to `Secret`s on a namespace-by-namespace basis as
new `Project`s are created and deleted.

_It is because this process does not account for "global" credential namespaces
that these bindings must be created manually by an operator._
:::

:::warning
Setting the `controller.serviceAccount.clusterWideSecretReadingEnabled` option
to `true` at installation will grant Kargo controllers cluster-wide read
permission on `Secret` resources.

__This is highly discouraged, especially in sharded environments where this
permission would have the undesirable effect of granting remote Kargo
controllers read permissions on all `Secret`s throughout the Kargo control
plane's cluster -- including `Secret`s having nothing to do with Kargo.__
:::

## Other Forms of Credentials

This section provides guidance on configuring Kargo and various cloud platforms
to support "ambient" credentials. Kargo users are presumed not to have
sufficient access to those platform to configure these options themselves, so
this section is intended for operators and cloud platform administrators.

### Amazon Elastic Container Registry (ECR)

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
the controller's IAM role as the value of the
`controller.serviceAccount.iamRole` setting in Kargo's Helm chart at
installation.
:::

At this point, an IAM role will be associated with the Kargo _controller_,
however, that controller acts on behalf of multiple Kargo projects, each of
which may require access to _different_ ECR repositories. To account for this,
when Kargo attempts to access an ECR repository on behalf of a specific project,
it will first attempt to
[assume an IAM role](https://docs.aws.amazon.com/STS/latest/APIReference/API_AssumeRole.html)
specific to that project. The name of the role it attempts to assume will
_always_ be of the form `kargo-project-<project name>`. It is this role that
should be granted read-only access to applicable ECR repositories.

:::info
The name of the IAM role associated with each Kargo project is deliberately
not configurable to prevent project admins from attempting to coerce Kargo into
assuming arbitrary IAM roles.
:::

:::caution
For optimal adherence to the principle of least permissions, the IAM role
associated with the `kargo-controller` `ServiceAccount` should be limited only
to the ability to assume project-specific IAM roles. Project-specific IAM roles
should be limited only to read-only access to applicable ECR repositories.
:::

:::info
If the Kargo controller is unable to assume a project-specific IAM role, it will
fall back to using its own IAM role directly. For organizations without strict
tenancy requirements, this can eliminate the need to manage a large number of
project-specific IAM roles. While useful, this approach is not strictly
recommended.
:::

Once Kargo is able to gain necessary permissions to access an ECR repository,
it will follow a process similar to that described in the previous section to
obtain a token that is valid for 12 hours and cached for 10.

### Google Artifact Registry

Kargo can be configured to authenticate to
[Google Artifact Registry](https://cloud.google.com/artifact-registry/docs/overview)
(GAR) repositories using
[Workload Identity Federation](https://cloud.google.com/kubernetes-engine/docs/how-to/workload-identity))
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

#### Kubernetes Service Account Principal

With WIF enabled,
[GCP Identity and Access Management](https://cloud.google.com/iam/docs/overview)
(IAM) automatically understands a
[principal](https://cloud.google.com/iam/docs/overview#principals)
identifier of the following form to be a reference to the Kargo controller's
_Kubernetes_ Service Account (KSA):

```
principal://iam.googleapis.com/projects/<gcp project number>/locations/global/workloadIdentityPools/<gcp project name>.svc.id.goog/subject/ns/<kargo namespace>/sa/kargo-controller
```

:::note
You do not need to annotate the Kargo controller's KSA in any specific way. At one time this may have been required. Later it may have been required for standard clusters but not for Autopilot clusters. At present, this is not required at all.
:::

#### Google Service Account Impersonation

Although associated with [this _one_ principal](#kubernetes-service-account-principal), the Kargo controller acts on
behalf of multiple Kargo projects, each of which may require access to
_different_ GAR repositories. To account for this, when
Kargo attempts to access a GAR repository on behalf of a
specific project, it will first attempt to
[impersonate a GSA](https://cloud.google.com/iam/docs/service-account-impersonation) 
specific to that project. 

The name of the GSA it attempts to
impersonate will _always_ be of the form
`kargo-project-<KARGO_PROJECT_NAME>@<GCP_PROJECT_NAME>.iam.gserviceaccount.com`.

:::info
The name of the GSA associated with each Kargo project is deliberately
not configurable to prevent project admins from attempting to coerce Kargo into
assuming arbitrary GSA's.
:::

Each [GSA(Google Service Account)](https://cloud.google.com/iam/docs/service-account-overview):

- Should be granted read-only access to applicable
  GAR repositories.

- Should have an [IAM policy](https://cloud.google.com/iam/docs/reference/rest/v1/Policy) that allows the Kargo controller's KSA to create a token. i.e. `roles/iam.serviceAccountTokenCreator`

:::note
This permits the Kargo controller's KSA to impersonate (Kargo) Project-specific GSAs while observing the practice of principle-of-least-priviledge.
:::


:::info
The name of the GSA associated with each Kargo project is
deliberately not configurable to prevent Kargo project admins from attempting to
coerce Kargo into impersonating arbitrary GSA's.
:::

Once Kargo is able to impersonate the appropriate GSA for a
given project, it will follow a process similar to that described in the
previous section to obtain a token that is valid for 60 minutes and cached for
40.

:::note
Beginning with Kargo `v1.5.0`, if maintaining a separate GSA for every Kargo Project is deemed too much of a hassle and strict adherence to the principle of least privilege is not a concern, GAR permissions may be granted directly to the [Kargo controller's KSA](#kubernetes-service-account-principal). In the event that a (Kargo) Project-specific GSA does not exist or cannot be impersonated by the Kargo controller's KSA, Kargo will fall back on using the controller's KSA to access GAR directly.
:::

:::info
Like in the case of EKS Pod Identity or IRSA if project-specific credentials could
not be assumed by the Kargo controller will fall back on on using its own 
IAM principal directly. For organizations without strict tenancy requirements, this 
can eliminate the need to manage a large number of project-specific IAM roles. 
While useful, this approach is not strictly recommended.
:::

### Azure Container Registry (ACR)

Support for authentication to ACR repositories using workload identity is not
yet implemented. Assuming/impersonating a project-specific principal in Azure is
notably complex. So, while a future release is very likely to implement some
form of support for ACR and workload identity, it is unlikely to match the
capabilities Kargo provides for ECR or GAR.
