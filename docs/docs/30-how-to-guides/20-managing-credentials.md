---
description: Find out how to manage repository credentials for use by Kargo
sidebar_label: Managing credentials
---

# Managing Credentials

To manage the progression of freight from stage to stage, Kargo will often
require read/write permissions on private GitOps repositories and read-only
permissions on private container image and/or Helm chart repositories.

This section presents an overview of how these credentials can be managed.

## Credentials as Kubernetes `Secret` Resources

:::caution
Kargo formerly borrowed its general credential-management approach from Argo CD,
but has since diverged.
:::

Kargo expects any credentials it requires to have been stored as specially
labeled Kubernetes `Secret` resources containing specially-formatted data. These
`Secret`s take the following form:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: <name>
  namespace: <project namespace>
  labels:
    kargo.akuity.io/cred-type: <cred type>
stringData:
  repoURL: <repo url>
  username: <username>
  password: <password>
```

The `name` of such a `Secret` is inconsequential and may follow any convention
preferred by the user.

:::info
Kargo uses Kubernetes `Namespace`s to mark project boundaries. `Secret`s
representing credentials will typically exist in the same `Namespace` as the
`Stage` resources that will require them. There are exceptions to this, which
are covered in the next section.
:::

The label key `kargo.akuity.io/cred-type` and its value, one of `git`, `helm`,
or `image`, is important, as it designates the `Secret` as representing
credentials for a Git repository, a Helm chart repository, or a container image
repository, respectively.

The `Secret`'s `data` field (set above using plaintext in the `stringData`
field), MUST contain the following keys:

* `repoURL`: The full URL of the repository the credentials are for.

* `username`: The username to use when authenticating to the repository.

* `password`: A password or personal access token.

    :::info
    If the value of the `password` key is a personal access token, the value of
    the `username` field may be inconsequential. You should consult your
    repository's documentation for more information.
    :::

Optionally, the following keys may also be included:

* `repoURLIsRegex`: Set this to `true` if the value of the `repoURL` key
  is a regular expression. Any other value of this key or the absence of this
  key is interpreted as `false`.

:::note
When Kargo searches for repository credentials in a project `Namespace`, it
_first_ checks all appropriately labeled `Secret`s for a `repoURL` value
matching the repository URL exactly. Only if no `Secret` is an exact match does
it check all appropriately labeled `Secret`s for a `repoURL` value containing a
regular expression matching the repository URL.

When searching for an exact match, and again when searching for a pattern match,
appropriately labeled `Secret`s are considered in lexical order by name.
:::

:::caution
Only username/password (or personal access token) authentication is
supported at this time. Others are likely to be added in the future.
:::

## Global Credentials

In cases where one or more sets of credentials are needed widely across _all_
Kargo projects, the administrator/operator installing Kargo may opt-in to
designating one or more `Namespace`s as homes for "global" credentials using the
`controller.globalCredentials.namespaces` setting in Kargo's Helm chart.
Refer to
[the advanced section of the installation guide](./10-installing-kargo.md#advanced-installation)
for more details.

:::note
Any matching credentials (exact match _or_ pattern match) found in a project's
own `Namespace` take precedence over those found in any global credentials
`Namespace`.

When Kargo searches for repository credentials in global credentials
`Namespace`s, it _first_ checks all appropriately labeled `Secret`s for a
`repoURL` value matching the repository URL exactly. Only if no `Secret` is an
exact match does it check all appropriately labeled `Secret`s for a
`repoURL` value containing a regular expression matching the repository URL.

When searching for an exact match, and again when searching for a pattern match,
appropriately labeled `Secret`s are considered in lexical order by name.

When Kargo is configured with multiple global credentials `Namespace`s, they are
searched in lexical order by name. Only after no exact match _and_ no pattern
match is found in one global credentials `Namespace` does Kargo search the next.
:::

:::caution
It is important to understand the security implications of this feature. Any
credentials stored in a global credentials `Namespace` will be available to
_all_ Kargo projects.
:::

## Managing Credentials with the CLI

The Kargo CLI can be used to manage credentials in a project's `Namespace.`

The following example creates credentials for a Git repository:

```shell
kargo create credentials --project kargo-demo my-credentials \
  --git --repo-url https://github.com/example/kargo-demo.git \
  --username my-username --password my-my-personal-access-token
