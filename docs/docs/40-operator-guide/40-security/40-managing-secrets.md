---
description: Learn to manage credentials and other secrets
sidebar_label: Managing Secrets
---

# Managing Secrets

Kargo uses Kubernetes `Secret` resources to store repository credentials and
other types of sensitive data, such as API keys for third-party services. The
namespaces within which they are contained impact who or what can access them.
Labels on those `Secret`s may also constrain _how_ they are accessed.

It is crucial that operators managing Kargo instances understand how `Secret`s
are organized and accessed.

:::info[Not what you were looking for?]

If you're a Kargo user looking to learn more about managing credentials or any
other kind of secret _within your own Project_, refer instead to the
[Managing Secrets](../../50-user-guide/50-security/30-managing-secrets.md)
section of the User's Guide.

:::

## Overview

Operators managing a Kargo instance will find themselves concerned with secrets
falling into one of two broad categories:

* **Shared secrets** intended for read-only access _by any or all Projects within
  the instance._ These, in turn, can be classified as one of:

  * **Repository credentials:** Secrets specifically representing credentials
    for the three types of repositories supported by Kargo: Git repositories,
    container image repositories, and Helm chart repositories.

  * **"Generic credentials":** Any secrets that are not specifically repository
    credentials.

    :::info

    The misnomer "generic _credentials_" is used for historical reasons, but
    nothing limits these to storing _only_ credentials. In actuality, they can
    be any sort of sensitive information. So although they are called "generic
    credentials," they are best thought of in more general terms as, simply,
    generic secrets.

    :::

* **System-level secrets** used by Kargo itself and _not_ intended to be
  accessed by Kargo Projects.

The remainder of this document will cover each of these in turn, explaining in
detail what such secrets look like, where they are stored, who can access them,
and _how_ they are accessed.

## Shared Secrets

As the name implies, shared secrets are those intended to be accessible by all
Projects within a Kargo instance. Their corresponding `Secret` resources belong
in one, specific Kubernetes namespace referred to as **the shared resources
namespace**.

:::info[Why shared _resources_?]

Prior to Kargo v1.9.0, what is now the shared resources namespace was referred
to as "global credentials namespaces" (plural). Three factors prompted the Kargo
team to refine and rename the concept:

* "Global" was prone to various misinterpretations.
* "Namespaces" (plural) added unnecessary technical complications to the system.
* "Credentials" was too specific. Not all secrets are credentials. And not all
  things to be shared across Projects are secrets. The more general term
  "resources" speaks to a broader purpose for the namespace.

:::

:::warning[Migration]

