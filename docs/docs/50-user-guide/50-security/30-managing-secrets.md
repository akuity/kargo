---
description: Learn to manage secrets used by your projects
sidebar_label: Managing Secrets
---

# Managing Secrets

Kargo Projects use Kubernetes `Secret` resources to store repository credentials
and other types of sensitive data, such as API keys for third-party services.

It is crucial that users managing Kargo Projects understand how `Secret`s are
organized and accessed.

:::info[Not what you were looking for?]

If you're an operator looking to understand your role in managing repository
credentials and other secrets, you may find a some value in this document, but
should refer primarily to the
[Managing Secrets](../../40-operator-guide/40-security/40-managing-secrets.md)
section of the Operator's Guide.

:::

## Overview

Users managing Kargo Projects will find themselves concerned with secrets
falling into one of two broad categories:

* **Repository credentials:** Secrets specifically representing credentials for
  the three types of repositories supported by Kargo: Git repositories,
  container image repositories, and Helm chart repositories.

* **"Generic credentials":** Any secrets that are not specifically repository
  credentials.

:::info

The misnomer "generic _credentials_" is used for historical reasons, but nothing
limits these to storing _only_ credentials. In actuality, they can be any sort
of sensitive information. So although they are called "generic credentials,"
they are best thought of in more general terms as, simply, generic secrets.

:::

## Repository Credentials

Kargo expects `Secret` resources representing repository credentials to be
labeled in specific ways and to conform to a specific format. Such `Secret`s
generally take the following form:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: <name>
  namespace: <project namespace>
  labels:
    kargo.akuity.io/cred-type: <type>
data: # base64 encoded
  repoURL: <repo url>
  username: <username>
  password: <password>
