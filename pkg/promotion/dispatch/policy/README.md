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
| `kargo/project/project.rego` | `kargo.project` | Extension-point defaults for the project custom policy |
| `kargo/cluster/cluster.rego` | `kargo.cluster` | Extension-point defaults for the cluster custom policy |

## Custom policies

Custom policy sources contain **rules only** — the engine prepends the
package declaration and the standard imports before compiling:

```rego
package kargo.project        # kargo.cluster for ClusterConfig

import rego.v1

import data.kargo.lib.exclusions
import data.kargo.lib.helpers
import data.kargo.lib.ratelimit
import data.kargo.lib.windows
```

There are two custom sources, each composing into — never replacing — the
default policy:

- **ProjectConfig `spec.customPolicy`** → `package kargo.project`
- **ClusterConfig `spec.customPolicy`** → `package kargo.cluster`,
  composed into every project's dispatch decision

Both packages expose the same hook points, inert when unused:

- `violation` — a set of `{rule, msg, requeue?}` objects, unioned with the
  standard blocks' violations. A numeric `requeue` (seconds) participates
  in the decision's requeue hint.
- `exclusions_bypass(e)` — a predicate consulted by `kargo.lib.exclusions`
  for each exclusion that would otherwise hold the promotion. The shipped
  modules default it to `false`; a custom policy overrides it.

The canonical example — hotfixes bypass every exclusion — is one line:

```rego
exclusions_bypass(e) if helpers.is_hotfix
```

A source that declares its own `package` is rejected (fail closed). Note
that compile-error line numbers are offset by the prepended header.

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
(see `TestPolicySchemaEnforcement`). Custom sources are not annotated and
therefore not schema-checked.

## Linting

```shell
regal lint pkg/promotion/dispatch/policy
```

`.regal/config.yaml` declares the custom built-ins (`kargo.rrule_active`,
`kargo.rrule_next`). Note `opa check` requires a full capabilities file
including these built-ins; the authoritative compile check is the engine
itself (`go test ./pkg/promotion/dispatch/...`).
