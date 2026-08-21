# git_steps

Exercises pull-request and tag git promotion steps, with **no Argo CD -- only git
steps**. Modeled on the prod stage of `git_commit_only`, it has one stage per
step under test, each promoting freight directly from the Warehouse:

| Stage | Step verified | How |
| --- | --- | --- |
| `merge-pr` | `git-merge-pr` | Opens a PR and merges it within the promotion (`wait: true`). A successful promotion confirms the merge. |
| `open-pr` | `git-open-pr` (`title`/`description`/`labels`) | Opens a PR with a title, description and labels; the test fetches the PR and asserts those fields. Leaves the PR open. |
| `wait-for-pr` | `git-wait-for-pr` | Opens a PR and blocks until it is merged; the test merges it via the API. |
| `tag` | `git-tag` | Creates an annotated tag and publishes it with a `git-push` (`tag:`). |
| `checkout-commits` | `git-clone` (multi-checkout) | Creates two commits, pushes them, then a final `git-clone` checks out each individual commit into a different path via `checkout[].as`; the test compares the exposed `commits` map to the created commits. |
| `github-push` | `github-push` | Pushes a commit via the GitHub API to a generated branch; the test asserts the remote branch head equals the pushed commit. See the FIXME in the test about the (unimplemented) push integration policy setup. |

All stages share a task that clones the warehouse commit and copies the demo
manifests, so the suite also covers `git-clone`, `git-clear`, `copy`,
`git-commit`, `git-push`, `git-open-pr` and `compose-output`.

## Covered steps and config keys

The steps this suite exercises and the config keys it uses (the `as` step alias
is omitted -- it is not step config):

| Step | Config keys covered |
| --- | --- |
| `git-clone` | `repoURL`, `checkout[].commit`, `checkout[].branch`, `checkout[].create`, `checkout[].path`, `checkout[].as` |
| `git-clear` | `path` |
| `copy` | `inPath`, `outPath` |
| `git-commit` | `path`, `message` |
| `git-push` | `path`, `targetBranch`, `generateTargetBranch`, `tag` |
| `git-open-pr` | `repoURL`, `sourceBranch`, `targetBranch`, `createTargetBranch`, `title`, `description`, `labels` |
| `git-merge-pr` | `repoURL`, `prNumber`, `wait` |
| `git-wait-for-pr` | `repoURL`, `prNumber` |
| `git-tag` | `path`, `tag`, `message` |
| `github-push` | `path`, `generateTargetBranch` |
| `compose-output` | (arbitrary output fields) |

Not exercised by this suite (per the reference docs): the `provider` and
`insecureSkipTLSVerify` options common to the provider-backed steps
(`git-open-pr`, `git-merge-pr`, `git-wait-for-pr`, `github-push`); `author`
(`git-clone`, `git-commit`); `blobless`/`recurseSubmodules` and
`checkout[].tag`/`checkout[].sparse` (`git-clone`); `maxAttempts`/`force`
(`git-push`, `github-push`); `mergeMethod`/`pollInterval` (`git-merge-pr`);
`pollInterval` (`git-wait-for-pr`); and `force` (`git-tag`).

## Required environment context

This suite reads the following from the `context` section of the env file
passed with `-env-file` (see [`../../envs`](../../envs)):

| Variable | Description |
| --- | --- |
| `kargo_demo_gitops_repo` | HTTPS URL of a fork of the `kargo-demo-gitops` repository. Substituted into the fixtures at runtime (the git credentials Secret, the promotion's `gitRepo` var, and the Warehouse git subscription). |
| `git_pat` | GitHub personal access token with **write** access to that fork. The promotions push branches, open and merge pull requests, and push tags. |

Example:

```yaml
context:
  kargo_demo_gitops_repo: https://github.com/<you>/kargo-demo-gitops.git
  git_pat: <github-personal-access-token>
```

## Note

Each promotion creates per-promotion branches (and, for the `tag` stage, a tag
named after the promotion) on the fork and does not delete them, consistent with
the other git-driven suites.
