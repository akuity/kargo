# Advisory draft format

The draft file has two parts: a notes block for the maintainer (never published) and the advisory text (published as-is). Paragraphs are single lines. Format-check against published examples: GHSA-f72x-6fm6-94rq, GHSA-5vvm-67pj-72g4, GHSA-7g9x-cp9g-92mr, GHSA-j94x-8wcp-x7hm, GHSA-g7gw-m874-7rmf (`gh api repos/akuity/kargo/security-advisories/<id> --jq .description`).

## Notes block

A blockquote at the top of the file, followed by `---`. Required items:

- **Proposed title** -- `<Weakness> Vulnerability in <Affected Surface>` (e.g. "Missing Authorization Vulnerability in Resource Refresh API Endpoints", "SSRF in Promotion http/http-download Steps ...", "Open Redirect in UI OIDC Login Flow ...")
- **CWE** -- one ID
- **CVSS v4** -- vector, score, rating; reporter's score if any; which metrics were judgment calls and whether alternatives change the rating
- **Affected versions** -- range with introducing PR/issue and the evidence used
- **Omitted deliberately** -- what from the original was left out and why

## Advisory text

```markdown
## Summary

<What the feature is, in one or two sentences, for a reader who does not know it.>

<What the code does wrong, and, where known, how it came to be that way (intended change vs. shipped change).>

<Only when the vulnerable surface is a discrete, enumerable set (API endpoints, promotion steps, CLI commands, webhook receivers), list it:>

The affected <endpoints|steps|...> are:

1. `...`
2. ...

<Otherwise name the surface in prose: a component, a flow (e.g. UI login), a library path.>

<Who can exploit it and what they can and cannot do. State the bounds as plainly as the exposure.>

## Base Metrics

The following sections provide the rationale for the values selected for each of CVSS v4's base metrics.

### Attack Vector (AV): <value>
### Attack Complexity (AC): <value>
### Attack Requirements (AT): <value>
### Privileges Required (PR): <value>
### User Interaction (UI): <value>
### Confidentiality Impact to Vulnerable System (VC): <value>
### Integrity Impact to Vulnerable System (VI): <value>
### Availability Impact to Vulnerable System (VA): <value>
### Confidentiality Impact to Subsequent Systems (SC): <value>
### Integrity Impact to Subsequent Systems (SI): <value>
### Availability Impact to Subsequent Systems (SA): <value>

## Mitigating Factors

- <Precondition the attacker must satisfy.>
- <What the attacker cannot do.>
- <Observability / audit trail, only if a cited line of code establishes it.>
- There is no evidence of exploitation in the wild.
```

Any list of affected surfaces names released surfaces only. When a surface does not exist in every affected version, append its range in parentheses: `` 6. The legacy `RefreshResource` RPC (v1.9.0 and later) ``. Verify each range against the tag walk; do not assume a surface spans the whole affected range.

Each metric heading is followed by one short paragraph justifying the value. A `None` still gets a sentence saying why.

If the computed score understates real-world risk (CVSS v4 ignores multi-step attacks), say so in the Summary or a paragraph after the Base Metrics intro and give a qualitative rating with the reason -- see GHSA-g7gw-m874-7rmf.

## What stays out of the published text

- PoC scripts, exact request sequences, timing/loop demonstrations
- Recipes for enumeration or reconnaissance
- Fix designs, code suggestions, structural refactoring advice
- Comparisons with, or rebuttals of, other advisories or the reporter's claims
- Reporter's rhetoric ("critically", "trivially", "the maintainer should")
- Hard line wraps
