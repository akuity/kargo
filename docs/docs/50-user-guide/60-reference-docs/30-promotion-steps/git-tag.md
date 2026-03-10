---
sidebar_label: git-tag
description: Creates a new tag for the latest committed changes.
---

# `git-tag`

The `git-tag` step creates a new, annotated tag in a local Git repository
referencing the current `HEAD` of a checked-out branch.

## Configuration

| Name   | Type     | Required | Description                                                                 |
|--------|----------|----------|-----------------------------------------------------------------------------|
| `path` | `string` | Y        | Path to a working directory of a local repository. This path is relative to the temporary workspace that Kargo provisions for use by the promotion process. |
| `tag`  | `string` | Y        | The tag to create. |
| `message` | `string` | Y | The message with which to annotate the tag. |
| `tagger.name` | `string` | N | The tagger's name. If unspecified, defaults to the repo-level configuration specified when the repo was cloned. Can also be configured at the system level. |
| `tagger.email` | `string` | N | The tagger's email address. If unspecified, defaults to the repo-level configuration specified when the repo was cloned. Can also be configured at the system level. |
| `tagger.signingKey` | `string` | N | The GPG signing key for the tagger. If provided `tagger.name` and `tagger.email` must also be provided and must match the email and name in the uid of the associated GPG key. |

## Output

| Name  | Type     | Description                                                                 |
|-------|----------|-----------------------------------------------------------------------------|
| `commit` | `string` | The ID (SHA) of the commit pushed by this step. |

## Examples

### Basic Usage

In this example, the `git-tag` step creates a tag named `v1.0.0` in a local Git repository.

```yaml
steps:
- uses: git-tag
  config:
    path: ./out
    tag: v1.0.0
```

### Tagging After a Commit

This example demonstrates how to use the git-tag step after a git-commit step to tag the latest commit with a version number.

```yaml
steps:
- uses: git-commit
  config:
    path: ./out
    message: "Committing changes for release v1.0.0"
- uses: git-tag
  config:
    path: ./out
    tag: v1.0.0
```

### Pushing After Tagging

In this example, the `git-tag` step creates a tag, and the `git-push` step pushes the tag to the remote repository.

```yaml
steps:
- uses: git-tag
  config:
    path: ./out
    tag: v1.0.0
- uses: git-push
  config:
    path: ./out
    tag: v1.0.0
```

### Creating A Signed Tag

In this example, the `git-tag` step creates a signed tag using the provided 
`tagger.signingKey` by sourcing it from an existing secret in the same namespace
using the [`secret()`](../40-expressions.md#secretname) expression function.

:::note

Tagger signing information may have been configured at the system level by a 
Kargo admin. If system-level configuration exists, the example shown below 
would override it.

:::

```yaml
steps:
- uses: git-tag
  config:
    path: ./out
    tag: v1.0.0
    message: My example tag
    tagger:
      name: Me
      email: me@example.com
      signingKey: ${{ secret('my-gpg-secret').privateKey }}
```

:::note

If `tagger.signingKey` is provided, but `tagger.name` and `tagger.email` do not
match the key's UID, tagging will fail.

:::