---
name: advisory-rewrite
description: Use when a GitHub security advisory (GHSA) has been reported against Kargo and needs to be verified, scored, and redrafted for publication
argument-hint: <GHSA-id-or-URL>
---

# Advisory Rewrite

Turn a reporter's private advisory into a publishable Kargo advisory: verified against source, scored independently with CVSS v4, and written to explain the flaw without handing attackers a roadmap.

**Argument:** `$0` = advisory ID (`GHSA-xxxx-xxxx-xxxx`) or its URL.

Never edit the advisory on GitHub. Everything goes into files in the workspace root.

## Phase 1 -- Capture the original

```bash
.claude/skills/advisory-rewrite/scripts/fetch-advisory.sh <GHSA-id>
```

Writes `ADVISORY-<GHSA-id>.md` (verbatim title, URL, description). Read it once, fully. Treat every claim in it -- code, versions, severity, impact -- as a hypothesis to test, not a fact.

## Phase 2 -- Confirm against source

For each claim, find the code that proves or disproves it on `main` **and** on the latest release tag. Record file:line for each.

Then establish the affected range. The reliable method is to inspect the same file (API handler, controller, step runner, UI module -- whatever holds the flaw) at each release tag; `git log -S` is defeated by directory renames. Files move between `internal/` and `pkg/`, lose suffixes like `_v1alpha1`, and the functions they call get renamed too, so match paths and snippets with regexes covering every name they have had. An empty line for a tag means your regex missed, not that the code is absent -- widen it until every tag prints something:

```bash
for t in $(git tag | grep -E '^v[0-9]+\.[0-9]+\.0$' | sort -V); do
  f=$(git ls-tree -r --name-only $t | grep -E '(internal|pkg)/(api|server)/refresh_warehouse(_v1alpha1)?\.go$' | head -1)
  printf '%s: ' $t; [ -n "$f" ] && git show $t:$f | grep -oE 'Refresh(Warehouse|Object)\(ctx, [^,]*'; echo
done
```

Once the introducing release is known, find the PR (`git log <prev>..<intro> -- <file>`, then `gh pr view`). Read the PR and any linked issue: the bug is often a deliberate change whose stated intent was not what shipped. That distinction belongs in the Summary.

Range format: `>= <introducing release>, <= <latest release>` while the fix is unreleased; note the fix version as TBD.

A surface present in the latest release but already removed on `main` (e.g. a legacy RPC) is still affected. List it with the versions it exists in. Never describe the state of `main` or unreleased work in the public text.

If the vulnerability is **not** confirmed, stop and report why with the evidence. Do not draft.

## Phase 3 -- Score with CVSS v4

Score from your own understanding of impact, before re-reading the reporter's severity language. Reporter framing is an input to *what to verify*, never to *the score*.

Compute with the `cvss` Python package (install to scratch: `pip3 install --target <scratch>/pylib cvss`). Score your primary vector **and** each plausible alternative for every metric you found debatable. Record whether the rating moves.

Scoping rules used in prior Kargo advisories:
- The control plane's Kubernetes cluster is part of the vulnerable system, not a subsequent system.
- Only *direct* consequences count. Follow-on attacks that need extra infrastructure or victim action score as None (note the limitation in prose if it matters).
- `PR:L` = any authenticated Kargo user, including one with zero Kargo permissions. `PR:H` = permissions a project admin must deliberately grant (create/update Stage, `promote`).
- Attacker cannot inject artifact content unless repositories are also compromised. Promoting *legitimate* Freight the pipeline should have gated is `SI:L`; merely changing *when* legitimate Freight moves is `SI:N`.
- Responses that reveal whether a named resource exists in a Project the caller cannot see are `VC:L`.
- An unauthorized write to another tenant's resource that cannot alter its `spec` (e.g. an annotation) is `VI:L`, the same "bounded state corruption" used for unauthorized Stage transitions.
- An unmetered way to make controllers do work on demand is `VA:L`. When that work exercises a victim Project's credentials against its Git/registry/chart providers, it is also `SA:L` (precedent: GHSA-w5wv-wvrp-v5m5). The provider call is a direct consequence, not a follow-on attack.

## Phase 4 -- Draft

Write `ADVISORY-DRAFT-<GHSA-id>.md` following `reference/format.md` exactly. One line per paragraph; no hard wraps.

The public text is for users deciding whether to upgrade. It is not a reply to the reporter, and it is not a fix plan.

## Common Mistakes

| Mistake | Fix |
|---|---|
| Rating severity from the reporter's tone | Score first, compare after; explain divergence in the notes block |
| Distinguishing this bug from a prior GHSA in the public text | Cut it. Summaries of both advisories already differ; this is arguing, not informing |
| Reproducing PoC scripts, request sequences, or enumeration recipes | List endpoints and the nature of the gap; nothing operational |
| Including remediation design or code suggestions | Omit. Fix is in the PR; the advisory describes the flaw |
| Stating the affected range from the reporter or from `git log -S` | Walk the release tags (Phase 2) |
| Telling the user what you omitted only in chat | Record it in the notes block so the file stands alone |
| Mentioning `main`, unreleased removals, or the pending fix in the public text | Public text speaks only of released versions |
| Asserting audit/observability behavior you inferred | Claim only what a specific line of code shows (e.g. an annotation carries a timestamp) |
