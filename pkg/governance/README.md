# Governance Bot

A GitHub App that enforces contribution-policy rules on issues and pull
requests according to a YAML config kept in the target repository.

The bot is config-driven: adding a new blocking label, slash command, or
canned reply is a YAML change. Adding a new *kind* of action (a new
primitive verb) requires code.

## Overview

The bot is a Go HTTP handler that processes GitHub webhooks. It can be run
as a long-lived HTTP server or behind an AWS Lambda Function URL. On each
event it:

1. Validates the webhook signature.
2. Loads `.github/governance.yaml` from the target repo's default branch
   (no caching — every event re-reads the file, so config changes take
   effect immediately on the next event).
3. Dispatches the event to the appropriate handler.

Webhook events handled:

| Event | Action(s) | Behavior |
|---|---|---|
| `issues` | `opened` | Required-label enforcement on the issue. |
| `pull_request` | `opened` | Auto-assign, label inheritance, required-label enforcement, policy evaluation. |
| `pull_request` | `reopened`, `ready_for_review` | Policy evaluation only (labeling is not re-run). |
| `issue_comment` | `created` | Slash-command dispatch (maintainer-only). |
| `ping` | — | Returns 200. |

All other events are silently ignored with HTTP 204.

## Configuration

The config file is `.github/governance.yaml` in the target repo's default
branch. Top-level shape:

```yaml
maintainerAssociations: [OWNER, MEMBER]
issues:
  requiredLabelPrefixes: [...]
  slashCommands: { ... }
pullRequests:
  exemptions: { ... }
  onNoLinkedIssue: { actions: [...] }
  onBlockedIssue: { blockingLabels: [...], actions: [...] }
  onPass: { actions: [...] }
  inheritedLabelPrefixes: [...]
  requiredLabelPrefixes: [...]
  slashCommands: { ... }
```

### `maintainerAssociations`

