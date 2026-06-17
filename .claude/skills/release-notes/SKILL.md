---
name: release-notes
description: Draft release notes for a new minor version
argument-hint: <new-version>
---

Draft release notes for a new Kargo minor release.

**Argument:** `$0` = new version (e.g. `1.10.0`). The previous minor is
inferred automatically (e.g. `1.9`).

## Setup

Create a working directory for intermediate files:

```bash
mkdir -p /tmp/kargo-rn/metadata
```

## Phase 1 -- Determine the commit range

1. Parse `$0` to derive:
   - `NEW_MINOR` = major.minor (e.g. `1.10`)
   - `PREV_MINOR` = the previous minor (e.g. `1.9`)
2. Ensure local branches are up to date with `upstream`:
   ```bash
   git fetch upstream
   ```
   For each branch needed (`release-$NEW_MINOR` if it exists on upstream,
   otherwise `main`; and `release-$PREV_MINOR`):
   - If the local branch already exists, check it out and pull:
     ```bash
     git checkout <branch> && git pull upstream <branch>
     ```
   - If it doesn't exist locally, create it from upstream:
     ```bash
     git checkout -b <branch> upstream/<branch>
     ```

   The **tip** is `release-$NEW_MINOR` if it exists, otherwise `main`.
   The **base** is the merge-base of the tip with `release-$PREV_MINOR`.
   Collect the full commit list:
   ```bash
   git log --oneline --no-merges <base>..<tip>
   ```

## Phase 2 -- Exclude backported commits

Run the companion script to remove commits that were cherry-picked or
backported to the previous release branch (they already shipped in a patch
release):

```bash
.claude/skills/release-notes/scripts/exclude-backports.sh <base> <tip> release-$PREV_MINOR > /tmp/kargo-rn/commits.txt
```

The script handles automated backports (`chore(backport ...):` prefix,
resolving original PR numbers via the backport PR body), manual backports
(`chore: manually backport #NNNN:`), "Merge commit from fork" entries,
and backport entries on the tip branch. It reports a summary to stderr.

Report the counts to the user.

## Phase 3 -- Preliminary classification

Run the companion script to classify commits by subject line alone (no API
calls needed):

```bash
.claude/skills/release-notes/scripts/classify-by-subject.sh \
  /tmp/kargo-rn/commits.txt \
  /tmp/kargo-rn/included.txt \
  /tmp/kargo-rn/excluded.txt \
  /tmp/kargo-rn/uncertain.txt
```

The script auto-excludes docs, deps, CI, typo, and chore commits;
auto-includes features and breaking changes; and sends the rest to
uncertain. It reports counts to stderr.

Report the counts to the user.

## Phase 4 -- Enrich and classify remaining commits

### GitHub authentication

Before fetching any GitHub data, verify that `gh` is authenticated and has
access to the `akuity/kargo` repository by running a test API call:

```bash
gh api repos/akuity/kargo --jq '.full_name'
```

If this fails for **any** reason -- SAML enforcement, expired token, missing
scopes, no `gh` CLI, etc. -- do **not** silently fall back to local-only
data. Instead:

1. Tell the user exactly what went wrong.
2. Suggest the user try `gh auth refresh` to reauthorize via the browser.
   Re-run the test call after they do.
3. If that doesn't work, ask the user to provide a GitHub personal access
   token authorized for the `akuity` organization. Either kind works:
   - **Fine-grained** (recommended): scoped to `akuity/kargo` with
     read-only repository access. Authorized for the org at creation time.
   - **Classic**: with `repo` scope, then separately authorized for the
     `akuity` org's SAML SSO.
4. Once the user provides a token, set it for the remainder of the session:
   ```bash
   export GH_TOKEN=<token>
   ```
5. Re-run the test call to confirm it works.
6. If it still fails, ask the user to troubleshoot -- do not proceed without
   GitHub API access. PR and issue context is essential for writing quality
   release notes.

### Fetch PR and issue metadata

Run the companion script to fetch PR details and linked issues for all
auto-included and uncertain commits:

```bash
cat /tmp/kargo-rn/included.txt /tmp/kargo-rn/uncertain.txt > /tmp/kargo-rn/needs-fetch.txt
.claude/skills/release-notes/scripts/fetch-pr-metadata.sh \
  /tmp/kargo-rn/needs-fetch.txt \
  /tmp/kargo-rn/metadata
```

