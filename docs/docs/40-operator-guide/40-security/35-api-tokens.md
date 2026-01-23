---
sidebar_label: API Tokens
---

# API Tokens

API tokens enable programmatic access to Kargo APIs without requiring users to
share short-lived credentials with non-human agents, such as CI processes.

:::info[Not what you were looking for?]

If you're a Project admin looking to create and manage API tokens within
your Projects, you may find some value in this document, but most of what you
need to know can be found in the
[User Guide's API Tokens](../../50-user-guide/50-security/25-api-tokens.md)
documentation.

:::

Kargo API tokens are associated directly with Kargo roles. Kargo comes with
several built-in, system-level roles in the namespace where Kargo is installed
(typically `kargo`). These roles provide different levels of system-wide access.

:::caution[Implementation Detail]

"Kargo roles," including built-in, system level ones are actually abstractions
over trios of Kubernetes `ServiceAccount`, `ClusterRole`, and
`ClusterRoleBinding` resources. Throughout this document, the term "role" refers
to this abstraction.

:::

To learn more about built-in, system-level roles, refer to the
[Access Controls](./30-access-controls.md) documentation.

## Creating Tokens

To generate a new token associated with a system-level role:

```shell
kargo create token --system --role kargo-admin kargo-admin-token-1
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

List all tokens associated with a specific system-level role:

```shell
kargo get tokens --system
```

```shell
NAME                  ROLE          KARGO MANAGED   AGE
kargo-admin-token-1   kargo-admin   true            5m
```

List tokens associated with a specific system-level role:

```shell
kargo get tokens --system --role kargo-admin
```

Retrieve details about a specific token (note that the token value will be
redacted):

```shell
kargo get token --system kargo-admin-token-1 -o yaml
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
kargo delete token --system kargo-admin-token-1
```

```shell
token.kargo.akuity.io/kargo-admin-token-1 deleted
```

Verify the token has been deleted:

```shell
kargo get tokens --system
```
