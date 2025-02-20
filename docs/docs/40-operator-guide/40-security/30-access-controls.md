---
sidebar_label: Access Controls
---

# Access Controls

Most access controls in Kargo are within the purview of highly-privileged
_users_ -- ones who might be considered to be "project admins." There are only
a few access controls that an operator might need to be concerned with and this
documentation focuses on those.

:::note
__Not what you were looking for?__

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
[OpenID Connect](./20-openid-connect.md) section of the Operator Guide.

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
[OpenID Connect](./20-openid-connect.md) section of the Operator Guide.

Also for every request, the Kargo API server queries the Kubernetes API server
to obtain a list of all `ServiceAccount` resources to which the user has been
mapped. This search is mostly limited to `ServiceAccount` resources in Kargo
project namespaces only (i.e. only those labeled with
`kargo.akuity.io/project: "true"`). _This section focuses on the exceptions to
that rule._

ServiceAccount resources may be mapped to users through the use of annotations
whose key begins with `rbac.kargo.akuity.io/claim.`. The value of the annotation
may be a single value, or a comma-delimited list of values.

In the following example, the `ServiceAccount` resource is mapped to all of:

* Users with a `sub` claim identifying them as either `alice` or `bob`.
* A user with the `email` claim `carl@example.com`.
* All users with a `groups` claim  containing _either_ the `devops` or
  `kargo-admin` group.

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
permissions are therefore the _union_ of the permissions associated with all
such `ServiceAccount` resources.

### Global Mappings

Now that we've seen how users are mapped to `ServiceAccount` resources, we can
zero in on the few places where these details are relevant to the operator role.

As previously mentioned, _most_ access controls are managed at the project level
by project admins, however, there are two ways in which an operator can also
map users to `ServiceAccount` resources.

#### `api.oidc.admins` / `api.oidc.viewers`

The `api.oidc.admins` and `api.oidc.viewers` configuration options of the Kargo
Helm chart permit an operator to map users with specific claims to
_system-wide_ admin and viewer roles respectively. If, for example, every user
in the group `devops` should be an admin, and every user in the group
`developers` should be a viewer, you would set these accordingly:

```yaml
api:
  oidc:
    # ... omitted for brevity ...
    admins:
      claims:
        groups:
        - devops
    viewer:
      claims:
        groups:
        - developers
```

Behind the scenes, the configuration above merely results in the `kargo-admin`
and `kargo-viewer` `ServiceAccounts` in the namespace in which Kargo is
installed being annotated as discussed in the previous section.

`kargo-admin`:

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: kargo-admin
  namespace: kargo
  annotations:
    rbac.kargo.akuity.io/claim.groups: devops
```

`kargo-viewer`:

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: kargo-viewer
  namespace: kargo
  annotations:
    rbac.kargo.akuity.io/claim.groups: developers
```

`ClusterRoleBinding` resources associating these `ServiceAccount` resources with
the correct permissions are pre-defined by the chart.

:::note
It is common to map _all_ authenticated users to the `kargo-viewer`
`ServiceAccount` to effect broad read-only permissions. These permissions _do
not_ extend to credentials and other project `Secret`s.
:::

:::info
For additional information, once again, refer to the
[OpenID Connect](./20-openid-connect.md) section of the Operator Guide.
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