```

```shell
secret/my-credentials created
```

:::caution
If you do not wish for your password or personal access token to be stored
in your shell history, you may wish to omit the `--password` flag, in which
case the CLI will prompt you to enter the password interactively.
:::

Credentials can be listed or viewed with `kargo get credentials`:

```shell
kargo get credentials --project kargo-demo my-credentials
```

```shell
NAME             TYPE   REGEX   REPO                                        AGE
my-credentials   git    false   https://github.com/example/kargo-demo.git   8m25s
```

If requesting output as YAML or JSON, passwords and other potentially sensitive
information will be redacted.

```shell
kargo get credentials --project kargo-demo my-credentials -o yaml
```

```shell
apiVersion: v1
kind: Secret
metadata:
  creationTimestamp: "2024-05-30T20:02:46Z"
  labels:
    kargo.akuity.io/cred-type: git
  name: my-credentials
  namespace: kargo-demo
  resourceVersion: "17614"
  uid: ca2660e4-867d-4709-b1a7-57fbb93fc6dc
stringData:
  password: '*** REDACTED ***'
  repoURL: https://github.com/example/kargo-demo.git
  username: my-username
type: Opaque
```

Credentials can be updated using the `kargo update credentials` command and
the flags corresponding to attributes of the credential that you wish to modify.
Other attributes of the credentials will remain unchanged.

The following example updates `my-credentials` with a regular expression for the
repository URL:

```shell
kargo update credentials --project kargo-demo my-credentials \
  --repo-url '^http://github.com/' --regex
```

```shell
secret/my-credentials updated
```

And credentials can, of course, be deleted with `kargo delete credentials`:

```shell
kargo delete credentials --project kargo-demo my-credentials
```

```shell
secret/my-credentials deleted
```

:::note
While the CLI may be a fine way of managing project-level credentials whilst
getting to know Kargo, it is unquestionably more secure to use other means to
ensure the existence of these specially-formatted `Secret`s in the appropriate
project `Namespace`s.
:::

## Registry-Specific Authentication Options

While many container image registries support authentication using long-lived
credentials, such as a username and password (or personal access token), some
either require or offer more secure options.

This section provides registry-specific guidance on credential management and
also covers options for gaining image repository access using workload identity
on applicable platforms.

### Amazon Elastic Container Registry (ECR)

#### Long-Lived Credentials

Elastic Container Registries do not _directly_ support long-lived credentials,
however, an AWS access key ID and secret access key
[can be exchanged for a token](https://docs.aws.amazon.com/AmazonECR/latest/userguide/registry_auth.html#registry-auth-token)
that is valid for 12 hours. Kargo can seamlessly make this exchange and will
cache the resulting token for a period of 10 hours.

To use this option, your `Secret` should take the following form:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: <name>
  namespace: <project namespace>
  labels:
    kargo.akuity.io/cred-type: <cred type>
stringData:
  awsAccessKeyID: <access key id>
  awsSecretAccessKey: <secret access key>
  repoURL: <ecr url>
```

:::caution
Following the principle of least privilege, the IAM user associated with the
access key ID and secret access key should be limited only to read-only access
to the required ECR repositories.
:::

:::caution
This method of authentication is a "lowest common denominator" approach that
will work regardless of where Kargo is deployed. i.e. If running Kargo outside
of EKS, this method will still work.

If running Kargo within EKS, you may wish to consider using EKS Pod Identity
instead.
:::

#### EKS Pod Identity

