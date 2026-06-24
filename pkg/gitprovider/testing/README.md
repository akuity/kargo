# Git provider integration tests

The `gitprovider` packages include integration tests that exercise
`CreatePullRequest` / `MergePullRequest` against a **real** Git hosting provider.
They are build-tagged and **never run in CI** — there is no Make target that
builds the `integration` tag. Run them manually when changing provider behavior.

Each provider has its own `pr_integration_test.go` (build tag
`integration && <provider>`) backed by the shared helpers in this package
(`helpers.go`, build tag `integration`).

## Required environment variables

Every test skips (does not fail) when its variables are unset. Point them at a
**disposable** repository — the tests create branches, commits, and PRs.

| Provider | Variables |
|----------|-----------|
| GitHub   | `TEST_GITHUB_REPO_URL`, `TEST_GITHUB_TOKEN`, `TEST_GITHUB_USERNAME` |
| GitLab   | `TEST_GITLAB_REPO_URL`, `TEST_GITLAB_TOKEN`, `TEST_GITLAB_USERNAME` |
| Gitea    | `TEST_GITEA_REPO_URL`, `TEST_GITEA_TOKEN`, `TEST_GITEA_USERNAME` |
| Azure    | `TEST_AZURE_REPO_URL`, `TEST_AZURE_TOKEN`, `TEST_AZURE_USERNAME` |

`*_TOKEN` must grant push access and PR create/merge. For GitHub, a classic PAT
with the `repo` scope works (`gh auth token` if it has that scope).

### GitHub `TestMergeGate` extras

`TestMergeGate` verifies the `mergeable_state`-aware merge gate. Its `clean` and
`dirty` subtests need only the three GitHub variables above. The `behind` subtest
is **opt-in** and additionally requires:

- `TEST_GITHUB_REQUIRE_UP_TO_DATE=true`, and
- the repo's `main` branch protected with **"Require branches to be up to date
  before merging"** (otherwise GitHub merges out-of-date branches and the state
  never becomes `behind`):

  ```bash
  gh api -X PUT repos/<owner>/<repo>/branches/main/protection --input - <<'JSON'
  {"required_status_checks":{"strict":true,"contexts":[]},
   "enforce_admins":true,"required_pull_request_reviews":null,"restrictions":null}
  JSON
  ```

  Branch protection is free on **public** repos; on private repos it needs
  GitHub Pro. The subtest skips cleanly when not configured.

## Running (in the dev container)

These tests assume the Kargo container environment. The shared helpers shell out
to `git` through Kargo's git client, which sets
`GIT_ASKPASS=/usr/local/bin/credential-helper` — a binary that exists in the
Kargo image but is not part of the `dev-tools` image and is not present on a
developer host. The container also avoids the macOS keychain credential helper,
which otherwise hijacks authentication and can hang non-interactive runs.

Run inside the `dev-tools` container, building the credential helper to its
expected path first:

```bash
make hack-build-dev-tools   # once, builds kargo:dev-tools

docker run --rm -u root \
  -v "$PWD":/workspaces/kargo -w /workspaces/kargo \
  -e TEST_GITHUB_REPO_URL=https://github.com/<owner>/<repo> \
  -e TEST_GITHUB_TOKEN="$(gh auth token)" \
  -e TEST_GITHUB_USERNAME=<owner> \
  kargo:dev-tools bash -c '
    go build -o /usr/local/bin/credential-helper ./cmd/credential-helper &&
    go test -v -tags "integration github" \
      -run TestMergeGate ./pkg/gitprovider/github/'
```

Swap the tag and variables for other providers (e.g. `-tags "integration gitlab"`
with the `TEST_GITLAB_*` variables).
