---
sidebar_label: Access Controls
---

# Access Controls

Most access controls in Kargo are within the purview of highly-privileged
_users_ -- ones who might be considered to be "project admins." There are only
a few access controls that an operator might need to be concerned with and this
documentation focuses on those.


:::info[Not what you were looking for?]

If you're a project admin looking to understand more about access controls,
you may find some value in this document, but most of what you need to know
can be found in the
[User Guide's Access Controls](../../50-user-guide/50-security/20-access-controls/index.md)
section.

:::

## Overview

Kargo is usually configured to support single-sign-on (SSO) using an identity
provider (IDP) that implements the
[OpenID Connect](https://openid.net/developers/how-connect-works/) (OIDC)
protocol. This topic is explored in much greater depth in the dedicated
[OpenID Connect](./20-openid-connect/index.md) section of the Operator Guide.

Kargo also implements access controls through _pure Kubernetes
[RBAC](https://kubernetes.io/docs/reference/access-authn-authz/rbac/)._

:::info

Kargo's creators learned from previous experience that when APIs are modeled as
Kubernetes resources, it is best to rely solely on Kubernetes-native
authorization mechanisms. By doing so, access controls are enforced even for
users with direct access to the Kargo control plane's underlying cluster who
might shun the Kargo CLI and UI in favor of `kubectl`.

:::

There is a natural _impedance_ between users authenticating to _Kargo_ through
some IDP and access controls being implemented through pure _Kubernetes_ RBAC.
To make a finer point of it: _It is impossible for a Kubernetes cluster to
enforce RBAC for users it does not recognize._ Assuming you do not wish to
resolve this by granting direct cluster access to a large number of developers
and training them to use `kubectl`, a different solution is required.

Kargo resolves this impedance through a simple scheme that permits users
authenticated via the IDP to be _mapped_ to Kubernetes `ServiceAccount`
resources. For the most part, these mappings are best managed at the project
level by project admins. The remainder of this document, therefore, touches on
the few access controls that an operator might need to be concerned with.

## User to `ServiceAccount` Mappings

First, operators should understand how the mapping of users to `ServiceAccount`
resources works.

Most Kargo users interact with Kargo via its API server, using its UI or CLI as
a client. In either case, those users are authenticated by a bearer token issued
by the IDP.

For every request, the Kargo API server validates and decodes the token to
obtain trusted information about the user which, importantly, includes _claims_
such as username, email address, and group membership. The exact claims
available depend on the IDP and the configuration of the Kargo API server. For
more details on this topic, refer to the
[OpenID Connect](./20-openid-connect/index.md) section of the Operator Guide.

Also for every request, the Kargo API server queries the Kubernetes API server
to obtain a list of all `ServiceAccount` resources to which the user has been
mapped. This search is mostly limited to `ServiceAccount` resources in Kargo
project namespaces only (i.e. only those labeled with
`kargo.akuity.io/project: "true"`). _This section focuses on the exceptions to
that rule._

ServiceAccount resources may be mapped to users via the
`rbac.kargo.akuity.io/claims` annotation, whose value is a string representation
of a JSON or YAML object with claim names as its keys and lists of claim values
as its values.

In the following example, the `ServiceAccount` resource is mapped to all of:

* Users with a `sub` claim identifying them as either `alice` or `bob`.
* A user with the `email` claim `carl@example.com`.
* All users with a `groups` claim containing _either_ the `devops` or
  `kargo-admin` group.

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: admin
  namespace: kargo-demo
  annotations:
    rbac.kargo.akuity.io/claims: |
      {
        "sub": ["alice", "bob" ],
        "email": "carl@example.com",
        "groups": ["devops", "kargo-admin"]
      }
```

:::info

Mappings specified using annotations with keys of the form
`rbac.kargo.akuity.io/claim.<name>` with comma-delimited values are also
supported for reasons of backwards compatibility. The effective mapping is
therefore the union of mappings defined using such annotations with any
mappings defined using the newer, recommended `rbac.kargo.akuity.io/claims`
annotation.

:::

A user may be mapped to multiple `ServiceAccount` resources. A user's effective
permissions are therefore the _union_ of the permissions associated with all
such `ServiceAccount` resources.

### Global Mappings

Now that we've seen how users are mapped to `ServiceAccount` resources, we can
zero in on the few places where these details are relevant to the operator role.

As previously mentioned, _most_ access controls are managed at the project level
by project admins, however, there are two ways in which an operator can also
map users to `ServiceAccount` resources.

#### Built-in System Roles

Kargo comes with four specific `ServiceAccount`s pre-defined by its Helm chart,
along with bindings to applicable permissions. These four `ServiceAccount`s can
easily be associated with users having specific claimes through chart
configuration at install-time:

| Name | Configuration Key | Description |
|------|-------------------|-------------|
| `kargo-admin` | `api.oidc.admins` | Complete, cluster-wide access to all Kargo resources. Access to `Secret`s is _not_ cluster-wide, but expands and contracts dynamically as projects and their underlying namespaces are created and deleted. |
| `kargo-viewer` | `api.oidc.viewers` | Read-only, cluster-wide access to all Kargo resources. This does _not_ include any level of access to `Secret`s. |
| `kargo-user` | `api.oidc.users` | The minimum level of permissions that can be granted to a user. It permits only listing `Project`s and viewing system-level configuration. This does _not_ include any level of access to `Secret`s. |
| `kargo-project-creator` | `api.oidc.projectCreators` | The permissions of the user role, plus permission to create new `Project`s. When a project is created by such a user via the CLI or UI (but not through `kubectl`) that user will dynamically receive admin permissions within that project's underlying namespace. This includes access to project `Secret`s. |

If, one wished to make the following associations:

- Alice and Bob should be admins.
- Team leads should be able to create new projects.
- Devops engineers should be able to view everything.
- Developers should have few permissions, with additional permissions granted on
  a project-by-project basis.

And assuming users have `email` and `group` claims, and groups `leads`, `devops`, and
`developers` exist, Kargo could be configured as follows at install-time:

```yaml
api:
  oidc:
    # ... omitted for brevity ...
    admins:
      claims:
        email:
        - alice@example.com
        - bob@example.com
    projectCreators:
      claims:
        groups:
        - leads
    viewers:
      claims:
        groups:
        - devops
    users:
      claims:
        groups:
        - developers
```

Behind the scenes, the configuration above merely results in applicable
`ServiceAccounts` in Kargo's own namespace being annotated as discussed in the
previous section.

For example, the `kargo-admin` `ServiceAccount` will be annotated as follows:

`kargo-admin`:

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: kargo-admin
  namespace: kargo
  annotations:
    rbac.kargo.akuity.io/claims: '{"email":["alice@example.com", "bob@example.com"]}'
```

:::info

For additional information, refer to the
[OpenID Connect](./20-openid-connect/index.md) section of the Operator Guide.

:::

#### Global `ServiceAccount` Namespaces

In some cases, an operator, in collaboration with project admins, may wish to
create a small number of `ServiceAccount` resources that are _pre-mapped_ to
users with specific claims. Operators can assign system-wide permissions to
such `ServiceAccount`s using `ClusterRoleBinding`s. Project admins retain
control over the project-level permissions of those `ServiceAccount`s using
`RoleBinding`s in the project namespaces.

:::info

The main convenience of this approach is that it enables individual
project admins to take the existence of certain classes of user for granted
instead of having to manage user-to-`ServiceAccount` mappings themselves.

:::

:::info

"Global" is a misnomer. `ServiceAccount` resources in designated namespaces are
not truly global because they are still mapped to users according to the rules
described in the previous sections.

:::

Enabling global `ServiceAccount` namespaces requires three steps be taken
by the operator:

1. Create one or more namespaces dedicated to this purpose. Typically, just one
   should suffice.

1. Define `ServiceAccount` resources in these namespaces, each annotated
   appropriately to effect the desired user-to-`ServiceAccount` mappings.

    Optionally, use a `ClusterRoleBinding` to grant any necessary
    system-wide permissions to these `ServiceAccount` resources.

1. Configure Kargo to look for `ServiceAccount` resources in these designated
   namespaces, by setting `api.oidc.globalServiceAccounts.namespaces` at
   installation time. For example:

   ```yaml
   api:
    oidc:
      globalServiceAccounts:
        namespaces:
        - kargo-global-service-accounts
   ```  
