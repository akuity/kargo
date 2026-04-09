---
sidebar_label: github-push
description: Pushes committed changes to a GitHub repository using the GitHub API, enabling the possibility of verified commits without associating a signing key with a user account.
---

# `github-push`

<span class="tag beta"></span>

`github-push` pushes committed changes from a specified working tree to a GitHub
repository using the GitHub REST API. It is a drop-in replacement for the
[`git-push`](git-push.md) step.

Many organizations prefer that Kargo authenticates to GitHub using a
[GitHub App](../../50-security/30-managing-secrets.md#github-app-authentication)
because it avoids coupling authentication to any one GitHub user account. GitHub
Apps, however, cannot be associated with a GPG signing key, so when branch
protection rules require verified commits, the key used for signing must still
be associated with a user account -- which undermines the benefits of having
used an App for authentication.

`github-push` solves this by "replaying" local commits using the GitHub REST
API. When a commit is eligible (see
[Trust, attribution, and verification](#trust-attribution-and-verification)
below), author/committer information is withheld from the API call. Under those
conditions, GitHub attributes the commit to the authenticated user (the GitHub
App) and signs it with GitHub's own key, producing a verified commit -- without
requiring a GPG key to be associated with any user account.

This step also implements its own, internal retry logic. If a push fails, with
the cause determined to be the presence of new commits in the remote branch that
are not present in the local branch, the step will integrate the remote changes
and retry the push. Any merge conflict requiring manual resolution will
immediately halt further attempts.

How remote changes are integrated is controlled by the system-level __push
integration policy__, which is configured by a Kargo operator. The available
policies are:

- `AlwaysRebase` — Unconditionally rebase. Simplest, but may re-sign or strip
  commit signatures.

- `RebaseOrMerge` — Rebase when a signature-trust analysis determines it is
  safe; merge otherwise. This preserves linear history when possible without
  undermining trust.

- `RebaseOrFail` — Rebase when the signature-trust analysis determines it is
  safe; fail the step otherwise.

- `AlwaysMerge` — Unconditionally create a merge commit. Most conservative.

:::caution

For parity with the `git-push` step, for which this step is a drop-in
replacement, the current default policy is `AlwaysRebase`.

Starting with v1.12.0, the default will change to `RebaseOrMerge` for both
steps.

:::

When the policy evaluates rebase safety (`RebaseOrMerge` and `RebaseOrFail`),
the decision to rebase or not is based on the GPG signature status of the local
commits that would be replayed as part of that rebase (not to be confused with
replaying of commits on GitHub via its REST API):

- If all local commits are __signed by a trusted key__ and signing was enabled
  at the time the repository was cloned, a __rebase__ is performed. The
  replacement commits are re-signed by Kargo.

- If all local commits are __unsigned__ and signing was _not_ enabled at the
  time the repository was cloned, a __rebase__ is performed. The replacement
  commits remain unsigned.

- In all other cases, the policy's fallback behavior applies (merge or fail).

A "trusted key" is one that was imported with ultimate trust when the repository
was cloned, either through explicit configuration of the
[`git-clone`](git-clone.md) step or via fallback on a system-level signing key
configured by a Kargo admin.

:::info

For more information on configuring the push integration policy, see the
[operator guide](../../../40-operator-guide/20-advanced-installation/30-common-configurations.md#push-integration-policy).

:::

:::info

This step's internal retry logic is helpful in scenarios when concurrent
Promotions to multiple Stages may all write to the same branch of the same
repository.

Because conflicts requiring manual resolution will halt further attempts, it is
recommended to design your Promotion processes such that Promotions to multiple
Stages that write to the same branch do not write to the same files.

:::

## Credentials

This step utilizes the [repository credentials](../../50-security/30-managing-secrets.md#repository-credentials)
system to access GitHub repositories and APIs.

:::caution

This step will succeed when authenticated to GitHub via a personal access token,
however, can only (if other conditions are met) result in verified commits when
authenticated via
a [GitHub App](../../50-security/30-managing-secrets.md#github-app-authentication).

:::

## How it works

Under the hood, `github-push`:

1. Integrates any remote changes into the local branch (identical to
   [`git-push`](git-push.md)).

2. Force-pushes the local branch to a temporary, non-visible staging ref on
   GitHub (`refs/kargo/staging/...`).

3. Uses the GitHub API to compare the staging ref against the target branch,
   identifying the commits that need to be "replayed."

4. Replays each commit via the GitHub API, applying the appropriate trust and
   attribution rules. As a safety guardrail, the number of commits replayed in
   a single push is capped (default 10). See the
   [operator guide](../../../40-operator-guide/20-advanced-installation/30-common-configurations.md#github-push-settings)
   to configure this limit.

5. Updates the target branch ref to point at the final replayed commit.

6. Cleans up the staging ref.

7. Syncs the local working tree to match the remote (since the replayed
   commits have new SHAs).

## Trust, attribution, and verification

When replaying commits through the GitHub API, `github-push` decides how each
commit should be attributed. This decision determines whether the commit will
receive GitHub's verified badge.

By default, the decision for each replayed commit is based on whether the
original commit was signed by a trusted key:

- __Trusted commits__ -- those signed by a key with ultimate trust in the GPG
  keyring -- are created __without__ explicit author/committer information.
  GitHub attributes the commit to the authenticated identity (App or user) and
  signs it with its own key, resulting in a __verified__ commit.

  If the original commit's author differs from its signer, a `Co-authored-by`
  trailer is added to the commit message to preserve that attribution.

- __Untrusted commits__ -- unsigned, or signed by an untrusted key -- are
  created __with__ their original author and committer information preserved
  exactly. GitHub will __not__ mark these as verified. Provenance remains
  completely intact.

:::caution

For any commits to be considered trusted, Kargo __must__ be configured with a
GPG signing key so that it signs commits locally. Without a signing key, all
commits are untrusted and will be replayed with their original attribution --
meaning no verified badge.

A signing key can be configured at the system level in two ways: at runtime via
[`ClusterConfig`](../../../40-operator-guide/20-advanced-installation/30-common-configurations.md#git-client-configuration),
or at install time via the Helm chart. The runtime path takes precedence.

This does __not__ undermine the purpose of the feature. The GPG key configured
for Kargo never needs to be uploaded to GitHub or associated with any user
account. It is used solely for local signing and trust evaluation. It can be a
purpose-built, "throwaway" key that exists only within Kargo's configuration.

:::

### How integration policies affect verification

The choice of integration policy can affect which commits are trusted when
replayed:

- With `AlwaysRebase`, all commits are rebased and re-signed by the configured
  signing key, making them trusted. After replay, they all receive the verified
  badge. However, this means commits that Kargo did not originally author are
  re-signed -- the operator has accepted this trade-off by choosing this policy.

- With `RebaseOrMerge` or `AlwaysMerge`, commits from the remote branch that
  were not originally signed by a trusted key are preserved as-is through the
  merge. When replayed, these commits retain their original attribution and will
  __not__ be verified. The merge commit itself and any other commits Kargo
  authored will be verified.

- With `RebaseOrFail`, the step fails if any commit cannot be safely rebased,
  so all replayed commits will be trusted and verified (if the step succeeds).

### Overriding trust: verifying all commits

For operators who want __every__ commit to receive the verified badge regardless
of trust, a system-level option is available that causes author/committer
information to be omitted for all commits -- not just trusted ones. GitHub then
signs every commit with its own key, and every commit in the push receives the
verified badge.

To preserve provenance, a `Co-authored-by` trailer is added to the commit
message whenever the original author's identity is known.

:::warning

This option can manufacture trust where none exists. It tells GitHub to vouch
for commits that Kargo could not independently verify. Enabling this places the
superficial value of a verified badge above genuine cryptographic trust.

:::

:::info

For more information on configuring this option, see the
[operator guide](../../../40-operator-guide/20-advanced-installation/30-common-configurations.md#github-push-settings).

:::

## Configuration

| Name | Type | Required | Description |
| ---- | ---- | -------- | ----------- |
| `path` | `string` | Y | Path to a Git working tree containing committed changes. |
| `targetBranch` | `string` | N | The branch to push to in the remote repository. Mutually exclusive with `generateTargetBranch=true`. If neither is provided, the target branch will be the same as the branch currently checked out in the working tree. |
| `maxAttempts` | `int32` | N | The maximum number of attempts to make when pushing to the remote repository. Default is 10. |
| `generateTargetBranch` | `boolean` | N | Whether to push to a remote branch named like `kargo/promotion/<promotionName>`. If such a branch does not already exist, it will be created. A value of `true` is mutually exclusive with `targetBranch`. This option is useful when a subsequent promotion step will open a pull request against a Stage-specific branch. |
| `force` | `boolean` | N | Whether to force push to the target branch, overwriting any existing history. This is useful for scenarios where you want to completely replace the branch content (e.g., pushing rendered manifests that don't depend on previous state). **Use with caution** as this will overwrite any commits that exist on the remote branch but not in your local branch. Default is `false`. |
| `insecureSkipTLSVerify` | `boolean` | N | Whether to skip TLS verification when communicating with the GitHub API. Default is `false`. Intended for GitHub Enterprise instances with self-signed certificates. |

## Output

| Name | Type | Description |
| ---- | ---- | ----------- |
| `branch` | `string` | The name of the remote branch pushed to by this step. This is especially useful when the `generateTargetBranch=true` option has been used, in which case a subsequent [`git-open-pr`](git-open-pr.md) will typically reference this output to learn what branch to use as the head branch of a new pull request. |
| `commit` | `string` | The ID (SHA) of the commit pushed by this step. |
| `commitURL` | `string` | The URL of the commit that was pushed to the remote repository. |

## Examples

### Common usage

In this example, changes prepared in a working directory are committed and pushed
to the same branch that was checked out. This is the simplest usage, directly
replacing `git-push`:

```yaml
steps:
# Clone, prepare the contents of ./out, etc...
- uses: git-commit
  config:
    path: ./out
    message: rendered updated manifests
- uses: github-push
  config:
    path: ./out
```

### For use with a pull request

In this example, changes are pushed to a generated branch name that follows the
pattern `kargo/promotion/<promotionName>`. By setting
`generateTargetBranch: true`, the step creates a unique branch name that can
be referenced by subsequent steps.

This is commonly used as part of a pull request workflow, where changes are
first pushed to an intermediate branch before being proposed as a pull request.
The step's output includes the generated branch name, which can then be used by
a subsequent [`git-open-pr` step](git-open-pr.md).

```yaml
steps:
# Clone, prepare the contents of ./out, etc...
- uses: git-commit
  config:
    path: ./out
    message: rendered updated manifests
- uses: github-push
  as: push
  config:
    path: ./out
    generateTargetBranch: true
# Open a PR and wait for it to be merged or closed...
```
