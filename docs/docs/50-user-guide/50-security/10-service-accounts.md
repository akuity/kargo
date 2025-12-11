---
sidebar_label: ServiceAccounts
---

# ServiceAccounts

Kargo `ServiceAccount`s enable programmatic access to Kargo APIs without
requiring a user to share short-lived credentials with non-human agents, such as
a CI process.


:::info Not what you were looking for?

If you're looking to use the built-in, system-level `ServiceAccount`s, refer to
the
[Operator Guide's ServiceAccounts](../../40-operator-guide/40-security/25-service-accounts.md)
documentation.

:::

## Understanding Kargo ServiceAccounts

A _Kargo_ `ServiceAccount` is a Kubernetes `ServiceAccount` resource that has
been specially labeled with `rbac.kargo.akuity.io/service-account: "true"` to
identify it as being intended for use with Kargo.

Project admins, or others with sufficient permissions, are able to create and
delete `ServiceAccounts`, assign roles to them or revoke roles from them, and
create and delete associated authentication tokens.

## Managing ServiceAccounts

Project admins can manage `ServiceAccount`s within their project namespace using
the `kargo` CLI or declaratively using Kubernetes manifests (i.e., GitOps'ing
them).

### Creating ServiceAccounts

<Tabs groupId="create-project">
<TabItem value="ui" label="Using the UI">

:::info

UI support for this feature is not yet implemented.

:::

</TabItem>
<TabItem value="cli" label="Using the CLI" default>

To create a ServiceAccount using the `kargo` CLI:

```shell
kargo create serviceaccount --project my-project my-sa
```

```shell
serviceaccount/my-sa-token created
```

Then list `ServiceAccount`s to verify:

```shell
kargo get serviceaccounts --project my-project
```

```shell
NAME    AGE
my-sa   30s
```

</TabItem>
<TabItem value="declaratively" label="Declaratively">

`ServiceAccount`s can be managed declaratively using Kubernetes manifests:

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: my-sa
  namespace: my-project
  labels:
    rbac.kargo.akuity.io/service-account: "true"
```

:::info

The `rbac.kargo.akuity.io/service-account: "true"` label identifies the resource
as a _Kargo_ `ServiceAccount`. The `ServiceAccount` is still completely usable
without this label (i.e. can still be used to interact with Kargo APIs
programmatically), however, neither the UI nor `kargo` CLI will recognize it as
being a _Kargo_ `ServiceAccount`.

:::

:::note

When managing resources declaratively (e.g., via GitOps), do _not_ include a
`rbac.kargo.akuity.io/managed: "true"` annotation. Resources with this
annotation can be modified or deleted via the Kargo API, which may conflict with
declarative management.

:::

</TabItem>
</Tabs>

### Assigning Roles

<Tabs groupId="assign-roles">
<TabItem value="ui" label="Using the UI">

:::info

UI support for this feature is not yet implemented.

:::

</TabItem>
<TabItem value="cli" label="Using the CLI" default>

Grant a _Kargo_ `Role` to a _Kargo_ `ServiceAccount`:

```shell
kargo grant --project my-project \
  --role developer \
  --service-account my-sa
```

```shell
role.rbac.kargo.akuity.io/developer updated
```

:::info

A _Kargo_ `Role` is an abstraction over a trio of `ServiceAccount`, `Role`, and
`RoleBinding` _Kubernetes_ resources. When you grant a Kargo `Role` to a Kargo
`ServiceAccount`, the Kargo `Role`'s underlying Kubernetes `RoleBinding`
resource is updated to include the Kargo `ServiceAccount` as a subject. For
more details about Kargo roles, see the
[Access Controls](./20-access-controls/index.md) documentation.

:::

</TabItem>
<TabItem value="declaratively" label="Declaratively">

You can also manage role assignments declaratively:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: developer
  namespace: my-project
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: developer
subjects:
- kind: ServiceAccount
  name: my-sa
  namespace: my-project
```

</TabItem>
</Tabs>

### Creating Authentication Tokens

<Tabs groupId="create-tokens">
<TabItem value="ui" label="Using the UI">

:::info

UI support for this feature is not yet implemented.

:::

</TabItem>
<TabItem value="cli" label="Using the CLI" default>

To generate a new authentication token for a Kargo `ServiceAccount`:

```shell
kargo create serviceaccounttoken --project my-project \
  --service-account my-sa my-sa-token
```

```shell
Token created successfully!

IMPORTANT: Save this token securely. It will not be shown again.

Token: eyJhbGciOiJSUzI1NiIsImtpZCI6IjdwQ0...
```

:::danger

The token value is displayed only once during creation. Do not lose it!

If you lose the token value, you must delete the token and create a new one _or_
the existing token's value can be retrieved by a user with sufficient permission
using `kubectl` instead of the `kargo` CLI.

:::

List all authentication tokens:

```shell
kargo get serviceaccounttokens --project my-project
```

```shell
NAME          SERVICE ACCOUNT   KARGO MANAGED   AGE
my-sa-token   my-sa             true            5m
```

List authentication tokens for a specific `ServiceAccount`:

```shell
kargo get serviceaccounttokens --project my-project \
  --service-account my-sa
```

Retrieve details about a specific token (note that the token value will be
redacted):

```shell
kargo get serviceaccounttoken --project my-project \
  my-sa-token -o yaml
```

</TabItem>
<TabItem value="declaratively" label="Declaratively">

Authentication tokens can be created declaratively:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: my-sa-token
  namespace: my-project
  labels:
    rbac.kargo.akuity.io/service-account-token: "true"
  annotations:
    kubernetes.io/service-account.name: my-sa
type: kubernetes.io/service-account-token
```

:::info

Kubernetes will automatically populate the token data asynchronously. The
`rbac.kargo.akuity.io/service-account-token: "true"` label is required to
identify the `Secret` as a Kargo `ServiceAccount` token.

For more information about ServiceAccount token secrets, see the
[Kubernetes documentation](https://kubernetes.io/docs/concepts/configuration/secret/#serviceaccount-token-secrets).

:::

</TabItem>
</Tabs>

### Using Authentication Tokens

Authentication tokens can be used with many Kargo or Kubernetes clients. This
includes tools like `kubectl` as well as any programming language client library
for Kubernetes or Kargo.

:::note

While the `kargo` CLI does not directly support specifying a token via command
line flags, you can configure it to use a token by editing
`~/.config/kargo/config`.

:::

### Deleting Authentication Tokens

<Tabs groupId="delete-sa-tokens">
<TabItem value="ui" label="Using the UI">

:::info

UI support for this feature is not yet implemented.

:::

</TabItem>
<TabItem value="cli" label="Using the CLI" default>

To delete a token when it's no longer needed or to rotate credentials:

```shell
kargo delete serviceaccounttoken --project my-project \
  my-sa-token
```

```shell
serviceaccounttoken.kargo.akuity.io/my-sa-token deleted
```

Verify the token has been deleted:

```shell
kargo get serviceaccounttokens --project my-project
```

</TabItem>
<TabItem value="declaratively" label="Declaratively">

To delete a token declaratively, simply remove the `Secret` resource from your
manifests.

</TabItem>
</Tabs>

### Revoking Roles

<Tabs groupId="revoke-roles">
<TabItem value="ui" label="Using the UI">

:::info

UI support for this feature is not yet implemented.

:::

</TabItem>
<TabItem value="cli" label="Using the CLI" default>

Revoke a Kargo `Role` from a `ServiceAccount`:

```shell
kargo revoke --project my-project \
  --role developer \
  --service-account my-sa
```

```shell
role.rbac.kargo.akuity.io/developer updated
```

This removes the `ServiceAccount` from the Kargo `Role`'s underlying Kubernetes
`RoleBinding` resource.

</TabItem>
<TabItem value="declaratively" label="Declaratively">

To revoke a role declaratively, remove the `ServiceAccount` from the `subjects`
list in the `RoleBinding` resource.

</TabItem>
</Tabs>

### Deleting ServiceAccounts

<Tabs groupId="delete-sa">
<TabItem value="ui" label="Using the UI">

:::info

UI support for this feature is not yet implemented.

:::

</TabItem>
<TabItem value="cli" label="Using the CLI" default>

To delete a `ServiceAccount` when it's no longer needed:

```shell
kargo delete serviceaccount --project my-project my-sa
```

```shell
serviceaccount/my-sa deleted
```


:::warning

Deleting a Kargo `ServiceAccount` via the `kargo` CLI automatically deletes all
associated authentication tokens as well.

:::

Verify the `ServiceAccount` has been deleted:

```shell
kargo get serviceaccounts --project my-project
```

</TabItem>
<TabItem value="declaratively" label="Declaratively">

To delete a `ServiceAccount` declaratively, simply remove the `ServiceAccount`
resource from your manifests.

:::warning

Deleting a `ServiceAccount` will invalidate all associated authentication
tokens. Be sure to also remove any associated `Secret` resources containing
tokens.

:::

</TabItem>
</Tabs>

## CLI Aliases and Shortcuts

The `kargo` CLI supports convenient aliases for Kargo `ServiceAccount` commands:

- `serviceaccount`, `serviceaccounts`, `sa`, `sas` all refer to Kargo
  `ServiceAccount`s.
- `serviceaccounttoken`, `serviceaccounttokens`, `sat`, `sats` all refer to
  authentication tokens.

For example:

```shell
kargo get sas --project my-project

kargo create sat --project my-project \
  --service-account my-sa my-sa-token-1
```