If Kargo is deployed within an EKS cluster, it is possible to use
[EKS Pod Identity](https://docs.aws.amazon.com/eks/latest/userguide/pod-identities.html)
to authenticate to ECR repositories. This option eliminates the need to store
credentials in a `Secret` resource.

First, follow
[this overview](https://docs.aws.amazon.com/eks/latest/userguide/pod-identities.html#pod-id-setup-overview)
to set up EKS Pod Identity in your EKS cluster and assign an IAM role to the
`kargo-controller` `ServiceAccount` within the `Namespace` to which Kargo is (or
will be) installed.

At this point, an IAM role will be associated with the Kargo _controller_,
however, that controller acts on behalf of multiple Kargo projects, each of
which may require access to _different_ ECR repositories. To account for this,
when Kargo attempts to access an ECR repository on behalf of a specific project,
it will first attempt to
[assume an IAM role](https://docs.aws.amazon.com/STS/latest/APIReference/API_AssumeRole.html)
specific to that project. The name of the role it attempts to assume will _always_
be of the form `kargo-project-<project name>`. It is this role that should be
granted read-only access to applicable ECR repositories.

:::info
The name of the IAM role associated with each Kargo project is deliberately
not configurable to prevent project admins from attempting to coerce Kargo into
assuming arbitrary IAM roles.
:::

Once Kargo is able to assume the appropriate IAM role for a given project, it
will follow a process similar to that described in the previous section to
obtain a token that is valid for 12 hours and cached for 10.

:::caution
Following the principle of least privilege, the IAM role associated with the
`kargo-controller` `ServiceAccount` should be limited only to the ability to
assume project-specific IAM roles. Project-specific IAM roles should be limited
only to read-only access to the applicable ECR repositories.
:::

### Google Artifact Registry

:::note
Google Container Registry (GCR) has been deprecated in favor of Google Artifact
Registry. For authentication to repositories with legacy GCR URLs, the same
options outlined here may be applied.
:::

#### Long-Lived Credentials

:::caution
Google Artifact Registry does _directly_ support long-lived credentials
[as described here](https://cloud.google.com/artifact-registry/docs/docker/authentication#json-key).
The username `_json_key_base64` and the base64-encoded service account key
may be stored in the `username` and `password` fields of a `Secret` resource as
described [in the first section](#credentials-as-kubernetes-secret-resources) of
this document. Google's own documentation strongly recommends against this
method however.
:::

Google documentation recommends
[exchanging a service account key for an access token](https://cloud.google.com/artifact-registry/docs/docker/authentication#token)
that is valid for 60 minutes. Kargo can seamlessly make this exchange and will
cache the resulting token for a period of 40 minutes.

To use this option, your `Secret` should take the following form:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: <name>
  namespace: <project namespace>
  labels:
    kargo.akuity.io/cred-type: <cred type>
stringData:
  gcpServiceAccountKey: <base64-encoded service account key>
  repoURL: <ecr url>
```

:::note
Service account keys contain structured data, so it is important that the
key be base64-encoded.
:::

:::caution
Following the principle of least privilege, the service account associated with
the service account key should be limited only to read-only access to the
required Google Artifact Registry repositories.
:::

:::caution
This method of authentication is a "lowest common denominator" approach that
will work regardless of where Kargo is deployed. i.e. If running Kargo outside
of GKE, this method will still work.

If running Kargo within GKE, you may wish to consider using Workload Identity
Federation instead.
:::

#### Workload Identity Federation

If Kargo is deployed within a GKE cluster, it is possible to use
[Workload Identity Federation](https://cloud.google.com/kubernetes-engine/docs/how-to/workload-identity)
to authenticate to Google Artifact Registry repositories. This option eliminates
the need to store credentials in a `Secret` resource.

First, follow
[these directions](https://cloud.google.com/kubernetes-engine/docs/how-to/workload-identity#enable_on_cluster)
to provision a new GKE cluster with Workload Identity Federation enabled or
[these directions](https://cloud.google.com/kubernetes-engine/docs/how-to/workload-identity#enable-existing-cluster)
to enable Workload Identity Federation on an existing GKE cluster.

At this point, the `kargo-controller` `ServiceAccount` within the `Namespace` to
which Kargo is (or will be) installed will be associated with an _IAM principal
identifier_, which takes the following form:

```plaintext
principal://iam.googleapis.com/projects/<gcp project number>/locations/global/workloadIdentityPools/<gcp project name>.svc.id.goog/subject/ns/<kargo namespace>/sa/kargo-controller
```

Although associated with this _one_ principal, the Kargo controller acts on
behalf of multiple Kargo projects, each of which may require access to
_different_ Google Artifact Registry repositories. To account for this, when
Kargo attempts to access a Google Artifact Registry repository on behalf of a
specific project, it will first attempt to
[impersonate a Google service account](https://cloud.google.com/iam/docs/service-account-impersonation) 
specific to that project. The name of the service account it attempts to
impersonate will _always_ be of the form
`kargo-project-<kargo project name>@<gcp project name>.iam.gserviceaccount.com`.
It is this service account that should be granted read-only access to applicable
Google Artifact Registry repositories.

:::info
The name of the Google service account associated with each Kargo project is
deliberately not configurable to prevent Kargo project admins from attempting to
coerce Kargo into impersonating arbitrary Google service accounts.
:::

Once Kargo is able to impersonate the appropriate Google service account for a
given project, it will follow a process similar to that described in the
previous section to obtain a token that is valid for 60 minutes and cached for
40.

:::caution
Following the principle of least privilege, the IAM principal associated with
the `kargo-controller` `ServiceAccount` should be limited only to the ability to
impersonate project-specific Google service accounts. Project-specific Google
service accounts should be limited only to read-only access to the applicable
Google Artifact Registry repositories.
:::

### Azure Container Registry (ACR)

Azure Container Registry directly supports long-lived credentials.

It is possible to
[create tokens with repository-scoped permissions](https://learn.microsoft.com/en-us/azure/container-registry/container-registry-repository-scoped-permissions),
with or without an expiration date. These tokens can be be stored in the
`username` and `password` fields of a `Secret` resource as described
[in the first section](#credentials-as-kubernetes-secret-resources) of this
document.

:::info
Support for authentication to ACR repositories using workload identity, on par
with Kargo's support for ECR and Google Artifact Registry, is likely to be
included in a future release of Kargo.
:::