```

The label key `kargo.akuity.io/cred-type`, together with its value, specify the
_type_ of the repository accessed with the credential:

* `git`: Credentials for Git repositories
* `helm`: Credentials for Helm chart repositories
* `image`: Credentials for container image repositories

`Secret`s representing repository credentials MUST include the key `repoURL` in
their `data` block. Its value may be either a full, exact URL **OR** a regular
expression matching the URLs of multiple repositories for which the credentials
are valid, in which case, the `data` block must also contain the key/value pair
`repoURLIsRegex: "true"`.

The remaining key/value pairs in such a `Secret`'s `data` block are dependent
upon exactly what kind of credential the `Secret` represents. Commonly, they may
be:

* `username`: The username to use when authenticating to the repository

* `password`: A password or **personal access token**

  :::info

  If the value of the `password` key is a **personal access token**, the value
  of the `username` field is often inconsequential. You should consult your
  repository's documentation for more information.

  :::

Alternatively, for Git repositories only (and specifically ones that support
SSH-style URLs of the form `git@github.com:example/repo.git`), the key
`sshPrivateKey` in the `Secret`'s `data` block may have as its value a
PEM-encoded SSH private key.

:::warning[Not Recommended]

While an SSH private key is an adequate credential for basic Git operations that
are formally part of the Git specification (i.e. `clone`, `checkout`, etc.), the
proprietary APIs offered by the major git hosting platforms (e.g. GitHub or
GitLab) to enable actions such as opening or closing pull requests are
invariably HTTP-based and therefore cannot use an SSH private key for
authentication.

If your Project will create or merge pull request, which is common, rather than
using an SSH private key for basic Git operations and a second credential, such
as a personal access token, for API calls, the Kargo team recommends using a
single credential that works for both -- and an SSH private key is not such a
credential.

:::

:::info[Credential Shapes]

`Secret` resources representing repository credentials come in a wide variety of
other "shapes" (different keys in the `data` block) for use with various
authentication mechanisms. The details of each are covered in their own,
dedicated sections toward the end of this document, following a more general
treatment of the secrets topic.

:::

### Using Repository Credentials

A unique property of `Secret` resources representing repository credentials is
is that there is no need to reference them directly. Any time Kargo accesses a
repository, it _automatically_ attempts to locate suitable credentials,
searching by _repository type and URL._

Project-level repository credentials _can_ be referenced directly from within an
expression by utilizing the
[`secret()`](../../50-user-guide/60-reference-docs/40-expressions.md#secretname)
expression function.

::info

This is in contrast to _shared_ repository credentials from the shared resources
namespace, which by design, _cannot_ be referenced directly. This limitation
permits operators managing a Kargo instance to place repository credentials in
the **shared resources namespace**, knowing that they can be used by all
Projects _without their values ever being exposed to users._

:::

When Kargo needs repository credentials, it searches for `Secret`s in _two_
specific namespaces, in the following order:

1. **Project namespace**: Kargo searches the Project's own namespace first.

2. **Shared resources namespace**: If no match is found in the Project's own
   namespace, Kargo searches the **shared resources namespace.**

:::info[Credential Matching Precedence]

_Within_ each namespace searched, Kargo considers credentials in this order:

1. Exact `repoURL` matches (where `repoURLIsRegex` is `"false"` or unspecified)
2. Pattern matches using regex (where `repoURLIsRegex` is `"true"`)

Within each category, `Secret`s are considered in lexical order by name.

The credentials used by Kargo will be the _first_ to match the repository type
and URL.

:::

## Generic Credentials

"Generic credentials" (a misnomer) are any secrets that are not specifically
repository credentials.

`Secret` resources representing generic credentials MUST be labeled with
`kargo.akuity.io/cred-type: generic`.

:::info

The misnomer "generic _credentials_" is used for historical reasons, but
nothing limits these to storing _only_ credentials. In actuality, they can
be any sort of sensitive information. So although they are called "generic
credentials," they are best thought of in more general terms as, simply,
generic secrets.

:::

### Using Generic Credentials

Generic credentials can be accessed directly by name and their `data` blocks are
not required to conform to any specific structure. This makes them suitable for
storing any arbitrary secret data a Project may depend upon. Elements of a
Project that support expressions, such as a promotion process, can access such
secrets by utilizing the
[`secret()`](../../50-user-guide/60-reference-docs/40-expressions.md#secretname)
expression function.

:::tip

Generic credentials from the shared resources namespace can be accessed using
the
[`sharedSecret()`](../../50-user-guide/60-reference-docs/40-expressions.md#sharedsecretname)
expression function instead.

:::

## Managing Credentials with the CLI

Unless the operator has disabled it, users with the appropriate permissions can
manage Project-level credentials using either the UI or CLI.

:::caution

While the UI or CLI may be a fine way of managing Project-level credentials
whilst getting to know Kargo, it is unquestionably more secure to use other
means to ensure the existence of these specially-formatted `Secret`s.

For precisely this reason, the operator managing your Kargo installation may
very well have disabled the ability to manage credentials using the UI and CLI.

If this is the case, managing your credentials is likely to involve GitOps'ing
your Kargo Projects and also leveraging additional tools such as
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
[in the first section](#repository-credentials) of this
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
    1. Take note of the <Hlt>Client ID</Hlt>.
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
      githubAppClientID: <client id>
      githubAppPrivateKey: <PEM-encoded private key>
      githubAppInstallationID: <installation id>
      repoURL: <repo url>
      repoURLIsRegex: <true if repoURL is a pattern matching multiple repositories>
    ```

    :::info

    GitHub currently recommends using an App's alphanumeric client ID whenever
    possible, but Kargo does support the deprecated, numeric App ID as an
    alternative identifier for a GitHub App. The following example is thus
    valid until such time that GitHub themselves may remove support for the
    deprecated App ID.

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

    In the event that a `Secret`'s data map includes values for _both_ the
    `githubAppClientID` and `githubAppID` keys, Kargo will prioritize the value
    of the `githubAppClientID` key as its means of uniquely identifying the
    GitHub App.
    :::

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
GitHub Apps.

