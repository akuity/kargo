---
sidebar_label: ServiceAccounts
---

# ServiceAccounts

Kargo `ServiceAccount`s enable programmatic access to Kargo APIs without
requiring a user to share short-lived credentials with non-human agents, such as
a CI process.


:::info[Not what you were looking for?]

If you're a project admin looking to create and manage `ServiceAccount`s within
your projects, you may find some value in this document, but most of what you
need to know can be found in the
[User Guide's ServiceAccounts](../../50-user-guide/50-security/10-service-accounts.md)
section.

:::

## Understanding Kargo ServiceAccounts

A _Kargo_ `ServiceAccount` is a Kubernetes `ServiceAccount` resource that has
been specially labeled with `rbac.kargo.akuity.io/service-account: "true"` to
identify it as being intended for use with Kargo.

Most `ServiceAccount` management in Kargo happens at the project level, where
project admins can create `ServiceAccount`s, assign them roles, and generate
authentication tokens. However, Kargo also comes with several built-in,
system-level `ServiceAccount`s, which operators may wish to take advantage of.
Users with the system-level `kargo-admin` role can create and delete
authentication tokens for any of these.

## Built-in, System-level ServiceAccounts

Kargo comes with several pre-defined, system-level `ServiceAccount`s in the
namespace where Kargo is installed (typically `kargo`). These `ServiceAccount`s
provide different levels of system-wide access:

| Name | Description |
|------|-------------|
| `kargo-admin` | Complete, cluster-wide access to all Kargo resources, including the ability to manage ServiceAccounts and their tokens in all project namespaces. |
| `kargo-viewer` | Read-only, cluster-wide access to all Kargo resources. This does _not_ include access to `Secret`s or ServiceAccount tokens. |
| `kargo-user` | Minimum permissions that permit listing `Project`s and viewing system-level configuration. Does _not_ include access to `Secret`s. |
| `kargo-project-creator` | Permissions of the `kargo-user` role, plus the ability to create new `Project`s. When a project is created using the API (but not directly via `kubectl`), the ServiceAccount receives admin permissions within that project. |

System-level `ServiceAccount`s can be listed using the `kargo` CLI:

```shell
kargo get serviceaccounts --system
```

```shell
NAME                    KARGO MANAGED   AGE
kargo-admin             false           4d8h
kargo-project-creator   false           4d8h
kargo-user              false           4d8h
kargo-viewer            false           4d8h
```

### Creating Authentication Tokens

To generate a new authentication token for a system-level Kargo
`ServiceAccount`:

```shell
kargo create serviceaccounttoken --system \
  --service-account kargo-admin \
  kargo-admin-token-1
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

List all system-level authentication tokens:

```shell
kargo get serviceaccounttokens --system
```

```shell
NAME                  SERVICE ACCOUNT   KARGO MANAGED   AGE
kargo-admin-token-1   kargo-admin       true            5m
```

List authentication tokens for a specific system-level `ServiceAccount`:

```shell
kargo get serviceaccounttokens --system kargo-admin-token-1
```

Retrieve details about a specific token (note that the token value will be
redacted):

```shell
kargo get serviceaccounttoken --system kargo-admin-token-1 -o yaml
```

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

To delete a token when it's no longer needed or to rotate credentials:

```shell
kargo delete serviceaccounttoken --system kargo-admin-token-1
```

```shell
serviceaccounttoken.kargo.akuity.io/kargo-admin-token-1 deleted
```

Verify the token has been deleted:

```shell
kargo get serviceaccounttokens --system
```

## CLI Aliases and Shortcuts

The `kargo` CLI supports convenient aliases for Kargo `ServiceAccount` commands:

- `serviceaccount`, `serviceaccounts`, `sa`, `sas` all refer to Kargo
  `ServiceAccount`s.
- `serviceaccounttoken`, `serviceaccounttokens`, `sat`, `sats` all refer to
  authentication tokens.

For example:

```shell
kargo get sas --system

kargo create sat --system --service-account kargo-admin \
  kargo-admin-token-1
```
