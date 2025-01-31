---
description: Learn to manage repository credentials used by your projects
sidebar_label: Managing Credentials
---

# Managing Credentials

To orchestrate the promotion of `Freight` from `Stage` to `Stage`, Kargo will
often require read/write permissions on private Git repositories and read-only
permissions on private container image or Helm chart repositories.

This section presents an overview of how users can manage and use such
credentials within their Kargo projects.

:::info
__Not what you were looking for?__

If you're an operator looking to understand your role in managing
credentials, you may find a some value in this document, but should
refer also to the
[Managing Credentials](../../40-operator-guide/40-security/40-managing-credentials.md)
section of the Operator's Guide.
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

:::info
`Secret`s defined within a project's namespace are accessible only to
`Warehouse`s and `Promotion`s within that project.
:::

The names of `Secret` resources are inconsequential because Kargo matches
credentials to repositories by repository type and URL. `Secret` names may
therefore observe any naming convention preferred by the user.

The label key `kargo.akuity.io/cred-type` and its value, one of `git`, `helm`,
`image`, or `generic` are important, as they designates a `Secret` as
representing credentials for a Git, Helm chart, or container image repository,
or _something else_, respectively.

:::info
Despite the appearance of "cred-type" in the label key, `Secret`s labeled as
`generic` do not actually need to represent credentials. They could contain
_any_ kind of sensitive information used in your promotion processes. Managing
such `Secret`s is covered separately in
[Managing Other Secrets](../50-security/40-managing-other-secrets.md).
:::

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

  OR:

  * `sshPrivateKey`: A PEM-encoded SSH private key. Applicable only to Git
    repositories using SSH-style URLs -- for instance
    `git@github.com:example/repo.git`.

:::info
Exceptions to the formatting discussed above are covered in later sections.
:::

:::note
__Precedence__

When Kargo searches for repository credentials in a project's namespace, it
_first_ iterates over all appropriately labeled `Secret`s _without_
`repoIsRegex` set to `true` looking for a `repoURL` value matching the
repository URL exactly.

Only if no exact match is found does found does it iterate over all
appropriately labeled `Secret`s with `repoIsRegex` set to `true` looking for a
regular expression matching the repository URL.

When searching for an exact match, and then again when searching for a pattern
match, appropriately labeled `Secret`s are considered in lexical order by name.
:::

## Global Credentials

In cases where one or more sets of credentials are needed widely across many or
all Kargo projects, an operator may opt into designating one or more namespaces
as containing "global" credentials, accessible to all projects. If you are an
operator looking for more information on this topic, please refer to the
[Managing Credentials](../../40-operator-guide/40-security/40-managing-credentials.md)
section of the Operator Guide.

When Kargo searches for repository credentials, these additional namespaces are
searched only _after_ finding no matching credentials in the project's own
namespace.

## Managing Credentials with the CLI

Unless the operator has disabled it, users with the appropriate permissions can
manage project-level credentials using either the UI or CLI.

:::info
UI-based instructions coming soon.
:::

:::caution
While the UI or CLI may be a fine way of managing project-level credentials
whilst getting to know Kargo, it is unquestionably more secure to use other
means to ensure the existence of these specially-formatted `Secret`s.

For precisely this reason, the operator managing your Kargo installation may
very well have disabled the ability to manage credentials using the UI and CLI.

