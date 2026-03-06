---
sidebar_label: API Tokens
---

# API Tokens

API tokens enable programmatic access to Kargo APIs without requiring users to
share short-lived credentials with non-human agents, such as CI processes.

:::info[Not what you were looking for?]

If you're looking to manage and use API tokens associated with built-in,
system-level roles refer to the
[Operator Guide's API Tokens](../../40-operator-guide/40-security/35-api-tokens.md)
documentation.

:::

Kargo API tokens are associated directly with Kargo roles. Project admins, or
others with sufficient permissions, are able to create and delete roles as well
as create and delete API tokens associated with those roles.

:::caution[Implementation Detail]

"Kargo roles," including built-in, system level ones are actually abstractions
over trios of Kubernetes `ServiceAccount`, `ClusterRole`, and
`ClusterRoleBinding` resources. Throughout this document, the term "role" refers
to this abstraction.

:::

To learn more about managing Project-level roles, refer to the
[Access Controls](./20-access-controls/index.md) documentation.

## Creating Tokens

To generate a new token associated with a Project-level role:

```shell
kargo create token --project my-project --role my-role my-role-token
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

List all tokens associated with a specific role:

```shell
kargo get tokens --project my-project
```

```shell
NAME            ROLE      KARGO MANAGED   AGE
my-role-token   my-role   true            5m
```

List tokens associated with a specific role:

```shell
kargo get tokens --project my-project --role my-role
```

Retrieve details about a specific token (note that the token value will be
redacted):

```shell
kargo get token --project my-project my-role-token -o yaml
```

## Using Tokens

API tokens can be used with many Kargo or Kubernetes clients. This includes
tools like `kubectl` as well as any programming language client library for
Kubernetes or Kargo.

:::note

While the `kargo` CLI does not directly support specifying a token via command
line flags, you can configure it to use a token by editing
`~/.config/kargo/config`.

:::

## Deleting Tokens

To delete a token when it's no longer needed or to rotate credentials:

```shell
kargo delete token --project my-project my-role-token
```

```shell
token.kargo.akuity.io/my-role-token deleted
```

Verify the token has been deleted:

```shell
kargo get tokens --project my-project
```
