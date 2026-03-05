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
| `tagger.name` | `string` | N | The tagger's name. If unspecified, defaults to the repo-level configuration specified when the repo was cloned. Can also be configured at the system level. |
| `tagger.email` | `string` | N | The tagger's email address. If unspecified, defaults to the repo-level configuration specified when the repo was cloned. Can also be configured at the system level. |
| `tagger.signingKey` | `string` | Y | The GPG signing key for the tagger. |
| `tagger.signingMessage` | `string` | Y | The message to annotate the tag with. If unspecified, defaults to the repo-level configuration specified when the repo was cloned. Can also be configured at the system level. |

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
tagger signing information.

:::note

Tagger signing information may have been configured at the system level by a 
Kargo admin. If a system-level configuration exists, the example shown below 
would override it.

:::

```yaml
steps:
- uses: git-tag
  config:
    path: ./out
    tag: v1.0.0
    tagger:
      name: yourname
      email: your@inbox.com
      signingKey: ${{ secret('my-gpg-secret').privateKey }}
      signingMessage: legitness
```