If you are migrating from a Kargo version lesser than v1.9.0 to version v1.9.0
or greater, please consult the [migration](#migrating-from-kargo--190) section
at the bottom of this page.

:::

### Repository Credentials

Kargo expects `Secret` resources representing repository credentials to be
labeled in specific ways and to conform to a specific format. Such `Secret`s
generally take the following form:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: <name>
  namespace: <project namespace or shared resources namespace>
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

If your Projects will create or merge pull request, which is common, rather than
using an SSH private key for basic operations and a second credential, such as a
personal access token, for API calls, the Kargo team recommends using a single
credential that works for both -- and an SSH private key is not such a
credential.

:::

:::info[Credential Shapes]

`Secret` resources representing repository credentials come in a wide variety
of other "shapes" (different keys in the `data` block) corresponding to various
authentication mechanisms. These are covered in the
[Managing Secrets](../../50-user-guide/50-security/30-managing-secrets.md)
section of the User's Guide.

:::

#### Using Repository Credentials

A unique property of `Secret` resources representing repository credentials is
that Projects do not (and cannot) reference them directly. Any time Kargo
accesses a repository, it _automatically_ attempts to locate suitable
credentials, searching by _repository type and URL._

:::tip

Because of the above, operators managing a Kargo instance can place repository
credentials in the **shared resources namespace**, knowing that they can be used
by all Projects _without their values ever being exposed to users._

:::

When Kargo needs repository credentials, it searches for `Secret`s in _two_
specific namespaces, in the following order:

1. **Project namespace**: Kargo searches the Project's own namespace first.

2. **Shared resources namespace**: If no match is found in the Project's own
   namespace, Kargo searches the shared resources namespace.

:::info[Credential Matching Precedence]

_Within_ each namespace searched, Kargo considers credentials in this order:

1. Exact `repoURL` matches (where `repoURLIsRegex` is `"false"` or unspecified)
2. Pattern matches using regex (where `repoURLIsRegex` is `"true"`)

Within each category, `Secret`s are considered in lexical order by name.

The credentials used by Kargo will be the _first_ to match the repository type and URL.

:::

### Generic Credentials

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

#### Using Generic Credentials

In contrast to repository credentials, `Secret` resources representing shared
generic credentials can be accessed directly by name and their `data` blocks are
not required to conform to any specific structure. This makes them suitable for
storing any arbitrary secret data that Projects may depend upon. Projects can
access such secrets within expressions used by their promotion processes by
utilizing the
[`sharedSecret()`](../../50-user-guide/60-reference-docs/40-expressions.md#sharedsecretname)
expression function.

:::caution

Always remember that any generic credential in the shared resources namespace
can be accessed directly by all Projects, which means it is possible to learn
their values.

Exercise due caution when deciding what secrets are suitable to be shared in
this manner.

:::

### Configuring the Shared Resources Namespace

The **shared resources namespace**, by default, is `kargo-shared-resources`.
Operators may override this at the time of installation or upgrade by overriding
the Kargo Helm chart's `global.sharedResources.namespace` setting.

## System Secrets

Various components of Kargo itself, at times, have the need to reference
operator-defined secrets. The canonical example for this involves configuring
cluster-scoped webhook receivers.

Cluster-scoped webhook receivers are defined as part of a `ClusterConfig`
resource, which is, unsurprisingly, a _cluster-scoped resource_. (i.e. It does
not belong to any namespace.) When such a configuration must reference a
`Secret`, because Kubernetes has no cluster-scoped "ClusterSecret" resource
type, the question is raised of exactly which namespace a `Secret` that is
_conceptually_ cluster-scoped should belong to.

The existence of the **system resources namespace** provides an answer to this
conundrum.

An example `ClusterConfig`:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: gh-wh-secret
  namespace: kargo-system-resources
  labels:
    kargo.akuity.io/cred-type: generic
data:
  secret: <base64-encoded secret>
---
apiVersion: kargo.akuity.io/v1alpha1
kind: ClusterConfig
metadata:
  # Note this resource is not namespaced
  name: cluster
spec:
  webhookReceivers: 
    - name: gh-wh-receiver
      github:
        # Referenced Secrets are implicitly known to be in
        # the system resources namespace (kargo-system-resources)
        secretRef:
          name: gh-wh-secret
```

:::info

Prior to Kargo v1.9.0, what is now the system resources namespace was referred
to as the "cluster secrets namespace." Two factors prompted the
Kargo team to refine and rename the concept:

* "Cluster" was prone to various misinterpretations.
* "Secrets" was too specific. The Kargo team does not anticipate that `Secret`
  resources will forever be the only type of namespaced resource that will be
  referenced by `ClusterConfig` as a workaround for a non-existent
  cluster-scoped analog.

:::

:::warning[Migration]

If you are migrating from a Kargo version lesser than v1.9.0 to version v1.9.0
or greater, please consult the [migration](#migrating-from-kargo--190) section
at the bottom of this page.

:::

### Configuring the System Resources Namespace

The **system resources namespace**, by default, is `kargo-system-resources`.
Operators may override this at the time of installation or upgrade by overriding
the Kargo Helm chart's `global.systemResources.namespace` setting.

## Migrating from Kargo < 1.9.0

Kargo v1.9.0 introduced terminology and configuration changes to better reflect
the intended use of what are now the **shared resources namespace** and **system
resources namespace**. These changes are summarized here.

**Terminology Changes:**

* Global credentials namespaces (plural) → **shared resources namespace** (singular)
* Cluster secrets namespace → **system resources namespace**

**Chart Setting Changes:**

* `controller.globalCredentials.namespaces` → `global.sharedResources.namespace`

  * The old setting had no default value(s).
  
  * The new setting has a default value of `kargo-shared-resources`.

  * The move from the `controller` section of the chart's settings to the
    `global` section reflects that this configuration is used by more than one
    Kargo component.

* `global.clusterSecretsNamespace` → `global.systemResources.namespace`

  * The old setting had a default value of `kargo-cluster-secrets`.

  * The new setting has a default value of `kargo-system-resources`.

**Automatic Migration:**

Kargo versions **v1.9.0 through v1.11.x** will automatically and continuously
perform a one-way sync of `Secret` resources from their old locations to their
new locations, with a few exceptions:

* If the old `controller.globalCredentials.namespaces` setting was empty (as it
  had no default values(s)), there will be no `Secret` resources in need of
  migration to the namespace specified by the new
  `global.sharedResources.namespace`.

* Due to the potential for name conflicts if Kargo were to attempt consolidating
  resources from multiple namespaces into a single namespace, a chart upgrade to
  v1.9.0 through v1.11.0 will **fail** if the old
  `controller.globalCredentials.namespaces` setting specified _multiple
  namespaces_. In this case (believed to be an outlier), the operator will need
  to migrate affected resources manually.

* If the value of the new `global.sharedResources.namespace` matches the value
  of the old `controller.globalCredentials.namespaces[0]` setting, no migration
  of shared `Secret` resources will be necessary.

* If the value of the new `global.systemResources.namespace` matches the value
  of the old `global.clusterSecretsNamespace` setting, no migration of system
  `Secret` resources will be necessary.

Kargo v1.12.0 will remove the automatic migration and upgrades to that version
or greater will **fail** if values are detected for any of the old settings.

**What this means, practically speaking:**

* New installations of Kargo need not be concerned with any of this.

* If you are upgrading:

  * If you manually manage credentials using the Kargo UI, everything will just
    work. Post upgrade, `Secret`s will automatically sync from their old
    locations to their new locations. Kargo will use and manage `Secret`s in
    their new locations. With due caution, you may manually delete the old
    namespaces using `kubectl`.
  
  * If you are a more advanced operator who GitOps'es your `Secret`s, you do
    not need to act with any urgency.

    If you initially do nothing, `Secret`s will continue to be synced from your
    GitOps repository to their original locations. Kargo will sync those
    `Secret`s to their new locations. Everything will behave as it should.

    You will have until Kargo v1.12.0 to update Kargo `Secret` manifests in your
    GitOps repository to reference their new namespaces. Depending on the
    configuration of the GitOps agent managing Kargo (e.g. Argo CD), `Secret`s
    may automatically be pruned from their old locations. If not, then with due
    caution, you may manually delete the old namespaces using `kubectl`.

    Summarizing the above, no matter what you do, things should continue working
    until upgrading to v1.12.0 and this should afford operators sufficient time
    to make the very minimal changes required to keep things running smoothly in
    v1.12.0 and beyond.
