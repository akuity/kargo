---
description: Find out how to manage repository credentials for use by Kargo
sidebar_label: Managing credentials
---

# Managing Credentials

To manage the progression of changes from stage to stage, Kargo will
often require read/write permissions on private GitOps repositories and
read-only permissions on private container image and/or Helm chart repositories.

This section presents an overview of how these credentials can be managed.

## Credentials as Kubernetes `Secret` Resources

Kargo borrows its general credential-management approach from Argo CD, meaning
that credentials are stored as Kubernetes `Secret` resources containing
specially-formatted data. These secrets generally take the following form:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: <name>
  namespace: <namespace>
  labels:
    kargo.akuity.io/secret-type: <secret type>
stringData:
  type: <type>
  url: <repo url>
  username: <username>
  password: <password>
```

The `name` of such a secret is inconsequential and may follow any convention
preferred by the user.

:::info
Kargo uses Kubernetes namespaces to demarcate project boundaries, so related
`Stage` resources always share a namespace.

Secrets representing credentials will typically exist in the same namespace as
the `Stage` resources that will require them. There are exceptions to this rule,
which are covered in the next section.
:::

The label key `kargo.akuity.io/secret-type` and its value, one of `repository`
or `repo-creds`, is important, as it designates the secret as representing
either credentials for a single repository (`repository`) or representing
credentials for multiple repositories whose URLs begin with a common pattern
(`repo-creds`). When searching for credentials, Kargo gives precedence to the
former.

The `Secret`'s `data` field (set above using plaintext in the `stringData`
field), MUST contain the following keys:

* `type`: One of `git`, `image`, or `helm`.

* `url`: The full URL of the repository (if `kargo.akuity.io/secret-type:
  repository`) or a prefix matching multiple repository URLs (if
  `kargo.akuity.io/secret-type: repo-creds`).

* `username`: The username to use when authenticating to the repository. If the
  value of the `password` key is a personal access token, the value of the
  `username` field may be inconsequential. You should consult your Git hosting
  provider's documentation for more information.

* `password`: A password or personal access token.

:::caution
Only username/password (or personal access token) authentication is
fully-supported at this time.
:::

## Global Credentials

In cases where one or more sets of credentials are needed widely across _all_
Kargo projects, the administrator/operator installing Kargo may opt-in to
designating one or more namespaces as homes for "global" credentials using the
`controller.globalCredentials.namespaces` setting in Kargo's Helm chart.
Refer to
[the advanced section of the installation guide](./10-installing-kargo.md#advanced-installation)
for more details.

:::caution
It is important to understand the security implications of this feature. Any
credentials stored in a global credentials namespace will be available to _all_
Kargo projects.
:::

## Borrowing Credentials from Argo CD

In many cases, Kargo and Argo CD will _both_ require credentials for the same
GitOps and/or Helm chart repositories. (Argo CD never has need for container
image repositories.) With this being the case, Kargo has support for _borrowing_
repository credentials from Argo CD.

Argo CD credentials are represented as Kubernetes `Secret` resources that are
formatted identically to those described in the previous section, except that:

* They should always be in the namespace that Argo CD runs in -- commonly
  `argocd`.

* Credentials for an individual repository must be labeled
  `argocd.argoproj.io/secret-type: repository`.

* Credentials for multiple repositories whose URLs begin with a common pattern
  must be labeled `argocd.argoproj.io/secret-type: repo-creds`.

:::info
Consult
[the Argo CD documentation](https://argo-cd.readthedocs.io/en/stable/operator-manual/declarative-setup/#repositories)
for more information.
:::

Since it would be a security risk to allow Kargo to borrow credentials from Argo
CD without the consent of the corresponding `Secret`'s owner, Kargo requires
Argo CD credentials to be specially annotated to indicate Kargo projects
(namespaces) that are permitted to borrow them. Without this annotation, Kargo
will refuse to borrow credentials from Argo CD. This annotation takes the form
`kargo.akuity.io/authorized-projects:<projects>` where `<projects>` is a
comma-separated list of Kargo projects (Kubernetes namespaces containing related
`Stage` resources).

Similarly to when retrieving credentials directly from a Kargo project's own
namespace, when borrowing credentials from Argo CD (if permitted) Kargo gives
precedence to `Secret` resources labeled
`argocd.argoproj.io/secret-type: repository` over those labeled
`argocd.argoproj.io/secret-type: repo-creds`.

Altogether, the order of precedence for credentials is:

1. Secrets in the same namespace as the `Stage` resource that are also
   labeled `kargo.akuity.io/secret-type: repository`.

1. Secrets in the same namespace as the `Stage` resource that are also
   labeled `kargo.akuity.io/secret-type: repo-creds`.

1. Secrets in Argo CD's namespace that are also labeled
   `argocd.argoproj.io/secret-type: repository` and whose
   `kargo.akuity.io/authorized-projects` annotation contains the namespace of
   the `Stage` resource.

1. Secrets in Argo CD's namespace that are also labeled
   `argocd.argoproj.io/secret-type: repo-creds` and whose
   `kargo.akuity.io/authorized-projects` annotation contains the namespace of
   the `Stage` resource.
