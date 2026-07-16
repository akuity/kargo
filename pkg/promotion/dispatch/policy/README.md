# Dispatch policy library

The embedded Rego modules deciding whether a Pending Promotion may be
dispatched. Directory structure matches package layout:

| Module | Package | Responsibility |
|---|---|---|
| `kargo/dispatch/dispatch.rego` | `kargo.dispatch` | Unions all violations; derives the `decision` document the engine queries |
| `kargo/lib/windows/windows.rego` | `kargo.lib.windows` | Holds forward promotions outside promotion windows |
| `kargo/lib/exclusions/exclusions.rego` | `kargo.lib.exclusions` | Holds promotions during system-wide exclusions |
| `kargo/lib/ratelimit/ratelimit.rego` | `kargo.lib.ratelimit` | Rolling-window rate limit on automatic dispatch |
| `kargo/lib/helpers/helpers.rego` | `kargo.lib.helpers` | Building blocks for custom policies (e.g. `is_hotfix`) |

## Custom project policies

A project extends the policy via ProjectConfig `spec.policy.custom`: a
single Rego module that **must** declare `package kargo.custom`. It composes
into — never replaces — the default policy through two hook points, both
inert when absent:

- `violation` — a set of `{rule, msg, requeue?}` objects, unioned into the
  decision alongside the standard blocks. A numeric `requeue` (seconds)
  participates in the decision's `requeue_after`.
- `exclusions_bypass` — a set of exclusion **names**; `kargo.lib.exclusions`
  raises no violation for a bypassed exclusion.

The canonical example — hotfixes bypass every exclusion:

```rego
package kargo.custom

import rego.v1

import data.kargo.lib.helpers

exclusions_bypass contains e.name if {
	some e in data.exclusions
	helpers.is_hotfix
}
```

## Schemas

`schemas/*.json` are JSON Schemas for the documents the engine supplies:

- `input.json` — the per-Promotion input document (`input.promotion`,
  `input.freight`, `input.stage`, `input.project`, `input.applications`,
  `input.now`)
- `windows.json`, `exclusions.json`, `scopes.json`, `ratelimit.json` — the
  `data.windows`, `data.exclusions`, `data.scopes`, and `data.rateLimit`
  documents

Each schema is registered with the compiler as `schema.<basename>`. The
standard library declares what it consumes via `# METADATA ... schemas:`
annotations, so it is type-checked against these schemas at compile time
(see `TestPolicySchemaEnforcement`). Custom modules are not annotated and
therefore not schema-checked, but may opt in with the same annotations.

## Linting

```shell
regal lint pkg/promotion/dispatch/policy
```

`.regal/config.yaml` declares the custom built-ins (`kargo.rrule_active`,
`kargo.rrule_next`). Note `opa check` requires a full capabilities file
including these built-ins; the authoritative compile check is the engine
itself (`go test ./pkg/promotion/dispatch/...`).
