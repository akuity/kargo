---
description: Learn how to manage user permissions
sidebar_label: Managing user permissions
---

# Managing User Permissions

Kargo is typically configured to support single-sign-on (SSO) using an external
identity provider that implements the
[OpenID Connect](https://openid.net/developers/how-connect-works/) protocol.

:::info
Refer to
[the advanced section of the installation guide](./10-installing-kargo.md#advanced-installation)
for more details on how to configure Kargo to use an external identity provider.
:::

Kargo also implements authorization of all user actions using pure Kubernetes
[RBAC](https://kubernetes.io/docs/reference/access-authn-authz/rbac/). i.e.
Permission to perform various actions on various Kargo resources is therefore
granted via `RoleBinding` resources that associate users or `ServiceAccount`
resources with `Role` resources.

Because Kargo users log into the Kargo CLI or UI via SSO, their identifies are
not known to Kargo's underlying Kubernetes cluster. This represents an
impediment to using Kubernetes RBAC to authorize the actions of such users.

Kargo answers this challenge through a scheme that permits users to be mapped
to zero or more Kubernetes `ServiceAccount` resources. The remainder of this
page describes how to create those mappings.

## User to `ServiceAccount` Mappings

Whether logged into the Kargo CLI or UI, Kargo users are interacting with Kargo
via the Kargo API server. The Kargo API server authenticates users via a bearer
token issued by the external identity provider. On every request, the Kargo
API server validates and decodes the token to obtain trusted information about
the user. This includes:

* The user's unique identifier (the standard OpenID Connect `sub` claim)
* The user's email address (the standard OpenID Connect `email` claim)
* Groups to which the user belongs (the non-standard, but widely supported
  `groups` claim)

Also at the time of authentication, the Kargo API server queries the Kubernetes
API server to obtain a list of all `ServiceAccount` resources to which the user
has been mapped. Kargo typically restricts this search to `ServiceAccount` resources
in Kargo project namespaces only (i.e. only those labeled with 
`kargo.akuity.io/project: "true"`). Refer to the next section for exceptions to
this rule.

ServiceAccount resources may be mapped to users through the use of three
different annotations:

* `rbac.kargo.akuity.io/sub`: This annotation's value may be a comma-delimited
  list of user identifiers. All users in the list will be mapped to the
  `ServiceAccount`.

* `rbac.kargo.akuity.io/email`: This annotation's value may be a comma-delimited
  list of user email addresses. All users in the list having one of the listed
  email addresses will be mapped to the `ServiceAccount`.

* `rbac.kargo.akuity.io/groups`: This annotation's value may be a
  comma-delimited list of group identifiers. All users in the list belonging to
  one or more of the listed groups will be mapped to the `ServiceAccount`.

In the following example, the `ServiceAccount` resource is mapped to all of:

* A user identified as `bob`.
* A user with the email address `alice@example.com`.
* All users in the group `devops`.

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: admin
  namespace: kargo-demo
  annotations:
    rbac.kargo.akuity.io/sub: bob
    rbac.kargo.akuity.io/email: alice@example.com
    rbac.kargo.akuity.io/groups: devops
```

:::note
Kargo has no built-in functionality for managing these `ServiceAccount`
resources and their annotations at this time, meaning they must currently be
managed through other means, such as `kubectl` or Argo CD.
:::

A user may be mapped to multiple `ServiceAccount` resources. A user's effective
permissions are therefore the union of the permissions associated with all such
`ServiceAccount` resources.

## Global Mappings

In cases where certain, broad sets of permissions may be required by a large
numbers of users, the administrator/operator installing Kargo may opt-in to
designating one or more namespaces as homes for "global" `ServiceAccount`
resources using the `api.oidc.globalServiceAccounts` setting in Kargo's Helm
chart. Refer to
[the advanced section of the installation guide](./10-installing-kargo.md#advanced-installation)
for more details.

Note that `ServiceAccount` resources in designated namespaces are not _truly_
global because they are _still_ mapped to users according to the rules described
in the previous section.

Making use of this feature could be, for instance, a convenient method of
granting read-only access to all Kargo resources in all projects to all users
within an organization. Additional permissions may then be granted to users on a
project-by-project basis.