GitHub author-association values that count as "maintainer" for the
purposes of slash commands and exemptions. Valid values are
[GitHub's standard associations](https://docs.github.com/en/graphql/reference/enums#commentauthorassociation):
`OWNER`, `MEMBER`, `COLLABORATOR`, `CONTRIBUTOR`, `FIRST_TIME_CONTRIBUTOR`,
`FIRST_TIMER`, `MANNEQUIN`, `NONE`.

If unset or empty, no one is treated as a maintainer.

### `issues`

#### `requiredLabelPrefixes`

List of label prefixes. For every prefix listed, at least one label
matching `<prefix>/...` must exist on the issue. For any missing prefix, a
`needs/<prefix>` label is added when the issue is opened.

Example: with `[area, kind, priority]`, an issue lacking any
`priority/*` label gets `needs/priority` added.

#### `slashCommands`

Map of command name (without the leading `/`) to a command definition.
See [Slash Commands](#slash-commands) below.

### `pullRequests`

#### `exemptions`

Criteria under which a PR is exempt from policy enforcement. **Any one
matching criterion exempts the PR (OR semantics).** Slash-command-driven
policy evaluation also consults this — a maintainer typing `/policy`
won't force-evaluate an exempt PR.

| Field | Type | Description |
|---|---|---|
| `maintainers` | bool | If true, PRs authored by a maintainer (per `maintainerAssociations`) are exempt. |
| `bots` | bool | If true, PRs whose author login ends in `[bot]` are exempt. |
| `maxChangedLines` | int | If > 0, PRs whose total `additions + deletions` ≤ this value are exempt. Disabled when 0 or unset. |
| `pathPatterns` | []string | Gitignore-style patterns. A PR is exempt if every file it changes matches at least one pattern. Empty/unset disables the check. |

The path check requires fetching the PR's changed files and is therefore
the most expensive criterion; it's evaluated last and only when the
cheaper checks haven't already exempted the PR. PRs touching more than
~100 files are treated as not path-exempt without paginating — they're
not drive-by changes by definition.

Pattern syntax follows full gitignore semantics (negation, anchored vs.
unanchored, trailing-slash directory shorthand). Example:

```yaml
exemptions:
  maxChangedLines: 5
  pathPatterns:
  - '**/*.md'        # any markdown
  - 'docs/'          # gitignore "everything under docs"
  - '!docs/internal' # but not under docs/internal
```

#### `onNoLinkedIssue`

Actions to run when a PR has no linked issue *and* the PR isn't exempt.
A "linked issue" is parsed from the PR body using GitHub's closing-keyword
syntax (`closes`, `fixes`, `resolves`, with their tense variants and the
URL form):

```
Closes #123
Fixes https://github.com/owner/repo/issues/456
```

If `onNoLinkedIssue` is not configured, a no-linked-issue PR is treated
as passing (i.e. `onPass` runs if configured).

#### `onBlockedIssue`

Actions to run when a PR is linked to an issue that carries one or more
labels listed in `blockingLabels`. Use case: the issue is still in
discussion, gated for maintainer-only work, etc.

```yaml
onBlockedIssue:
  blockingLabels:
  - kind/proposal
  - needs discussion
  actions:
  - addLabels: [policy/blocked-issue]
  - comment: |
      Linked issue (#{{.IssueNumber}}) carries: {{.BlockingLabels}}.
  - convertToDraft: true
```

If `onBlockedIssue` is not configured, blocked-issue PRs are treated as
passing.

#### `onPass`

Actions to run when neither `onNoLinkedIssue` nor `onBlockedIssue`
fired — including for exempt PRs. **`onPass` is the cleanup hook**:
operators typically remove labels added by prior failing evaluations.

Behavior matrix:

| State | onNoLinkedIssue | onBlockedIssue | onPass |
|---|---|---|---|
| Exempt | — | — | ✓ |
| Not exempt, no linked issue | ✓ | — | — |
| Not exempt, blocked issue | — | ✓ | — |
| Not exempt, passing | — | — | ✓ |

Example cleanup:

```yaml
onPass:
  actions:
  - removeLabels:
    - policy/no-linked-issue
    - policy/blocked-issue
```

Idempotent — `removeLabels` for a label not present is a no-op (404 from
GitHub is swallowed).

#### `inheritedLabelPrefixes`

When a PR is opened with a linked issue, copy any of the issue's labels
whose name starts with one of these prefixes. Useful for inheriting
`kind/*`, `area/*`, etc. from the issue without the author having to set
them again.

#### `requiredLabelPrefixes`

Same as the `issues.requiredLabelPrefixes` rule but applied to PRs. The
inherited labels count toward satisfying the requirement.

#### `slashCommands`

Map of command name to definition. See [Slash Commands](#slash-commands).

## Actions

An `action` is a primitive operation the bot can perform. Multiple actions
in a list run **in order**, fail-fast (a failed action short-circuits the
list — operator-authored ordering is intentional).

| Field | Type | Applies to | Notes |
|---|---|---|---|
| `addLabels` | []string | issues, PRs | Idempotent at GitHub. |
| `removeLabels` | []string | issues, PRs | 404 (label not present) is swallowed. |
| `comment` | string | issues, PRs | Template — see below. |
| `close` | bool | issues, PRs | Issues close with `state_reason: not_planned`. |
| `convertToDraft` | bool | PRs only | No-op if the PR is already a draft or closed. Implemented via the `convertPullRequestToDraft` GraphQL mutation. |
| `applyPRPolicy` | bool | PRs only | Re-runs policy evaluation. Honors exemptions. |

### Comment templates

Comments use Go's `text/template` syntax. Variables vary by context:

| Context | Variables |
|---|---|
| Slash command | `.Arg`, `.RepoFullName` |
| `onNoLinkedIssue` | (none) |
| `onBlockedIssue` | `.IssueNumber`, `.BlockingLabels` |

Example:

```yaml
comment: |
  Closing as a duplicate of #{{.Arg}}. See
  https://github.com/{{.RepoFullName}}/issues/{{.Arg}}.
```

## Slash Commands

A slash command is a comment whose first non-whitespace characters on a
line are `/`-prefixed. Multiple commands in a single comment run in
order. Indented commands (e.g. inside quoted text) work too.

```
Hello!

/discuss
/research
```

The above runs `/discuss` then `/research` in order.

**Maintainer-only.** Commands posted by users not in
`maintainerAssociations` are silently ignored.

**Context-aware.** A comment on an issue dispatches against
`issues.slashCommands`; a comment on a PR dispatches against
`pullRequests.slashCommands`. Unknown commands are silently ignored.

### Command definition

```yaml
slashCommands:
  duplicate:
    description: "Close as a duplicate"
    requiresArg: true
    actions:
    - addLabels: [duplicate]
    - comment: "Duplicate of #{{.Arg}}."
    - close: true
```

| Field | Type | Description |
|---|---|---|
| `description` | string | Used by `/help`. |
| `requiresArg` | bool | If true, the command is silently ignored when no argument follows it. The argument is the second whitespace-separated token (with a leading `#` stripped if present). |
| `actions` | []action | The work to do. |

### `/help`

`/help` is built-in and not configurable. It posts a Markdown table of
available commands for the current context (issue or PR), pulling
descriptions from each command's `description` field.

### `/policy` (a convention, not a built-in)

There's no built-in `/policy` command, but the `applyPRPolicy: true`
action makes one trivial:

```yaml
pullRequests:
  slashCommands:
    policy:
      description: "Re-evaluate PR policy"
      actions:
      - applyPRPolicy: true
```

A maintainer typing `/policy` re-runs the same evaluation that fires on
`opened`/`reopened`/`ready_for_review`. Useful when labels on the linked
issue have changed and you want the bot to act on the new state without
waiting for the next webhook event.

## PR Lifecycle

When a PR is **opened**, the bot runs in this order:

1. Auto-assign the PR to its author.
2. Inherit labels from the linked issue (per `inheritedLabelPrefixes`).
3. Enforce required-label prefixes (per `requiredLabelPrefixes`).
4. Apply PR policy.

Steps 1–3 are independent: a failure in one is logged and accumulated but
does not prevent the others from running. Step 4 is the only step that
honors `exemptions`.

When a PR is **reopened** or marked **ready for review**, only step 4
runs. Labeling is considered a one-shot operation at open time.

## GitHub App Permissions

The bot needs the following from its installation. Some operations
require non-obvious permission combinations — flagged inline.

| Permission | Level | What for |
|---|---|---|
| Metadata | Read | Required by all GitHub Apps. |
| Issues | Read & write | Add/remove labels, comment, close issues. (Labels and comments on PRs also route through the Issues API.) |
| Pull requests | Read & write | Close PRs, fetch PR file lists for path-based exemptions, mark draft. |
| Contents | Read & write | **Required** for the `convertPullRequestToDraft` GraphQL mutation in addition to `pull_requests:write`. GitHub doesn't document this, but it's empirically required. |

Webhook event subscriptions:

- Issues
- Issue comment
- Pull request

A signing secret must be configured on the App and provided to the bot
via the `GITHUB_WEBHOOK_SECRET` env var.

## Deployment

The bot is a single binary at `cmd/governance-bot/main.go`. It runs in
two modes, switched on by the presence of `AWS_LAMBDA_RUNTIME_API`:

- **HTTP server** (default) — listens on `:$PORT` (default `8080`).
- **AWS Lambda** — wraps the same handler with the Lambda HTTP adapter.

### Required env vars

| Var | Description |
|---|---|
| `GITHUB_APP_CLIENT_ID` | Client ID of the GitHub App. |
| `GITHUB_APP_PRIVATE_KEY` | PEM-encoded private key, **base64-encoded** before being placed in the env. |
| `GITHUB_WEBHOOK_SECRET` | Webhook signing secret. |

Optional:

| Var | Default | Description |
|---|---|---|
| `PORT` | `8080` | HTTP listen port (HTTP-server mode only). |
| `LOG_LEVEL` | `info` | One of `trace`, `debug`, `info`, `error`, `discard`. |

### Building

```bash
# Local HTTP binary
make build-governance-bot

# Lambda binary (linux/arm64, stripped)
make build-governance-bot-lambda

# Lambda zip
make package-governance-bot-lambda
```

### Lambda redeploy

```bash
make package-governance-bot-lambda
aws lambda update-function-code \
  --function-name <FUNCTION_NAME> \
  --zip-file fileb://bin/governance-bot-lambda/governance-bot.zip \
  --region <REGION> --publish
```

`update-function-code` only replaces the code; env vars and other
function configuration are preserved.

### Local development

Run the binary, point a tunneling tool (e.g. ngrok) at it, and set the
GitHub App's webhook URL to the tunneled address. The same
`.github/governance.yaml` applies.

## Operational Notes

- **GitHub does NOT auto-retry failed webhook deliveries.** A non-2xx
  response is logged in the App's delivery UI as a failed delivery, but
  the event is not redelivered automatically. The operator can manually
  redeliver from the UI within 3 days, or run a script against the
  `/app/hook/deliveries` endpoint.
- **Each handler accumulates errors** rather than short-circuiting. If
  e.g. label inheritance fails, the bot still runs required-label
  enforcement and policy. The aggregated error propagates to the HTTP
  response (→ 500 → red in the delivery UI) so failures are visible.
- **Inside `executeActions`, behavior is fail-fast.** Operator-authored
  action sequences have intentional ordering (`comment`-then-`close`,
  etc.) so a failed action short-circuits subsequent actions in the
  same list.
- **Comments are not idempotent** — a duplicate event delivery (e.g.
  manual redelivery) will post a duplicate comment. With auto-retry
  off this is uncommon, but worth knowing.
- **Config is read fresh on every event.** No restart needed when
  changing `.github/governance.yaml`.
