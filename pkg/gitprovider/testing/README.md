# Git provider integration tests

The `gitprovider` packages include integration tests that exercise
`CreatePullRequest` / `MergePullRequest` against a **real** Git hosting provider.
They are build-tagged and **never run in CI** — there is no Make target that
builds the `integration` tag. Run them manually when changing provider behavior.

Each provider has its own `pr_integration_test.go` (build tag
`integration && <provider>`) backed by the shared helpers in this package
(`helpers.go`, build tag `integration`).

## What is covered

GitHub, GitLab, and Gitea each have two live tests; Azure has only the first,
since it does not implement `DeleteBranch`:

- `TestCreateAndMergePullRequest` opens a PR, merges it with each supported
  merge method, and checks the merge commit's parent count.
- `TestDeleteBranchLive` pushes a branch, opens a PR from it, asserts the
  provider reports that branch as the PR's head, deletes it through
  `DeleteBranch`, confirms it is gone from the remote, and then deletes it
  again to check that a missing branch is treated as success. Providers report
  a missing branch in very different ways (GitHub 422, GitLab a sentinel error,
  Gitea 500 on delete but 404 on lookup), so this last step is the one most
  worth running against a real server.

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
`dirty` subtests need only the three GitHub variables above. The `blocked`
subtest is **opt-in** and additionally requires:

- `TEST_GITHUB_REQUIRE_STATUS_CHECK=true`, and the repo's `main` branch protected with a **required
  status check the PR will never satisfy**. With an unsatisfied required check, GitHub reports the
  PR's `mergeable_state` as `blocked` — the same gate branch as `behind`, where the merge is
  attempted and GitHub decides. `enforce_admins` below matters: the subtest expects GitHub to
  decline, so `TEST_GITHUB_TOKEN` must not be authorized to bypass the protection.

  ```bash
  gh api -X PUT repos/<owner>/<repo>/branches/main/protection --input - <<'JSON'
  {"required_status_checks":{"strict":true,"contexts":["kargo-required-check"]},
   "enforce_admins":true,"required_pull_request_reviews":null,"restrictions":null}
  JSON
  ```

  Branch protection needs a **public** repo or GitHub Pro. Because a required
  check is repo-global, it blocks *every* PR — so this mode is mutually exclusive
  with the `clean`/`dirty` subtests, which skip when
  `TEST_GITHUB_REQUIRE_STATUS_CHECK=true`. Run the two modes separately. Each
  subtest skips cleanly when its mode is not selected.

  (`behind` itself is not exercised live: it additionally requires a *passing*
  required check on an out-of-date branch, i.e. publishing commit statuses. It
  takes the same gate path as `blocked` and is covered by the unit tests.)

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

## Self-hosted providers (no external account needed)

Gitea and GitLab can both run locally in Docker, which makes them the easiest
way to get real coverage without credentials for a hosted service. Plain
`http://` repository URLs are supported by the helpers for exactly this case.

Put the provider and the test container on one Docker network so the test can
reach the provider by container name. The provider's hostname must satisfy the
registration predicate (contain `gitea` or `gitlab`), so name the container
accordingly.

Gitea, for example:

```bash
docker network create kargo-gp-test
docker run -d --name gitea-test --network kargo-gp-test -p 3300:3000 \
  -e GITEA__security__INSTALL_LOCK=true gitea/gitea:1.24
docker exec -u git gitea-test gitea admin user create --admin \
  --username kargotest --password 'Kargo-Test-1234' \
  --email kargotest@example.com --must-change-password=false
# Then create a token (scopes write:repository, write:user) and a repo through
# the API at http://localhost:3300/api/v1, and run the tests with
#   --network kargo-gp-test
#   TEST_GITEA_REPO_URL=http://gitea-test:3000/kargotest/<repo>
```

GitLab CE works the same way with `gitlab/gitlab-ce`, but boots slowly and
wants several GB of memory. Create a root personal access token with
`gitlab-rails runner` and a project through `/api/v4/projects`, then point
`TEST_GITLAB_REPO_URL` at `http://gitlab-test/root/<project>`.

Azure DevOps has no self-hosted equivalent with the same API, so its test needs
a real organization and token. Bitbucket Cloud has no live test at all.