If this is the case, managing your credentials is likely to involve GitOps'ing
your Kargo projects and also leveraging additional tools such as
[Sealed Secrets](https://github.com/bitnami-labs/sealed-secrets) or the
[External Secrets Operator](https://external-secrets.io/latest/).
:::

### Creating Credentials

The following example creates credentials for a specific Git repository:

```shell
kargo create credentials \
  --project kargo-demo my-credentials \
  --git \
  --repo-url https://github.com/example/kargo-demo.git \
  --username my-username \
  --password my-personal-access-token
```

```shell
secret/my-credentials created
```

:::caution
If you do not wish for your password or personal access token to be stored
in your shell history, you may wish to omit the `--password` flag, in which
case the CLI will prompt you to enter the password interactively.
:::

### Listing / Viewing Credentials

Credentials can be listed or viewed with `kargo get credentials`:

```shell
kargo get credentials --project kargo-demo my-credentials
```

```shell
NAME             TYPE   REGEX   REPO                                        AGE
my-credentials   git    false   https://github.com/example/kargo-demo.git   8m25s
```

If requesting output as YAML or JSON, the values of all data fields and
annotations not explicitly deemed "safe" are redacted. In the example below, you
can see the values of `repoURL` and `username` have not been redacted because
those fields are assumed not to contain sensitive information. `password` is
redacted, however. Values of arbitrarily named data fields are also redacted
because Kargo cannot infer their sensitivity.

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
  foo: '*** REDACTED ***'
type: Opaque
```

### Updating Credentials

Credentials can be updated using the `kargo update credentials` command and
the flags corresponding to attributes of the credentials that you wish to
modify. Other attributes of the credentials will remain unchanged.

The following example updates `my-credentials` with a regular expression for the
repository URL:

```shell
kargo update credentials \
  --project kargo-demo my-credentials \
  --repo-url '^https://github.com/' \
  --regex
```

```shell
secret/my-credentials updated
```

### Deleting Credentials

Credentials can, of course, be deleted with `kargo delete credentials`:

```shell
kargo delete credentials --project kargo-demo my-credentials
```

```shell
secret/my-credentials deleted
```

## Other Forms of Credentials

This section provides guidance on managing credentials for GitHub and for
several popular container image registries. These options range from long-lived
tokens to "ambient" credentials that can be obtained automatically when running
within certain cloud platforms.

:::note
In many cases, applying the options discussed in the following sections may
require the assistance of an operator/administrator for the applicable
platforms.
:::

### GitHub Authentication Options

The following two sections cover GitHub-specific authentication options that
are more secure than simply using a username and password.

#### Personal Access Token

GitHub supports authentication using a
[personal access token](https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/managing-your-personal-access-tokens),
which can be used in place of a password. The corresponding username must be
the GitHub handle of the user who created the token. These can be stored in
the `username` and `password` fields of a `Secret` resource as described
[in the first section](#repository-credentials-as-secret-resources) of this
document.

:::info
This method of authentication may be best when wishing to rigorously enforce
the principle of least privilege, as personal access tokens can be scoped to
specific permissions on specific repositories.

A drawback to this method, however, is that the token is owned by a specific
GitHub user, and if that user should lose their own access to the repositories
in question, Kargo will also lose access.
:::

#### GitHub App Authentication

[GitHub Apps](https://docs.github.com/en/apps) can be used
[as an authentication method](https://docs.github.com/en/apps/creating-github-apps/authenticating-with-a-github-app/about-authentication-with-a-github-app).

:::note
You may require the assistance of your GitHub organization's administrator to
create or install a GitHub App.
:::

1. [Create a GitHub App](https://github.com/settings/apps/new):

    1. In the <Hlt>GitHub App name</Hlt> field, specify a unique
       name.
    1. Set the <Hlt>Homepage URL</Hlt> to any URL you like.
    1. Under <Hlt>Webhook</Hlt>, de-select
       <Hlt>Active</Hlt>.
    1. Under <Hlt>Permissions</Hlt> → <Hlt>Repository
       permissions</Hlt> → <Hlt>Contents</Hlt>, select whether
       the App will require <Hlt>Read-only</Hlt> or <Hlt>Read
       and write</Hlt> permissions. _The App will receive these
       permissions on all repositories into which it is installed._
    1. Under <Hlt>Where can this GitHub App be installed?</Hlt>,
       leave <Hlt>Only on this account</Hlt> selected.
    1. Click <Hlt>Create GitHub App</Hlt>.
    1. Take note of the <Hlt>App ID</Hlt>.
    1. Scroll to the bottom of the page and click <Hlt>Generate a private
       key</Hlt>. The resulting key will be downloaded immediately. Store
       it securely.
    1. On the left-hand side of the page, click <Hlt>Install
       App</Hlt>.
    1. Choose an account to install the App into by clicking
       <Hlt>Install</Hlt>.
    1. Select <Hlt>Only select repositories</Hlt> and choose the
       repositories you wish to grant the App access to. Remember that the App
        will receive the permissions you selected earlier on _all_ of these
        repositories.
    1. Click <Hlt>Install</Hlt>.
    1. In your browser's address bar, take note of the numeric identifier at the
       end of the current page's URL. This is the <Hlt>Installation
       ID</Hlt>.

2. Create a `Secret` resource with the following structure:

    ```yaml
    apiVersion: v1
    kind: Secret
    metadata:
      name: <name>
      namespace: <project namespace>
      labels:
        kargo.akuity.io/cred-type: git
    stringData:
      githubAppID: <app id>
      githubAppPrivateKey: <PEM-encoded private key>
      githubAppInstallationID: <installation id>
      repoURL: <repo url>
      repoURLIsRegex: <true if repoURL is a pattern matching multiple repositories>
    ```

    :::note
    The `kargo create/update credentials` commands do not support creating or
    updating non username/password credentials. To create or update a `Secret`
    such as the one shown above, use GitOps instead, or the
    `kargo apply --project <project> -f <filename>` command.
    :::

:::info
Compared to personal access tokens, a benefit of authenticating with a GitHub
App is that the App's permissions are not tied to a specific GitHub user.
:::

:::caution
It is easy to violate the principle of least privilege when authenticating using
this method.

For convenience sake, it may be tempting to register a single GitHub App and
select a broad set of repositories when installing that App into your
organization. It may also be tempting to create a single set of
[global credentials](#global-credentials) such that all Kargo projects can use
them to access their repositories, _however_, this will have the undesirable
effect of granting _all_ Kargo projects access to _all_ of the repositories
selected when the App was installed.

It is, instead, recommended to register a separate GitHub App for
each Kargo project. When installing each App into your organization, only those
repositories to which each Kargo project requires access should be selected.

GitHub organizations are limited to registering 100 GitHub Apps, however, so
this approach may not be feasible for organizations with many Kargo projects.
:::

:::caution
A second way in which authentication using GitHub Apps may violate the principle
of least privilege involves the fact that the same permissions are granted to
the App on _all_ repositories that are selected when it is installed.

If a Kargo project requires read-only access to one repository and read/write
access to another, it is not possible to grant the App different permissions on
the two. This may then lead to granting broader permissions than are strictly
necessary.
:::

### Amazon Elastic Container Registry (ECR)

The authentication options described in this section are applicable only to
container image repositories whose URLs indicate they are hosted in ECR.

#### Long-Lived Credentials {#ecr-long-lived-credentials}

Elastic Container Registries do not _directly_ support long-lived credentials,
however, an AWS access key ID and secret access key
[can be used to obtain an authorization token](https://docs.aws.amazon.com/AmazonECR/latest/userguide/registry_auth.html#registry-auth-token)
that is valid for 12 hours. Kargo can seamlessly obtain such a token and will
cache it for a period of 10 hours.

To use this option, your `Secret` should take the following form:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: <name>
  namespace: <project namespace>
  labels:
    kargo.akuity.io/cred-type: image
stringData:
  awsAccessKeyID: <access key id>
  awsSecretAccessKey: <secret access key>
  repoURL: <ecr url>
```

:::note
The `kargo create/update credentials` commands do not support creating or
updating non username/password credentials. To create or update a `Secret` such
as the one shown above, use GitOps instead, or the
`kargo apply --project <project> -f <filename>` command.
:::

:::caution
Following the principle of least privilege, the IAM user associated with the
access key ID and secret access key should be limited only to read-only access
to the required ECR repositories. Configuring this will likely require the
assistance of an AWS account administrator.
:::

:::caution
This method of authentication is a "lowest common denominator" approach that
will work regardless of where Kargo is deployed. i.e. if running Kargo outside
EKS, this method will still work.

If running Kargo within EKS, you may wish to either consider using EKS Pod
Identity or IRSA instead.
:::

#### EKS Pod Identity or IAM Roles for Service Accounts (IRSA)

If Kargo locates no `Secret` resources matching a repository URL and is deployed
within an EKS cluster, it will attempt to use
[EKS Pod Identity](https://docs.aws.amazon.com/eks/latest/userguide/pod-identities.html)
or
[IAM Roles for Service Accounts (IRSA)](https://docs.aws.amazon.com/eks/latest/userguide/iam-roles-for-service-accounts.html)
to authenticate. Leveraging either eliminates the need to store ECR credentials
in a `Secret` resource.

:::info
Both of these options rely upon extensive external configuration that likely
requires the assistance of Kargo's operator and an AWS account administrator,
and as such, further details are covered in the
[Managing Credentials](../../40-operator-guide/40-security/40-managing-credentials.md)
section of the Operator Guide.
:::

### Google Artifact Registry

The authentication options described in this section are applicable only to
container image repositories whose URLs indicate they are hosted in Google
Artifact Registry.

#### Long-Lived Credentials {#gar-long-lived-credentials}

:::caution
Google Artifact Registry does _directly_ support long-lived credentials
[as described here](https://cloud.google.com/artifact-registry/docs/docker/authentication#json-key).
The username `_json_key_base64` and the base64-encoded service account key
may be stored in the `username` and `password` fields of a `Secret` resource as
described [in the first section](#repository-credentials-as-secret-resources) of
this document.

__Google strongly discourages this method of authentication however, and so do
we.__
:::

Google documentation recommends
[using a service account key to obtain an access token](https://cloud.google.com/artifact-registry/docs/docker/authentication#token)
that is valid for 60 minutes. Compared to the discouraged method of using the
service account key to authenticate to the registry directly, this process does
_not_ transmit the service account key over the wire. Kargo can seamlessly carry
out this process and will cache the access token for a period of 40 minutes.

To use this option, your `Secret` should take the following form:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: <name>
  namespace: <project namespace>
  labels:
    kargo.akuity.io/cred-type: image
stringData:
  gcpServiceAccountKey: <base64-encoded service account key>
  repoURL: <artifact registry url>
```

:::note
Service account keys contain structured data, so it is important that the
key be base64-encoded.
:::

:::caution
Following the principle of least privilege, the service account associated with
the service account key should be limited only to read-only access to the
required Google Artifact Registry repositories. Configuring this will likely
require the assistance of a GCP project administrator.
:::

:::caution
This method of authentication is a "lowest common denominator" approach that
will work regardless of where Kargo is deployed. i.e. If running Kargo outside
of GKE, this method will still work.

If running Kargo within GKE, you may wish to consider using Workload Identity
Federation instead.
:::

#### Workload Identity Federation

If Kargo locates no `Secret` resources matching a repository URL, and if Kargo
is deployed within a GKE cluster, it will attempt to use
[Workload Identity Federation](https://cloud.google.com/kubernetes-engine/docs/how-to/workload-identity)
to authenticate, but this relies upon some external setup. Leveraging this
option eliminates the need to store credentials in a `Secret` resource.

:::info
This option relies upon extensive external configuration that likely requires
the assistance of Kargo's operator and a GCP project administrator, and as such,
further coverage is delegated to the
[Managing Credentials](../../40-operator-guide/40-security/40-managing-credentials.md)
section of the Operator Guide.
:::

### Azure Container Registry (ACR)

Azure Container Registry directly supports long-lived credentials.

It is possible to
[create tokens with repository-scoped permissions](https://learn.microsoft.com/en-us/azure/container-registry/container-registry-repository-scoped-permissions),
with or without an expiration date. These tokens can be stored in the
`username` and `password` fields of a `Secret` resource as described
[in the first section](#repository-credentials-as-secret-resources) of this
document.

:::info
Support for authentication to ACR repositories using workload identity is not
yet implemented. Assuming/impersonating a project-specific principal in Azure is
notably complex. So, while a future Kargo release is very likely to add some
form of support for ACR and workload identity, it is unlikely to match the
capabilities Kargo provides for ECR or GAR.
:::
