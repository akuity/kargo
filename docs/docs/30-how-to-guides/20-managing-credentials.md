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
