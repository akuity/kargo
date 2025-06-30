---
sidebar_label: git-clone
description: Clones a remote Git repository and checks out specified revisions to working trees at specified paths.
---

# `git-clone`

`git-clone` is often the first step in a promotion process. It creates a
[bare clone](https://git-scm.com/docs/git-clone#Documentation/git-clone.txt-code--barecode)
of a remote Git repository, then checks out one or more branches, tags, or
commits to working trees at specified paths. Checking out different revisions to
different paths is useful for the common scenarios of combining content from
multiple sources or rendering Stage-specific manifests to a Stage-specific
branch.

:::note
It is a noteworthy limitation of Git that one branch cannot be checked out in
multiple working trees.
:::

## Configuration

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `repoURL` | `string` | Y | The URL of a remote Git repository to clone. |
| `insecureSkipTLSVerify` | `boolean` | N | Whether to bypass TLS certificate verification when cloning (and for all subsequent operations involving this clone). Setting this to `true` is highly discouraged in production. |
| `author` | `[]object` | N | Default authorship information for any commits made to the cloned repository. If provided, this overrides any system-level defaults. Note: Configuration of the [`git-commit`](./git-commit.md) step can override this information. |
| `author.name` | `string` | Y | The committer's name. |
| `author.email` | `string` | Y | The committer's email address. |
| `author.signingKey` | `string` | N | The GPG signing key for the author. This field is optional. |
| `checkout` | `[]object` | Y | The commits, branches, or tags to check out from the repository and the paths where they should be checked out. At least one must be specified. |
| `checkout[].as` | `string` | N | Used as the key in the `commits` output map. If not specified, the value of the `path` field is used as a key instead. Providing a value for this field is useful when expressions in downstream steps may need to reference specific commits checked out by this step. |
| `checkout[].branch` | `string` | N | A branch to check out. Mutually exclusive with `commit` and `tag`. If none of these is specified, the default branch will be checked out. |
| `checkout[].create` | `boolean` | N | In the event `branch` does not already exist on the remote, whether a new, empty, orphaned branch should be created. Default is `false`, but should commonly be set to `true` for Stage-specific branches, which may not exist yet at the time of a Stage's first promotion. |
| `checkout[].commit` | `string` | N | A specific commit to check out. Mutually exclusive with `branch` and `tag`. If none of these is specified, the default branch will be checked out. |
| `checkout[].path` | `string` | Y | The path for a working tree that will be created from the checked out revision. This path is relative to the temporary workspace that Kargo provisions for use by the promotion process. |
| `checkout[].tag` | `string` | N | A tag to check out. Mutually exclusive with `branch` and `commit`. If none of these is specified, the default branch will be checked out. |

## Output

| Name    | Type                | Description                                                                                      |
|---------|---------------------|--------------------------------------------------------------------------------------------------|
| `commits` | `map[string]string` | A map of checkout keys (either the `as` alias or the `path` if `as` is not set) to the HEAD commit hash checked out at each path. This allows downstream steps to reference the exact commit checked out for each working tree. |

## Examples

### Common Usage

The most common usage of this step is to check out a commit specified by the
Freight being promoted as well as a Stage-specific branch. Subsequent steps are
likely to perform actions that revise the contents of the Stage-specific branch
using the commit from the Freight as input.

:::info
For more information on `commitFrom` and expressions, see the
[Expressions](../40-expressions.md#functions) documentation.
:::

```yaml
vars:
- name: gitRepo
  value: https://github.com/example/repo.git
steps:
- uses: git-clone
  config:
    repoURL: ${{ vars.gitRepo }}
    checkout:
    - commit: ${{ commitFrom(vars.gitRepo).ID }}
      path: ./src
    - branch: stage/${{ ctx.stage }}
      create: true
      path: ./out
# Prepare the contents of ./out ...
# Commit, push, etc...
```

### Combining Multiple Sources

For this more advanced example, consider a Stage that requests Freight from two
Warehouses, where one provides Kustomize "base" configuration, while the other
provides a Stage-specific Kustomize overlay. Rendering the manifests intended
for such a Stage will require combining the base and overlay configurations
with the help of a [`copy` step](copy.md). For this case, a `git-clone` step
may be configured similarly to the following.

:::info
For more information on `commitFrom` and expressions, see the
[Expressions](../40-expressions.md#functions) documentation.
:::

```yaml
vars:
- name: gitRepo
  value: https://github.com/example/repo.git
steps:
- uses: git-clone
  config:
    repoURL: ${{ vars.gitRepo }}
    checkout:
    - commit: ${{ commitFrom(vars.gitRepo, warehouse("base")).ID }}
      path: ./src
    - commit: ${{ commitFrom(vars.gitRepo, warehouse(ctx.stage + "-overlay")).ID }}
      path: ./overlay
    - branch: stage/${{ ctx.stage }}
      create: true
      path: ./out
- uses: git-clear
  config:
    path: ./out
- uses: copy
  config:
    inPath: ./overlay/stages/${{ ctx.stage }}/kustomization.yaml
    outPath: ./src/stages/${{ ctx.stage }}/kustomization.yaml
- uses: kustomize-build
  config:
    path: ./src/stages/${{ ctx.stage }}
    outPath: ./out
# Commit, push, etc...
```