The script creates `pr-NNNN.json` and `issue-NNNN.json` files in the
output directory, plus an `index.tsv` summary. It is idempotent (skips
already-fetched PRs).

### Classify unclassified commits

Read the fetched metadata (`/tmp/kargo-rn/metadata/`) and classify the
uncertain commits using PR body, labels, linked issues, and diff stats:

- **Auto-exclude**: minor bug fixes that appear routine (small diff,
  not touching user-facing behavior significantly). Use the PR body and
  linked issue (if any) to judge significance -- a bug fix linked to an
  issue with many reactions or comments may be noteworthy.
  Also exclude: internal refactors with no user-facing change.
- **Auto-include**: PRs labeled `enhancement` / `feature`, or PR bodies
  mentioning `deprecat`
- **Ask about the rest**: present unclassified commits to the user in
  batches of ~10. For each, show:
  - Commit subject
  - PR title (if different from subject) and a one-line summary of the
    PR body
  - Linked issue title (if any)
  - Files changed (short stat)

  Provide a recommended include/exclude for each based on your best
  judgment of the PR and issue context.

## Phase 5 -- Identify first-time contributors

Run the companion script to find first-time contributors and map their
emails to GitHub logins:

```bash
.claude/skills/release-notes/scripts/map-contributors.sh <base> <tip> release-$PREV_MINOR > /tmp/kargo-rn/contributors.txt
```

The script finds all author emails in the range, filters bots, checks for
prior commits before the base, and maps each first-time email to a GitHub
login deterministically (commit SHA → GitHub API). Output is `email|login`
lines. Use **only** this mapping for the contributor list -- do not
cross-reference or override it with PR author logins or other heuristics.

## Phase 6 -- Group and draft

### Read prior release notes for tone

Before writing, re-read `docs/docs/80-release-notes/` for the two most
recent minor versions. Match their tone, structure, emoji usage, and level of
detail.

### Determine the file name

Release notes files use descending numeric prefixes. Read the existing files
in `docs/docs/80-release-notes/` and choose a prefix one less than the
current lowest-numbered version file (e.g. if `90-v1.9.0.md` exists, use
`89-v1.10.0.md`).

### Structure

The release notes file should NOT have YAML frontmatter (matching the pattern
of recent release notes files). Start with a one- or two-sentence intro with
a rocket emoji.

Organize the included changes into sections and subsections that make sense
for the content. Use the prior release notes as a guide for the kinds of
sections and headings that work well, but adapt to fit the actual changes --
don't force content into a rigid template. Always end with a Special Thanks
section for first-time contributors (if any).

Use emoji in section and subsection headings where a sensible one exists.
The tone should be light and positive -- release notes are a celebration of
what shipped.

### Writing guidelines

- **Use PR and issue context for narratives.** The PR description explains
  what and why; the linked issue explains the user's original problem. Lead
  with the user benefit derived from the issue, then describe the solution
  using the PR context. Do not copy PR text verbatim -- distill it into
  concise, user-facing language.
- For breaking changes and deprecations, always state:
  1. What changed or will change
  2. Why (brief rationale -- often found in the PR body)
  3. What to do about it (migration path or link to docs)
  4. When deprecated features will be removed
- Link to relevant documentation sections using `https://docs.kargo.io/...`
  URLs. Find the right doc path by searching `docs/docs/` for the relevant
  content.
- For features, lead with the user benefit, not the implementation detail.
  Keep descriptions concise -- a sentence or two plus a docs link.
- Use `@username` for contributor mentions.
- "Freight" is a mass noun -- never "freights" or "a freight."
- Say "promote" not "deploy" when describing what Kargo does.
- End with a full changelog link:
  ```markdown
  **Full Changelog**: [v<PREV_LATEST>...v$0](https://github.com/akuity/kargo/compare/v<PREV_LATEST>...v$0)
  ```
  where `PREV_LATEST` is the latest tag on `upstream/release-$PREV_MINOR`:
  ```bash
  git tag --list "v$PREV_MINOR.*" --sort=-v:refname | head -1
  ```

### Update the deprecations page

If there are new deprecations or breaking changes involving removals of
previously deprecated features, update
`docs/docs/80-release-notes/100-deprecations.md` to reflect them, following
the existing table format.

## Phase 7 -- Present for review

1. Show the complete draft.
2. Summarize: number of commits analyzed, excluded, included; number of
   PRs and issues consulted; number of first-time contributors found.
3. Do **not** commit automatically -- wait for the user to review, suggest
   edits, and approve.
