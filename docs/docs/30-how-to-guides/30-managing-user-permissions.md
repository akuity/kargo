---
description: Learn how to manage user permissions
sidebar_label: Managing User Permissions
---

# Managing User Permissions

Kargo is typically configured to support single-sign-on (SSO) using an external
identity provider that implements the
[OpenID Connect](https://openid.net/developers/how-connect-works/) (OIDC)
 protocol.

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
the user which, importantly, includes _claims_ such as username, email address,
and group membership. The exact claims available depend on the identity provider
and the configuration of the Kargo API server. (Refer again to
[the advanced section of the installation guide](./10-installing-kargo.md#advanced-installation).)

Also at the time of authentication, the Kargo API server queries the Kubernetes
API server to obtain a list of all `ServiceAccount` resources to which the user
has been mapped. Kargo typically restricts this search to `ServiceAccount` resources
in Kargo project namespaces only (i.e. only those labeled with 
`kargo.akuity.io/project: "true"`). Refer to the next section for exceptions to
this rule.

ServiceAccount resources may be mapped to users through the use of annotations
whose key begins with `rbac.kargo.akuity.io/claim.`. The value of the annotation
may be a single value, or a comma-delimited list of values.

In the following example, the `ServiceAccount` resource is mapped to all of:

* Users identified as `alice` or `bob`.
* A user with the email address `carl@example.com`.
* All users in _either_ the `devops` or `kargo-admin` group.

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: admin
  namespace: kargo-demo
  annotations:
    rbac.kargo.akuity.io/claim.sub: alice,bob
    rbac.kargo.akuity.io/claim.email: carl@example.com
    rbac.kargo.akuity.io/claim.groups: devops,kargo-admin
```

A user may be mapped to multiple `ServiceAccount` resources. A user's effective
permissions are therefore the union of the permissions associated with all such
`ServiceAccount` resources.

## Managing Project-Level "Kargo Roles" with the CLI

The Kargo CLI offers several conveniences for working with "Kargo Roles," which
are simplified abstractions over Kubernetes `ServiceAccount`, `Role`, and
`RoleBinding` resources.

A Kargo Role combines the "who" of a `ServiceAccount` (with the mapping
annotations described in the previous section) with the "what" of a `Role`
resource's policy rules.

Creating a Kargo Role, therefore, effects creation of an underlying
`ServiceAccount`/`Role`/`RoleBinding` trio. Similarly, deleting a Kargo Role
deletes those same underlying resources. Updating a Kargo Role via `grant` or
`revoke` commands updates the underlying `ServiceAccount` or `RoleBinding`
accordingly.

:::info
Before diving into the commands for Kargo Role management, please note the
following details:

* A Kargo Role _exists_ as long as an underlying `ServiceAccount` resource with
  the same name exists in the Project namespace.
* If any `RoleBinding` resources in the Project namespace reference the
  `ServiceAccount` resource, then all of the `Role` resources referenced by
  those `RoleBinding` resources are also considered part of the Kargo Role, as
  are the `RoleBinding` resources themselves.
* Kargo can only _manage_ Kargo Roles that are:
    * Comprised of precisely one `RoleBinding` and `Role` resource.
    * Explicitly annotated as _being_ Kargo-managed.
* Kargo also normalizes the representation of policy rules in any `Role`
  resource it manages, which is necessary to ensure that `grant` and `revoke`
  operations can modify policy rules accurately, without unintended side
  effects.

In practice, if you GitOps your Project-level `ServiceAccount`, `Role`, and
`RoleBinding` resources, those resources should _not_ be annotated as being
Kargo-managed. This will prevent modification of those resources through
imperative Kargo CLI commands like `grant` and `revoke`.
:::

All Kargo Roles associated with a Project are listed using the `kargo get roles`
command:

```shell
kargo get roles --project kargo-demo 
```

```shell
NAME      KARGO MANAGED   AGE
default   false           20m
```

Here we see that Kargo counts a Kargo Role as existing due to the existence of
the `default` `ServiceAccount` resource that Kubernetes automatically creates in
any new namespace, including Kargo Project namespaces. Because the `default`
`ServiceAccount` lacks the necessary annotations, Kargo does not consider it
Kargo-managed.

We can create a new Kargo-manged Kargo role with the `kargo create role`
command:

```shell
kargo create role developer --project kargo-demo
```

```shell
role.rbac.kargo.akuity.io/developer created
```

If we list the Kargo Roles again, we see the new `developer` Kargo Role, and
that it is Kargo-managed:

```shell
kargo get roles --project kargo-demo
```

```shell
NAME        KARGO MANAGED   AGE
default     false           24m
developer   true            64s
```

We can view a YAML representation of the `developer` Kargo Role and see that
it's not very interesting yet:

```shell
kargo get role developer --project kargo-demo -o yaml
```

```shell
apiVersion: rbac.kargo.akuity.io/v1alpha1
kargoManaged: true
kind: Role
metadata:
  creationTimestamp: "2024-05-01T13:27:20Z"
  name: developer
  namespace: kargo-demo
```

To make things more interesting, we can grant the `developer` Kargo Role to
a users having the value `developer` in their `groups` claim:

```shell
kargo grant --role developer --claim groups=developer --project kargo-demo
```

```shell
role.rbac.kargo.akuity.io/developer updated
```

And we can grant broad permissions on `Stage` resources to the `developer` Kargo
Role:

```shell
kargo grant --role developer \
  --verb '*' --resource-type stages \
  --project kargo-demo
```

```shell
role.rbac.kargo.akuity.io/developer updated
```

We can view the updated `developer` Kargo Role and see that it is now
considerably more interesting:

```shell
kargo get role developer --project kargo-demo -o yaml
```

```yaml
apiVersion: rbac.kargo.akuity.io/v1alpha1
claims:
  - name: groups
    values:
    - developer
kargoManaged: true
kind: Role
metadata:
  creationTimestamp: "2024-05-01T13:27:20Z"
  name: developer
  namespace: kargo-demo
rules:
- apiGroups:
  - kargo.akuity.io
  resources:
  - stages
  verbs:
  - create
  - delete
  - deletecollection
  - get
  - list
  - patch
  - update
  - watch
```

We can also revoke the `developer` Kargo Role from the `developer` group or
revoke any permissions from the `developer` Kargo Role using the `kargo revoke`
command, which supports all the same flags as the `kargo grant` command.

Last, it may sometimes be useful to view a Kargo Role's underlying
`ServiceAccount`, `Role`, and `RoleBinding` resources. This may be useful, for
instance, to users who have managed Project-level permissions imperatively up to
a point and now wish to make the transition to GitOps'ing those permissions.

```shell
kargo get role developer --as-kubernetes-resources --project kargo-demo
```

```yaml
NAME        K8S SERVICE ACCOUNT   K8S ROLE BINDINGS   K8S ROLES   AGE
developer   developer             developer           developer   13m
```

It is also possible to request alternative representations of the underlying
resources:

```shell
kargo get role developer \
  --as-kubernetes-resources -o yaml \
  --project kargo-demo
```

:::note
Output of the above command is not shown here due to its length.
:::

Last, it is, of course, possible to delete a Kargo Role:

```shell
kargo delete role developer --project kargo-demo
```

```shell
role.rbac.kargo.akuity.io/developer deleted
```

## Kargo Role Matrix

The table below outlines the maximum rules required based on the `kargo-admin` ClusterRole. When specifying verbs, it's recommended to apply the principle of least privilege, ensuring access is limited to what is necessary for the specific role.

| **API Groups**              | **Resources**                                  | **Verbs**                                           |
|-----------------------------|------------------------------------------------|-----------------------------------------------------|
| `""`                        | `events`, `namespaces`, `serviceaccounts`      | `get`, `list`, `watch`                              |
| `rbac.authorization.k8s.io` | `rolebindings`, `roles`                        | `get`, `list`, `watch`                              |
| `kargo.akuity.io`           | `freights`, `projects`, `stages`, `warehouses` | `*`                                                 |
| `kargo.akuity.io`           | `stages`                                       | `promote`                                           |
| `kargo.akuity.io`           | `promotions`                                   | `create`, `delete`, `get`, `list`, `watch`          |
| `kargo.akuity.io`           | `freights/status`                              | `patch`                                             |
| `argoproj.io`               | `analysisruns`                                 | `delete`, `get`, `list`, `watch`                    |
| `argoproj.io`               | `analysistemplates`                            | `*`                                                 |

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
