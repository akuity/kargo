# Dispatch policy library

The embedded Rego modules deciding whether a Pending Promotion may be
dispatched. Directory structure matches package layout:

| Module | Package | Responsibility |
|---|---|---|
| `kargo/dispatch/dispatch.rego` | `kargo.dispatch` | Unions all violations; derives the `decision` document the engine queries |
| `kargo/lib/windows/windows.rego` | `kargo.lib.windows` | Holds forward promotions outside promotion windows |
| `kargo/lib/freezes/freezes.rego` | `kargo.lib.freezes` | Holds promotions during system-wide freezes |
| `kargo/lib/ratelimit/ratelimit.rego` | `kargo.lib.ratelimit` | Rolling-window rate limit on automatic dispatch |
| `kargo/lib/ordering/ordering.rego` | `kargo.lib.ordering` | Class priority, Stage-monotonicity guards, auto-promotion holds, and per-Promotion scheduling |
| `kargo/lib/lib.rego` | `kargo.lib` | Building blocks for custom policies (`kargo.is_forward`, `kargo.is_semver_patch`, `kargo.advances`/`kargo.regresses`) |
| `kargo/project/project.rego` | `kargo.project` | Extension-point defaults for the project custom policy |
| `kargo/cluster/cluster.rego` | `kargo.cluster` | Extension-point defaults for the cluster custom policy |

## Ordering

Because the gate may dispatch a permitted Promotion from deeper in the queue,
`kargo.lib.ordering` enforces the invariants that keep out-of-order dispatch
correct, composed into the built-in decision (not only available to custom
policies):

- **Class priority** (`rollback ≻ manual-forward ≻ auto-forward`) — a forward
  candidate yields to a queued `rollback` (`yield-to-rollback`); an
  auto-forward additionally yields to any queued `manual-forward`
  (`yield-to-manual`). Both re-check shortly (a 5s requeue).
- **Stage monotonicity** — an auto-forward that would not strictly advance the
  Stage is stale (`regression`); a manual-forward whose Freight is strictly
  older than the origin's current Freight is held to await an operator decision
  (`would-regress` — re-issue as a rollback if the regression is intended). A
  re-promote of the current Freight is neither.
- **Auto-promotion holds** — an auto-forward for an origin with a committed
  hold (`data.autoPromotionHolds`) is denied (`auto-hold`) until an operator
  resumes. This rule is unconditional: a custom policy may read the holds but
  cannot suppress the deny (violation sets only union).
- **Scheduling** — a promotion carrying a `kargo.akuity.io/promote-after`
  annotation is held until that time, then self-resumes (`scheduled`).

Coalescing of shadowed auto-forwards is intentionally left to grooming, not
done here: `data.queue` carries no origin, so a gate coalesce rule could not
scope per-origin without risking multi-origin starvation.

## Custom policies

Custom policy sources contain **rules only** — the engine prepends the
package declaration and the standard imports before compiling:

```rego
package kargo.project        # kargo.cluster for ClusterConfig

import rego.v1

import data.kargo.lib as kargo
```

There are two custom sources, each composing into — never replacing — the
default policy:

- **ProjectConfig `spec.customPolicy`** → `package kargo.project`
- **ClusterConfig `spec.customPolicy`** → `package kargo.cluster`,
  composed into every project's dispatch decision

Both packages expose the same hook points, inert when unused:

- `violation` — a set of `{rule, msg, requeue?, blocked_by?, until?}`
  objects, unioned with the standard blocks' violations. A numeric `requeue`
  (seconds) participates in the decision's requeue hint. The optional
  `blocked_by` (a queued Promotion name) and `until` (an RFC3339 time the
  hold self-clears) are surfaced verbatim in the decision's structured
  `reasons` and, from there, as annotations on the held Promotion's
  `PromotionBlocked` event — set them when your rule has that context.
- `freeze_bypass(f)` — a predicate consulted by `kargo.lib.freezes`
  for each freeze that would otherwise hold the promotion. The shipped
  modules default it to `false`; a custom policy overrides it.