For convenience's sake, it may be tempting to register a single GitHub App,
select a broad set of repositories when installing that App, then create a
single set of
[shared credentials](../../40-operator-guide/40-security/40-managing-secrets.md#repository-credentials),
_however_, this will have the undesirable effect of granting _all_ Kargo
Projects access to _all_ of the selected repositories.

Alternatively, you might consider registering a _separate_ GitHub App for each
Kargo Project, selecting a narrower set of repositories when installing each
App, then creating corresponding Secrets in individual Project namespaces.
While this better adheres to the principle of least privilege, it can be
onerous to manage. Worse, because GitHub organizations are limited to
registering 100 GitHub Apps each, the approach does not scale beyond 100
Projects.

Beginning with Kargo v1.8.0, a third, experimental (stability not guaranteed)
approach builds upon the first, by adding an optional annotation to the
[shared credentials](../../40-operator-guide/40-security/40-managing-secrets.md#repository-credentials) `Secret` containing a map that
constrains the scopes (repositories) available to each Project.

In the following example, the credentials defined by the `github` `Secret` in
the shared credentials namespace are available to all Kargo Projects, however,
the `kargo-demo-1` Project is able to obtain access tokens scoped to either
`repo-a` or `repo-b` only, while the `kargo-demo-2` Project is able to obtain
access tokens scoped to `repo-c` only. No other Project is able to obtain access
tokens scoped to _any_ repository.

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: github
  namespace: kargo-shared-credentials
  labels:
    kargo.akuity.io/cred-type: git
    annotations:
      kargo.akuity.io/github-token-scopes: |
        {
          "kargo-demo-1": ["repo-a", "repo-b"],
          "kargo-demo-2": ["repo-c"]
        }
data:
  # ...
```

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
  awsRegion: us-west-2
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
[Managing Secrets](../../40-operator-guide/40-security/40-managing-secrets.md)
section of the Operator Guide.

:::

### Google Artifact Registry

The authentication options described in this section are applicable to both
container image repositories and OCI Helm chart repositories whose URLs indicate
they are hosted in Google Artifact Registry.

#### Long-Lived Credentials {#gar-long-lived-credentials}

:::caution

Google Artifact Registry does _directly_ support long-lived credentials
[as described here](https://cloud.google.com/artifact-registry/docs/docker/authentication#json-key).
The username `_json_key_base64` and the base64-encoded service account key
may be stored in the `username` and `password` fields of a `Secret` resource as
described [in the first section](#repository-credentials) of
this document.

**Google strongly discourages this method of authentication however, and so do
we.**

:::

Google documentation recommends
[using a service account key to obtain an access token](https://cloud.google.com/artifact-registry/docs/docker/authentication#token)
that is valid for 60 minutes. Compared to the discouraged method of using the
service account key to authenticate to the registry directly, this process does
_not_ transmit the service account key over the wire. Kargo can seamlessly carry
out this process and will cache the access token for a period of 40 minutes.

To use this option, your `Secret` should take the following form:

* For a container image repository:

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
    repoURL: us-central1-docker.pkg.dev/my-project/my-images
  ```

* For an OCI Helm chart repository:

  ```yaml
  apiVersion: v1
  kind: Secret
  metadata:
    name: <name>
    namespace: <project namespace>
    labels:
      kargo.akuity.io/cred-type: helm
  stringData:
    gcpServiceAccountKey: <base64-encoded service account key>
    repoURL: oci://us-central1-docker.pkg.dev/my-project/my-helm-charts/my-chart
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
to authenticate. This works for both container image repositories and OCI Helm
chart repositories, and relies upon some external setup. Leveraging this
option eliminates the need to store credentials in a `Secret` resource.

:::info

This option relies upon extensive external configuration that likely requires
the assistance of Kargo's operator and a GCP project administrator, and as such,
further coverage is delegated to the
[Managing Secrets](../../40-operator-guide/40-security/40-managing-secrets.md)
section of the Operator Guide.

:::

### Azure Container Registry (ACR)

The authentication options described in this section are applicable only to
container image repositories whose URLs indicate they are hosted in ACR.

#### Long-Lived Credentials {#acr-long-lived-credentials}

Azure Container Registry directly supports long-lived credentials.

It is possible to
[create tokens with repository-scoped permissions](https://learn.microsoft.com/en-us/azure/container-registry/container-registry-repository-scoped-permissions),
with or without an expiration date. These tokens can be stored in the
`username` and `password` fields of a `Secret` resource as described
[in the first section](#repository-credentials) of this
document.

:::caution

Following the principle of least privilege, the ACR token should be limited only
to read-only access to the required ACR repositories. Configuring this will
likely require the assistance of an Azure administrator.

:::

:::caution

This method of authentication is a "lowest common denominator" approach that
will work regardless of where Kargo is deployed. i.e. If running Kargo outside
of AKS, this method will still work.

If running Kargo within AKS, you may wish to consider using Azure Workload
Identity instead.

:::

#### Azure Workload Identity

If Kargo locates no `Secret` resources matching a repository URL, and if Kargo
is deployed within an AKS cluster with workload identity enabled, it will attempt
to use [Azure Workload Identity](https://learn.microsoft.com/en-us/azure/aks/workload-identity-overview)
to authenticate. Leveraging this option eliminates the need to store credentials
in a `Secret` resource.

:::info

This option relies upon extensive external configuration that likely requires
the assistance of Kargo's operator and an Azure administrator, and as such,
further coverage is delegated to the
[Managing Secrets](../../40-operator-guide/40-security/40-managing-secrets.md)
section of the Operator Guide.

:::
