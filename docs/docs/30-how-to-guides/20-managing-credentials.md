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
Kargo uses Kubernetes `Namespace`s to demarcate project boundaries, so related
`Stage` resources always share a `Namespace`.

Secrets representing credentials will typically exist in the same `Namespace` as
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
designating one or more `Namespace`s as homes for "global" credentials using the
`controller.globalCredentials.namespaces` setting in Kargo's Helm chart.
Refer to
[the advanced section of the installation guide](./10-installing-kargo.md#advanced-installation)
for more details.

:::note
Any matching credentials found in a `Project`/`Namespace` take precedence over
those found in a global credentials `Namespace`.
:::

:::caution
It is important to understand the security implications of this feature. Any
credentials stored in a global credentials `Namespace` will be available to
_all_ Kargo projects.
:::

:::caution
Versions of Kargo prior to v0.4.0 used a different mechanism for managing global
:::