The aliased import puts the `kargo.lib` building blocks one qualifier
away: `kargo.is_forward`, `kargo.is_semver_patch(old, new)`, and the
Freight-ordering helpers below.

Alongside the per-candidate `input`, a custom policy can read `data.queue`
— the Stage's Promotions awaiting dispatch, in gate order, each `{name,
class, createdAt}`. A policy locates the candidate under evaluation by
`input.promotion.name` and reasons about the rest: yielding to a queued
rollback, or growing conservative under a deep backlog. The queue reports
true depth (it is not capped at the gate's evaluation limit); dispositions
are re-derived by the policy, not fed back from prior evaluations.

### Freight ordering (does this advance or regress the Stage?)

A promotion moves the Stage *forward* or *backward* by the discovery time
of its Freight, per origin. The candidate's discovery time is
`input.freight.discoveredAt`; the Stage's current Freight per origin is
`data.currentFreight` (keyed by origin, each `{name, discoveredAt}`,
resolved by the gate). `kargo.lib` turns these into ready predicates that
mirror the controller's auto-promotion ordering key (`EffectiveDiscoveredAt`,
then name as a tiebreak), so a policy and auto-selection never disagree on
"newer":

- `kargo.freight_newer(a, b)` — Freight `a` was discovered strictly after `b`.
- `kargo.current_freight` — the current Freight for the candidate's origin;
  undefined on a fresh origin (nothing deployed yet).
- `kargo.advances` — the candidate is strictly newer than current (moves
  forward).
- `kargo.regresses` — the candidate is strictly older than current (moves
  backward). A re-promote of the *current* Freight is neither.

All are undefined (hence false) when there is no current Freight for the
origin or a `discoveredAt` is absent, so a policy needs no presence guard —
e.g. hold a promotion that would move the Stage backward:

```rego
violation contains {"rule": "no-regress", "msg": "would move the stage backward"} if {
	kargo.regresses
}
```

The canonical example — an operator-defined hotfix lane through every
freeze, typically a **cluster** custom policy. Hotfix semantics live in
the custom policy itself; the standard library supplies only the semver
building block (`kargo.is_semver_patch`):

```rego
freeze_bypass(f) if is_hotfix

is_hotfix if {
	count(shared_images) > 0
	every pair in shared_images {
		kargo.is_semver_patch(pair.old, pair.new)
	}
}

shared_images := [pair |
	some img in input.freight.images
	some last in input.stage.lastPromotion.freight.images
	img.repoURL == last.repoURL
	pair := {"old": last.tag, "new": img.tag}
]
```

A source that declares its own `package` is rejected (fail closed). Note
that compile-error line numbers are offset by the prepended header.

## Schemas

`schemas/*.json` are JSON Schemas for the documents the engine supplies:

- `input.json` — the per-Promotion input document (`input.promotion`,
  `input.freight`, `input.stage`, `input.project`, `input.applications`,
  `input.now`)
- `windows.json`, `freezes.json`, `scopes.json`, `ratelimit.json`,
  `queue.json`, `currentFreight.json`, `autopromotionholds.json` — the
  `data.windows`, `data.freezes`, `data.scopes`, `data.rateLimit`,
  `data.queue`, `data.currentFreight`, and `data.autoPromotionHolds` documents

Each schema is registered with the compiler as `schema.<basename>`. The
standard library declares what it consumes via `# METADATA ... schemas:`
annotations, so it is type-checked against these schemas at compile time
(see `TestPolicySchemaEnforcement`). Custom sources are not annotated and
therefore not schema-checked.

## Linting

```shell
regal lint pkg/promotion/dispatch/policy
```

`.regal/config.yaml` declares the custom built-ins (`kargo.rrule_active`,
`kargo.rrule_next`, `kargo.rrule_active_end`). Note `opa check` requires a
full capabilities file
including these built-ins; the authoritative compile check is the engine
itself (`go test ./pkg/promotion/dispatch/...`).
