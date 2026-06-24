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

## Running

```bash
export TEST_GITHUB_REPO_URL=https://github.com/<owner>/<repo>
export TEST_GITHUB_TOKEN=$(gh auth token)
export TEST_GITHUB_USERNAME=<owner>

go test -v -tags 'integration github' \
  -run TestMergeGate ./pkg/gitprovider/github/
```

Swap the tag and variables for other providers (e.g. `-tags 'integration gitlab'`).

## Host setup (running outside the dev container)

The shared helpers shell out to `git` through Kargo's git client, which is built
for the Kargo container. Two things must be handled on a developer host:

1. **Credential helper binary.** Kargo's git client sets
   `GIT_ASKPASS=/usr/local/bin/credential-helper`. That binary ships in the
   container but not on a host. Build and install it once:

   ```bash
   go build -o /tmp/credential-helper ./cmd/credential-helper
   sudo install -m 0755 /tmp/credential-helper /usr/local/bin/credential-helper
   ```

2. **macOS keychain credential helper.** Apple's `/usr/bin/git` enables
   `credential.helper=osxkeychain` via a system gitconfig, which can pop a GUI
   password prompt and hang non-interactive runs. Shim `git` with a wrapper that
   disables the system config, and put it first on `PATH`:

   ```bash
   mkdir -p /tmp/gitwrap
   cat > /tmp/gitwrap/git <<'EOF'
   #!/bin/sh
   export GIT_CONFIG_NOSYSTEM=1
   export GIT_TERMINAL_PROMPT=0
   exec /usr/bin/git "$@"
   EOF
   chmod +x /tmp/gitwrap/git
   export PATH=/tmp/gitwrap:$PATH
   ```

Neither step is needed when running the tests inside the Kargo dev container.
