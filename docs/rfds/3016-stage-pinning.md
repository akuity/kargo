# RFD: Auto-promotion holds after rollback

Issue: https://github.com/akuity/kargo/issues/3016

## Problem

Auto-promotion normally chooses the best available `Freight` for a `Stage`: the
newest eligible `Freight`, or the immediate upstream `Freight` when using
`MatchUpstream`.

That is correct during normal operation, but it creates a rollback footgun. If
auto-promotion promotes `Freight B`, and an operator rolls the `Stage` back to
`Freight A`, Kargo can later promote forward again to `B`, or to newer
`Freight C/D/E` that may contain the same regression.

During an incident, rollback is operator intent. Automation should yield.

## Decision

When a Kargo-controlled rollback path promotes `Freight` that is not the current
auto-promotion candidate for that origin, Kargo should pause auto-promotion for
that origin.

The internal mechanism is an origin-scoped pin on the `Stage`; the UX should
describe this as "auto-promotion paused after rollback."

This hold is explicit to resume. Kargo should not auto-resume just because newer
`Freight` appears.

The important product rule is simple: if rollback intent deliberately chooses
something other than what automation would choose, automation steps back until
something explicitly resumes it.

## Terms

- **Origin**: the requested `Freight` origin, keyed like `Warehouse/my-warehouse`.
- **Eligible Freight**: `Freight` available to a `Stage` through direct
  subscription, upstream verification and soak, or manual approval.
- **Auto candidate**: the `Freight` auto-promotion would currently choose for
  an origin. For `NewestFreight`, this is the newest eligible `Freight`. For
  `MatchUpstream`, this is the `Freight` currently occupying the upstream
  `Stage`(s) per the policy's matching rules. Comparison must respect both
  policies.
- **Promotion source**: a persisted marker on every `Promotion` that
  distinguishes `Promotion`s created by the Stage controller's auto-promotion
  loop from everything else. This RFD only needs that distinction. The exact
  enum should be `auto` / `nonAuto`. The non-auto value is deliberately named
  for the control-path distinction, not for actor identity: future automated
  rollback may create non-auto `Promotion`s too. Admission should default
  missing source markers to `nonAuto` and should not allow users to spoof
  `auto`; `auto` is reserved for the Kargo control plane. Source should be
  immutable after create and is never inferred from missing actor data.
- **Rollback-intent Promotion**: a non-auto `Promotion` created by a
  Kargo-controlled path to move a `Stage` to something other than the current
  auto candidate for that origin. In this RFD, those paths are API, CLI, and
  UI rollback flows. Future automated rollback can use the same hold path, or
  provide an equivalent guard, as long as it preserves the same safety property.
- **Hold / pin**: status on a `Stage` saying auto-promotion is paused for one
  origin and preserving the selected rollback `Freight`.

## Behavior

Rollback should be atomic. The API, CLI, and UI should support a single action
that promotes selected `Freight` and pauses auto-promotion for the affected
origin before any newer auto-promotion can race in.

While an origin is held:

- Kargo does not create new auto-promotions for that origin
- Kargo does not run already-pending auto-created promotions for that origin
- non-auto-created promotions are still allowed
- other origins requested by the same `Stage` may continue auto-promoting
- downstream `Stage`s continue to see and follow the held `Stage` normally

For multi-origin `Stage`s, the pin applies only to the origin of the target
`Freight` in the rollback-intent `Promotion`. Carried-over `Freight` from the
`Promotion`'s full `FreightCollection` should not cause unrelated origins to be
pinned.

The same target-origin rule applies to auto-created `Promotion`s. An
auto-created `Promotion` targets exactly one origin through `spec.freight`; it
may carry over the current `Freight` for other origins in its
`FreightCollection`. A hold only blocks auto-created `Promotion`s whose target
`spec.freight` belongs to the held origin. Carrying held `Freight` forward
unchanged while another origin advances is expected multi-origin behavior.

Promotion of the current auto candidate should not create a hold. It is the
"resume automation by choosing the thing automation would choose" case.

## Lifecycle

Create or update a hold when a Kargo-controlled rollback action creates a
non-auto `Promotion` to non-auto-candidate `Freight`.

V1 does not guarantee protection for direct `Promotion` CRD writes or external
automation that creates rollback `Promotion`s without also writing a hold. The
future-work fix is controller-side detection of non-auto `Promotion`s that
target non-auto-candidate `Freight`, but this RFD keeps V1 scoped to write
paths Kargo can make atomic.

Rollback holds have two states:

- **Pending**: blocks auto-promotion while the rollback `Promotion` is
  waiting or running.
- **Active**: preserves the rollback after that `Promotion` succeeds.

For API, CLI, and UI rollback flows, Kargo should write the pending hold before
creating the rollback `Promotion`. The pending hold stores the selected `Freight`
and the pending `Promotion` name/UID. If the process stops before the
`Promotion` is created, controller recovery clears the pending hold after a
short grace period.

That grace-period cleanup is a deliberate availability/safety tradeoff. Keeping
an abandoned `Pending` hold forever preserves the rollback intent but can leave
a `Stage` stuck in a state nobody can explain. Clearing it after the grace
period may allow auto-promotion to move forward if the client never retries
during an incident. The event for this path should therefore be a loud
operator signal, not routine cleanup.

The hold `CreatedAt` timestamp is part of the safety model. A malformed active
hold without `CreatedAt` should not be cleared by a later clear-on-success
annotation because the controller cannot prove the clear request happened after
the rollback intent. Explicit resume can still clear that hold.

Finalize the hold only when the linked rollback `Promotion` succeeds. If that
`Promotion` fails, errors, or is aborted, clear the pending hold only if it
still references the same `Promotion` UID. This prevents an older failed
rollback from clearing a newer hold.

Clear a hold when the user explicitly resumes auto-promotion for that origin,
or when the `Stage` no longer requests that origin.

If the pinned `Freight` is deleted or garbage-collected, keep the hold and
surface that the selected `Freight` is gone along with the original `Reason`
and `Actor` so an operator can decide whether to resume or hold longer. Losing
the pinned object should not silently resume automation during an incident.
The resume endpoint remains valid in this state — the operator chooses
deliberately, with audit context, rather than the system defaulting either
way.

Successive rollback requests do not stack or replace a hold that already exists
for that origin. If an origin already has a `Pending` or `Active` hold, a second
same-origin rollback request should return `409 Conflict` and ask the operator
to wait for the current rollback to settle or explicitly resume first. This
keeps a failed second rollback request from erasing the hold that is already
protecting the `Stage`.

## Resume UX

Resuming automation should be deliberate and cheap:

- `Resume auto-promotion` clears the hold after showing the current auto
  candidate. It does not itself create a `Promotion`.
- `Promote current auto candidate and resume` creates a `Promotion` for the
  current auto candidate and clears the hold on success.
- If no candidate exists, resume simply clears the hold.

These are two user intents, not necessarily two adjacent buttons. In the
current UI, the Stage card's `Resume` action clears the hold after a
confirmation that names the current candidate. The existing promote flow covers
"promote current candidate and resume" by marking that `Promotion` to clear the
hold only after it succeeds.

Resume is rejected while the hold is still `Pending` — i.e. the rollback
`Promotion` itself has not reached a terminal state. The UI should disable
the resume action and surface "rollback in progress" copy until the
controller transitions the hold to `Active` (or clears it on rollback
failure). Operators who genuinely want to abandon an in-flight rollback should
use the existing abort `Promotion` action. When the rollback `Promotion` reaches
`Aborted`, controller recovery clears the pending hold.

The UI should say which origin is paused and which `Freight` is pinned, for
example:

> Auto-promotion is paused for `Warehouse/api` after rollback to `Freight A`.
> Other origins for this Stage continue auto-promoting.

The same capability should exist in CLI and API flows, not only in the UI. The
REST API is part of this change; CLI ergonomics can land as a follow-up wrapper
over that API. See [API surface](#api-surface) for the concrete endpoints and
request shapes.

### UX invariants

Pixel-level treatment is left to the UI implementation pass, but these
invariants are load-bearing for the feature to deliver the safety the
[Decision](#decision) promises:

1. **The pin is rendered on the `Freight` indicator, not just the `Stage`
   node.** The thing pinned is a `Freight`. A Stage-only badge invites
   "the Stage is broken" misreadings. A `Stage`-level "paused" badge may
   also exist as an at-a-glance pipeline signal, but it does not replace
   the per-`Freight` indicator.
2. **Resume intents stay distinct.** "Clear the hold" and "promote current
   candidate and clear the hold" express different operator intents
   post-incident. The UI may put them in different surfaces, but it must not
   make one look like the other.
3. **`Pending` and `Active` are visually distinct.** `Pending` shows the
   rollback `Promotion` in flight and disables resume. `Active` shows the
   hold settled and enables resume. A single "paused" state that conflates
   them re-introduces the "can I resume mid-rollback?" ambiguity.
4. **Multi-origin paused state is rendered per origin, never collapsed
   into a Stage-wide "paused."** A Stage with one paused origin out of
   three is still auto-promoting two origins. A Stage-wide treatment
   would misrepresent that and discourage multi-origin adoption.
5. **The pinned `Freight` being deleted does not silently hide the
   hold.** The hold's `Reason` and `Actor` remain visible alongside a
   "pinned `Freight` no longer exists" indicator, so the operator
   resumes (or not) with full context. See [Lifecycle](#lifecycle).

### Initiating a rollback from the UI

The existing "Promote" modal already lets operators pick a `Freight` for a
`Stage`. When the selected `Freight` is the current auto candidate, the
modal's behavior is unchanged — primary button is `Promote`, no extra
copy. When the selected `Freight` is *not* the current auto candidate, the
modal must:

- Change the primary button copy to **`Roll back and pause auto-promotion`**.
- Surface a one-line warning naming the current auto candidate, for
  example "This is older than the current auto-promotion candidate
  `1.27.3`. Auto-promotion for `Warehouse/api` will pause until you
  explicitly resume."
- Offer an optional **Reason** text field that is passed through to the
  `reason` field on the promote request and persisted on the hold.

The decision of which copy/state to render is made client-side from the
GET candidate endpoint described in [API surface](#api-surface). The
server still enforces the rule independently — a client that fails to
render the warning still gets a hold created — but the copy change is
what makes the consequence legible *before* the click.

## Implementation shape

Add origin-scoped pin state to `StageStatus`.

```go
type StageStatus struct {
    // ...
    AutoPromotionHolds map[string]AutoPromotionHold `json:"autoPromotionHolds,omitempty" protobuf:"bytes,16,rep,name=autoPromotionHolds" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
}

type AutoPromotionHoldState string

const (
    AutoPromotionHoldStatePending AutoPromotionHoldState = "Pending"
    AutoPromotionHoldStateActive  AutoPromotionHoldState = "Active"
)

type AutoPromotionHold struct {
    Freight       FreightReference       `json:"freight"`
    State         AutoPromotionHoldState `json:"state"`
    PromotionName string                 `json:"promotionName,omitempty"`
    PromotionUID  types.UID              `json:"promotionUID,omitempty"`
    Actor         string                 `json:"actor,omitempty"`
    Reason        string                 `json:"reason,omitempty"`
    CreatedAt     *metav1.Time           `json:"createdAt,omitempty"`
}
```

`AutoPromotionHolds` is a map keyed by the canonical string representation of
the held origin, e.g. `Warehouse/api`. This matches the control-state
semantics: one hold per origin. It also gives API server and controller writers
a natural patch boundary, avoids duplicate holds for the same origin by
construction, and leaves room for future per-origin options such as
expiration/policy fields.

The map key is the source of truth for hold identity. `FreightReference.Origin`
should still be written consistently for client convenience, but controllers
must look up and mutate holds by key. The key format is therefore a persisted
status contract. Implementation should centralize it behind helpers such as
`FreightOrigin.String()` / `ParseFreightOriginKey()` and cover the format with
tests.

A list shape was considered because `FreightOrigin` is a structured object, but
lists are awkward to patch safely when multiple API requests and controllers
may update one origin. The previous `+listType=atomic` shape made every
single-origin update a whole-list update, which is the wrong mutation model for
origin-scoped status.

The hold deliberately does **not** carry a `CurrentCandidate` field. Clients
fetch the current auto candidate on demand via the dedicated endpoint in
[API surface](#api-surface). Embedding it in status would force a controller
write on every reconcile of a held origin even when no candidate change had
occurred, fanning out into watches and consumers for purely informational data.

Add a persisted source marker to `Promotion`s. The load-bearing distinction is
`auto` for `Promotion`s created by the Stage controller's auto-promotion loop
versus `nonAuto` for all other paths. `nonAuto` is intentionally not named
`manual`: the source value must not imply a human actor because other automation
may also create non-auto `Promotion`s. Missing source markers default to
`nonAuto` at admission. `auto` is reserved for the Kargo control plane: the
validating admission webhook must reject any request that attempts to set or
change the source to `auto` when the authenticated subject is not the
controller's service account. Source should be immutable after create and is
never inferred from missing actor data.

The marker is **load-bearing**, not bookkeeping: controllers rely on it to
identify which queued `Promotion`s are auto-created and therefore abortable as
`SupersededByAutoPromotionHold` when a hold lands. It is also the foundation
for audit display and any future controller-side rollback auto-detection.

The controller should:

- apply pending holds before creating or selecting auto-promotions
- use one shared auto-candidate resolver with the API server. The Stage
  controller and the promote endpoint must compare against the same candidate
  for an origin.
- keep candidate resolution limited to the policy decision: live
  `ProjectConfig` enablement, `Freight` availability, and the origin's
  selection policy. The temporary duplicate-`Promotion` guard described below
  is a write-side coexistence guard, not part of "what would automation choose
  right now." Mixing it into candidate resolution would hide the latest
  candidate from rollback/hold logic and could let a future newer `Freight`
  move the `Stage` forward unexpectedly.
- thread accumulated `newStatus` forward between sub-reconcilers (or use an
  in-memory signal) so a hold written by `syncPromotions` is visible to
  `autoPromoteFreight` in the same reconcile pass. The current sub-reconciler
  pattern deep-copies `stage.Status` from the original pointer at each step,
  which leaves a one-pass race window
- abort pending auto-created promotions that target held origins, using a
  terminal reason such as `SupersededByAutoPromotionHold` so the audit trail is
  clear. Abort applies only to `Promotion`s that have not yet started running;
  in-flight `Promotion`s run to terminal state, and the hold blocks the next
  iteration
- ensure the Promotion controller checks for a held target origin before
  starting an auto-created `Promotion`. This closes the stale-cache race where
  the Stage controller creates an auto-`Promotion` after a hold has landed but
  before its cache observes the hold. If the `Promotion` is still pending, mark
  it terminal with `SupersededByAutoPromotionHold` before any steps run.
- compare non-auto promotions to the auto candidate for that origin, respecting
  both `NewestFreight` and `MatchUpstream`
- use source-specific availability times where possible: direct Freight
  creation, approval time, verification plus soak, or upstream occupancy
- fill the empty `PromotionUID` on `Pending` holds during the first reconcile
  that observes the referenced `Promotion`, per the recovery rules in
  [Controller recovery](#controller-recovery-pending--active-or-cleared)
- garbage-collect holds for origins no longer requested by the `Stage`

The endpoints that create or clear holds must also enqueue the `Stage` for
reconciliation (e.g. by bumping `metadata.annotations[kargo.akuity.io/refresh]`
on the patch). The `Stage` controller does not watch its own status, so a
status-only mutation will not cause auto-promotion to resume until the next
periodic reconcile or unrelated event.

Status is the right home for holds because they reflect observed rollback
intent and controller-managed recovery state, not desired declarative
configuration. Explicit resume should go through API/CLI endpoints that
authorize the `promote` verb and patch status through the internal client.
That internal status patch means the API server ServiceAccount also needs
RBAC for `stages/status`; ordinary users only need `promote` authorization on
the target `Stage`.
Declarative, spec-level preemptive pinning can be considered later as a
separate feature.

Once holds are in place, the current broad "never auto-promote any `Freight`
that already has a `Promotion` for this `Stage`" mitigation becomes redundant.
Phase the change in two steps to avoid regressing to the original #3016
footgun: ship holds first and let the mechanism soak in a release, then relax
the broad guard in a follow-up. Holds become the primary mechanism for
preserving rollback intent; duplicate guards should only prevent redundant or
previously failed auto-promotions.

## API surface

Three REST capabilities live under the existing project-scoped router in
`pkg/server/rest_router.go`:

- extend the existing promote endpoint
- add a resume endpoint
- add a read-only current-candidate endpoint

ConnectRPC equivalents are intentionally omitted because ConnectRPC is slated
for removal in v1.12.0. The UI implementation for this feature should use the
REST endpoints even if neighboring UI flows have not yet migrated.

The single load-bearing rule is the one stated in [Decision](#decision):

> If a non-auto promotion targets something other than the current auto
> candidate for an origin, Kargo pauses auto-promotion for that origin.

The server detects the comparison; the client does not opt in. The promote
endpoint recomputes the current auto candidate at request time using the same
resolver as the Stage controller. Status fields are useful for display, but
they are not the source of truth for this decision.

### Promote (existing endpoint, extended)

```
POST /v1beta1/projects/:project/stages/:stage/promotions
```

Request:

```json
{
  "freight": "<freight-name>",          // or "freightAlias"
  "reason": "Regression in 1.27.2",     // NEW: optional, free-form,
                                        // persisted to the hold when one
                                        // is created
  "expectedAutoCandidate": "<freight-name>" // NEW: optional precondition
}
```

The new fields are additive and OpenAPI-marked optional.
`expectedAutoCandidate` is a request precondition for UX flows that were
rendered from the candidate endpoint. When set, the server recomputes the
candidate for the target origin at request time and returns `409 Conflict`
without creating a `Promotion` or hold if the candidate no longer matches. The
UI should set this field for "promote current auto candidate and resume" and may
also set it for rollback confirmation copy that names a specific current
candidate.

Handler behavior depends on whether the target `Freight` is the current
auto candidate for its origin (see [Terms](#terms)):

| Target                  | Hold change                          | `Promotion` |
|-------------------------|--------------------------------------|-------------|
| **!= auto candidate**   | Pending immediately; Active on success | created     |
| **== auto candidate**   | Existing hold for the origin cleared on success | created     |

If no auto candidate exists, a normal promotion does not create a hold and does
not implicitly clear an existing hold. Explicit resume remains the way to clear
an active hold when there is no candidate to promote.

If the target is not the auto candidate and a hold already exists for the same
origin, return `409 Conflict` without creating a `Promotion` or mutating the
hold. There is only one rollback hold lifecycle per origin at a time.

The "clear on success" case needs a durable marker on the `Promotion`, not an
immediate status mutation. The promote endpoint annotates the `Promotion` with
the origin and observed hold identity to clear, and the Stage controller removes
that active hold only after the `Promotion` succeeds and the identity still
matches. This avoids resuming auto-promotion when the "promote current
candidate" attempt fails, is aborted, or races with a newer rollback hold.

#### Crash-safe write sequence (target != auto candidate)

The handler computes its writes in an order that survives a server crash
between any two steps. The controller's recovery rules (below) drain any
partial state.

1. **Build the `Promotion` object in memory** via `NewPromotionBuilder`.
   This determines the final `Promotion.Name` (`<stage>.<ulid>.<short-hash>`)
   without contacting the API server. The `Promotion` is stamped with a non-auto
   source marker.
2. **Patch a Pending `AutoPromotionHold`** into
   `Stage.Status.AutoPromotionHolds[originKey]`, where `originKey` is the
   canonical string for the target `Freight`'s `Origin`. `PromotionName`
   is populated from step 1; `PromotionUID` is left empty (UIDs are
   assigned by the API server on Create); `Freight` is the request target;
   `Reason` is the request field; `Actor` is the authenticated caller;
   `State = Pending`.
3. **Create the `Promotion`** by writing the object from step 1. The API
   server assigns the UID at this moment.

Steps 2 and 3 are the atomicity-critical pair: as soon as step 2 lands,
auto-promotion for that origin is blocked. The pending hold blocks newer
auto-promotions during the window before step 3, so even if a
reconciliation interleaves the rollback intent wins.

`PromotionUID` is populated asynchronously by the Stage controller during
its first reconcile of the held origin — see *Controller recovery* below.
The API server intentionally does not block on a fourth write to fill the
UID itself: doing so would add a non-atomic step without changing the
semantics, since the controller has to handle the empty-UID case anyway
for crash recovery.

On any step failure, return the error to the client without rolling back
prior writes. Controller recovery handles partial state.

#### Controller recovery (Pending → Active or cleared)

`syncPromotions` is the home of the recovery logic, since it already
observes `Promotion`s for the `Stage`:

- **Pending hold with `PromotionName` set, hold's `PromotionUID` empty.**
  Look up the `Promotion` by name in the `Stage`'s namespace.
  - Found, `Promotion.UID != ""` → patch hold with the UID.
  - Not found, hold age < grace period (default 10m) → leave Pending;
    a retry from the API will likely complete the chain.
  - Not found, hold age >= grace period → clear the hold; emit a visible
    `HoldAbandoned` event. This is the one recovery path that can resume
    automation after an attempted rollback never produced a `Promotion`; the
    event should be visible enough for operators to investigate.
- **Pending hold with `PromotionUID` set.** Standard lifecycle.
  - `Promotion.Status.Phase == Succeeded` → transition hold to Active.
  - `Promotion.Status.Phase` terminal but not Succeeded → clear the
    hold *only if* the hold's `PromotionUID` still matches (UID-gated
    per [Lifecycle](#lifecycle)). Emitting a `RollbackFailed` event is useful
    follow-up audit work.
- **Pending hold with `PromotionUID` set, referenced `Promotion` no longer
  exists in the API.** Clear the hold. Emitting a `HoldAbandoned` event is
  useful follow-up audit work. This is an orphan cleanup path. The normal way
  to abandon an in-flight rollback is to abort the rollback `Promotion` and let
  the terminal non-success path above clear the hold.
- **Active hold with referenced `Promotion` deleted or GC'd.** Keep the
  hold; surface that the originating `Promotion` record is no longer available
  if the UI needs to show audit detail. Do not silently
  resume automation (also per [Lifecycle](#lifecycle)). Active holds
  describe operator intent that has already been carried out; losing
  audit history for the originating `Promotion` does not invalidate
  the intent.

Together these rules mean every partial write from the API server is
reconciled into a coherent end state without requiring transactional
semantics across two CRDs.

#### Why no `pauseAutoPromotion` flag

Earlier drafts of this RFD made the pause opt-in via a request flag.
That contradicted the rule above: a caller who forgot the flag could
reproduce #3016. Server-detected pause-by-default is the only safe
default. Defensive freeze (pinning to current state without a different
target) is intentionally out of scope.

### Resume auto-promotion (new endpoint)

```
POST /v1beta1/projects/:project/stages/:stage/auto-promotion/resume
```

Request:

```json
{
  "origin": { "kind": "Warehouse", "name": "nginx" }
}
```

Handler:

1. Read the `Stage`'s current holds.
2. **Reject `Pending` holds.** If any hold targeted by this request is
   `State == Pending`, return `409 Conflict` with a body explaining that
   a rollback `Promotion` is still in flight and naming the
   `PromotionName`. The caller should wait for the rollback `Promotion`
   to reach a terminal state — at which point controller recovery
   either promotes the hold to `Active` (resume is then legal) or
   clears the hold (resume is then unnecessary). Callers who genuinely
   want to abandon a rollback in flight should abort the rollback
   `Promotion`; the same recovery rules then apply.
3. Remove the matching `Active` hold entry from
   `Stage.Status.AutoPromotionHolds`. `origin` is required. This keeps
   multi-origin resume deliberate and prevents one click from resuming
   unrelated origins. The status patch removes only the same hold observed
   by the handler; if the hold changes during the retry window, return
   `409 Conflict` and ask the caller to reload.
4. Bump `metadata.annotations[kargo.akuity.io/refresh]` on the same
   `Stage`. This is required: the `Stage` controller's event filter is
   `GenerationChangedPredicate || RefreshRequested || …`, so a
   status-only mutation does not enqueue the `Stage`.

Both writes use the API server's internal client; authorization is
enforced at endpoint entry.

Returns `204 No Content` on success, `400 Bad Request` when `origin` is
missing or malformed, `409 Conflict` if the targeted hold is `Pending` or
changes during the request, and `404 Not Found` if no matching active hold
exists.

### Get current auto candidate (new endpoint)

```
GET /v1beta1/projects/:project/stages/:stage/auto-promotion/candidates
```

Returns the `Freight` automation would currently promote for each
requested origin on this `Stage`, computed with the same resolver the
controller uses. The resolver checks live `ProjectConfig` policy rather than
trusting the `Stage.status.autoPromotionEnabled` observation, so a stale
status field does not create or suppress holds. Authorizes against `get` on
`stages` (the same permission needed to read the `Stage` itself).

Response:

```json
{
  "candidates": [
    {
      "origin":  {"kind": "Warehouse", "name": "nginx"},
      "freight": {
        "name": "<freight-name>",
        "origin": {"kind": "Warehouse", "name": "nginx"}
      }
    }
  ]
}
```

Origins with no eligible candidate are omitted from the response.
If auto-promotion is not currently enabled for the `Stage`, the response is
empty. A candidate is only useful when automation is actually allowed to act;
otherwise the UI would warn about automation that cannot currently preempt the
operator.

This endpoint exists so the "Promote current auto candidate and resume"
UX from [Resume UX](#resume-ux) does not require clients to reimplement
candidate resolution. It is read-only and not status-resident, which
avoids per-reconcile status writes for held origins (a real concern
given that watches and consumers would observe each candidate refresh
as a state change).

Mutating endpoints still recompute the candidate at request time before
creating or resuming a `Promotion`. This endpoint is informational only.

### CLI follow-up shape

```
kargo promote --stage <stage> --freight <name> [--reason "..."]
kargo promote --stage <stage> --freight-alias <alias> [--reason "..."]
kargo resume-auto-promotion --stage <stage> --origin <kind>/<name>
```

The existing `promote` command can carry an optional reason; the server-side
rule handles pausing automatically when the target is not the current auto
candidate. `resume-auto-promotion` targets the resume endpoint directly and
requires the held origin so multi-origin stages resume only the intended lane.

### Authorization

Promote and resume require the caller to hold the `promote` verb on the target
`Stage`. The candidate endpoint requires `get` on the target `Stage`. The
status patch (hold mutation) and the annotation patch (refresh) use the API
server's internal client because status is not user-writable. Per `AGENTS.md`,
each call site that bypasses the authorizing client must carry a comment
documenting the justification.

## Concurrency and safety

Three actors can race on `AutoPromotionHold` state: API server replicas
handling user requests, the Stage controller's reconciliation loop, and any
future component that writes holds. This section enumerates the race scenarios
the implementation must address and the primitives it must use to address them.
Each scenario below is a required test case.

### Required primitives

**All hold mutations use resourceVersion-gated writes.** API server, Stage
controller, and any future status writer mutate
`Stage.Status.AutoPromotionHolds` with a status `Update` or JSON merge patch
that is guarded by the current `resourceVersion`. The map shape lets writers
patch a single origin key instead of rewriting a whole list, but it does not
remove the need to handle conflicts: two writers can still race on the same
origin. Strategic merge patch is not available for CRDs. On `Conflict`, the
writer re-reads, re-evaluates intent, and re-patches. Bounded retry budget
(default 3 attempts) is sufficient for this workload; failures past the budget
surface to the caller.

**Source marker is set atomically with `Promotion` Create, never via a
follow-up Patch.** Setting it on the in-memory object before Create
eliminates a "Create-then-Patch source" window where a controller could
read an unstamped `Promotion` and misclassify it. Admission enforces the
spoof check (below) on every Create.

**Refresh annotation bump on resume is a separate write.** Status and
metadata are different subresources, so the two writes cannot be a single
atomic patch. Intermediate state (hold cleared, refresh not yet bumped) is
safe: the next reconcile — whether triggered by the eventual annotation
bump, an unrelated event, or periodic resync — resumes auto-promotion. A
crash between the two writes delays resumption but does not break it.

### Race scenarios

**1. Concurrent user rollbacks for the same origin.** Two users (or one
double-click) submit rollbacks targeting different `Freight` for the same
origin within the API server's request window.

- Both build `Promotion`s in memory, both attempt to patch the Pending hold.
- ResourceVersion guards the second patch. It conflicts; the API server
  retries by re-reading status.
- On retry, the Pending hold for the origin already exists from the first
  request. Per [Lifecycle](#lifecycle)'s "successive rollback requests
  update the existing hold," the second request overwrites the hold with
  its own `Freight` and `PromotionName`, then creates its `Promotion`.
- Both `Promotion`s exist; only the second is referenced by the hold. The
  first runs to terminal state but cannot clear the hold (UID mismatch
  per [Controller recovery](#controller-recovery-pending--active-or-cleared)).
  Whichever `Promotion` succeeds last in wall-clock terms determines the
  Stage's `FreightHistory[0]`.
- Acceptable: rapid-fire double-submission converging on the latest
  intent is the right semantic.

**2. Auto-promotion racing with a new hold (controller TOCTOU).**
Controller reconcile starts at `t=0` reading no hold. API server writes a
Pending hold at `t=ε`. The Stage controller creates an auto-`Promotion` at
`t=δ` because its cache has not observed the hold yet.

- Mitigation A — final hold check: immediately before creating an
  auto-`Promotion`, the Stage controller re-checks hold state for the target
  origin using the freshest practical read and skips creation if a hold is
  present. This reduces the race window but is not the only guard.
- Mitigation B — start-time gate: the Promotion controller checks the target
  origin before starting any auto-created `Promotion`. If the origin is held
  and the `Promotion` is still pending, mark it terminal with reason
  `SupersededByAutoPromotionHold` before any steps run. This is the
  load-bearing guard for stale cache races.
- Mitigation C — cleanup: the next Stage reconcile also detects any pending
  auto-created `Promotion` targeting a held origin and aborts it with the same
  terminal reason.
- Correctness depends on Mitigation B. Mitigation A reduces wasted
  `Promotion` creation, and Mitigation C cleans up any orphaned pending
  `Promotion` left behind by stale reads.
- In-flight exception: if the auto-created `Promotion` was already running when
  the hold landed, it runs to terminal state. The hold blocks the next
  iteration, per [Lifecycle](#lifecycle). Acceptable.

**3. Resume racing with a new rollback.** API server reads an `Active`
hold for resume at `t=0`. Concurrent rollback writes a new Pending hold
for the same origin at `t=ε`. Resume's patch fires at `t=1`.

- ResourceVersion on the resume patch conflicts. Retry re-reads status,
  observes that the targeted hold is no longer the same active hold, returns
  [`409 Conflict`](#resume-auto-promotion-new-endpoint).
- Rollback `Promotion` proceeds normally. No silent break.

**4. Auto candidate moves between GET candidate and POST promote.** Client
fetches candidate `D` at `t=0`, renders "Promote D and resume." Between `t=0`
and the user's click at `t=1`, the auto candidate moves to `E` (new `Freight`,
soak elapsed). Client POSTs `freight=D` with `expectedAutoCandidate=D`.

- Server recomputes candidate at request time: `E`.
- Because the precondition no longer matches, the server returns `409 Conflict`
  without creating a `Promotion` or hold.
- The UI re-fetches candidates and asks the user to confirm the new target. This
  avoids turning a stale "resume to normal" click into a fresh rollback hold.
- This precondition is a stale-click guard, not a liveness guarantee. If a
  Warehouse is moving faster than an operator can confirm, repeated `409`s are
  acceptable for V1; the UI should keep the failure clear and let the operator
  make a deliberate rollback choice.

**5. Rollback `Promotion` aborted or deleted while hold is `Pending`.** User
aborts the rollback `Promotion`, or the referenced `Promotion` disappears before
the hold becomes Active.

- If the `Promotion` reaches terminal `Aborted`, controller recovery clears the
  pending hold through the normal terminal non-success path and emits
  `RollbackFailed`.
- If the referenced `Promotion` no longer exists, controller recovery clears
  the hold and emits `HoldAbandoned`. See
  [Controller recovery](#controller-recovery-pending--active-or-cleared).

**6. Source marker spoofing.** A non-controller subject submits a
`Promotion` with `source: auto` directly via kubectl, the REST API, or a
GitOps agent.

- The validating admission webhook reads the requesting subject from the
  `AdmissionReview` and rejects any Create or Update that attempts to set or
  change `source: auto` when the subject is not the controller's configured
  service account. Missing source defaults to `nonAuto`.
- Admission runs on every write path, so this closes the spoof loophole
  regardless of write source. The controller's SA name is configured at
  webhook deploy time (envvar or flag), not inferred.

**7. Sub-reconciler one-pass race within a single reconcile.** Documented
by the spike: each sub-reconciler does `newStatus := *stage.Status.DeepCopy()`
from the original pointer, so a hold written by an earlier sub-reconciler
in the same pass is invisible to a later one.

- Implementation MUST thread `newStatus` forward (assign to `stage.Status`
  between sub-reconcilers) so within-pass writes are observed downstream.
  See [Implementation shape](#implementation-shape) and the spike README.
- Tests must exercise both orderings: `syncPromotions` writing a hold
  before `autoPromoteFreight` runs, and `autoPromoteFreight` running
  before a hypothetical future sub-reconciler that observes holds.

### Multi-API-server and leader election

Stage controllers run with leader election (controller-runtime default);
only the leader actively reconciles. There is no intra-controller race
within a single Kargo deployment.

API server replicas have no leader election; multiple replicas can serve
concurrent requests for the same `Stage`. ResourceVersion-gated patches +
retry-on-conflict are sufficient to serialize them at the API server
layer; no separate locking, lease, or leader is needed.

### Acceptance criteria

Each numbered race scenario above is a required test case. Where the
scenario crosses API server and controller, the test exercises the
controller alone (with API-server writes simulated as direct
`client.Patch` calls) and asserts the controller converges to the correct
end state regardless of write ordering.

## Relationship to automated rollbacks

This RFD does not design automated rollback policy. That belongs with the
FailurePolicy / automated rollback work.

It provides a reusable safety primitive for that future work, but the two
features do not need to be tightly coupled. If a future policy promotes a
`Stage` back to a previous known-good `Freight`, normal auto-promotion must not
immediately move the `Stage` forward again to the failed `Freight`, or to newer
`Freight` that may contain the same regression.

Future automated rollback can reuse this hold path:

1. Resolve the rollback target.
2. Create or update a Pending hold for the affected origin.
3. Create the rollback `Promotion`.
4. Activate or clear the hold based on that `Promotion`'s terminal result.

It can also choose a different implementation, as long as it preserves the same
safety property: rollback must not be immediately undone by normal
auto-promotion.

The stable contract is that safety property, not a policy-specific annotation
or source value. Automated rollback may eventually get its own source marker or
event reason for better audit display, but this RFD's hold correctness must not
depend on knowing which policy created the rollback. The only source value with
special control-plane behavior in this RFD is `auto`, meaning "created by the
normal Stage auto-promotion loop."

This boundary keeps the features from becoming a ball of mud:

- The hold object stores only rollback-preservation state: origin, pinned
  `Freight`, lifecycle state, linked `Promotion`, actor, reason, and time.
- The future rollback design owns policy questions: when to roll back, which
  `Freight` is considered stable, how many attempts to make, and when automation
  may resume.
- The resume endpoint is available as the shared way to clear holds. Future
  policy may call it deliberately when its own criteria are met; OSS does not
  auto-resume merely because newer `Freight` appears.

### Out-of-band rollbacks remain unprotected in V1

Direct `kubectl create` of a rollback `Promotion`, or external automation that
writes `Promotion`s without also writing a hold, will not get hold protection in
V1. The OSS auto-promotion loop may re-promote the auto candidate.

The future-work fix is controller-side auto-detection: `syncPromotions`
observing a non-auto `Promotion` targeting non-auto-candidate `Freight` and
writing the hold itself. Deferred because the sub-reconciler one-pass race
([Race scenario 7](#race-scenarios)) becomes load-bearing rather than
belt-and-suspenders. Revisit in a follow-up RFD if real users hit this gap.

### Event and audit consistency

Hold creation events on the `Stage` carry the `Actor` and `Reason` from the
hold. Human rollback paths set `Actor` to the authenticated user. Future
automated rollback can set it to its controller identity or another stable
synthetic actor. Audit readers should distinguish humans from automation by the
actor/source fields, not by assuming every non-auto `Promotion` is human.

## Validation

A spike on `jboykin/spike-auto-promotion-holds` confirmed end-to-end that the
type addition plus the early-continue in `autoPromoteFreight` correctly
suppresses auto-promotion while an Active hold exists, and that auto-promotion
resumes once the hold is cleared and a refresh is bumped. The spike also
surfaced the three implementation requirements that drive [Implementation
shape](#implementation-shape) and [API surface](#api-surface): the refresh
bump on resume, the sub-reconciler `newStatus` propagation, and the
`NewPromotionBuilder` naming requirement.

Full reproduction steps, manifests, and the empirical narrative are in
[`hack/spike-3016/README.md`](../../hack/spike-3016/README.md).

A later local Tilt/kind run on the production branch validated the same user
path through REST, CLI, and the dashboard: stale candidate preconditions return
`409`, `kargo promote --reason` records an Active origin hold, the dashboard
shows the per-origin `Auto-paused` state and candidate-aware resume
confirmation, and `resume-auto-promotion` clears the hold. One coexistence
gotcha showed up clearly: while the older duplicate-`Promotion` guard remains,
clearing a hold may not immediately create another auto-`Promotion` for a
`Freight` that already has a `Promotion` to the same `Stage`. That is consistent
with the phased rollout; the stable contract for this RFD is that the hold is
cleared deliberately and automation is no longer blocked by the hold.

## Test scenarios

The implementation should explicitly cover:

- Rollback to `A` while the auto candidate is `B` → hold becomes
  Active; `B` and any newer `C/D/E` are blocked.
- Promotion of the current auto candidate → no hold created.
- Promotion of the current auto candidate while an active hold exists →
  `Promotion` is marked to clear that exact hold only after success.
- Clear-on-success `Promotion` races with a newer same-origin rollback hold →
  newer hold survives because the recorded hold identity no longer matches.
- Auto-promotion disabled on the `Stage` → candidate endpoint returns no
  candidates; promote does not create or clear holds based on phantom
  candidates.
- Pending hold + rollback `Promotion` creation fails → pending hold eventually
  cleared by controller recovery.
- Pending hold + rollback `Promotion` fails after running → pending hold cleared
  only when `PromotionUID` matches the failed `Promotion`.
- Successive rollback requests → `409 Conflict` while the origin already has a
  `Pending` or `Active` hold.
- Multi-origin `Stage`: rollback on origin `X` → origin `Y` continues
  auto-promoting; carried-over `Freight` from `X`'s rollback `Promotion` does
  not pin `Y`.
- Multi-origin `Stage`: origin `X` is held, and an auto-created `Promotion`
  advancing origin `Y` may carry `X`'s pinned `Freight` forward unchanged
  without being blocked. Holds block the target origin of an auto-`Promotion`,
  not every origin present in its `FreightCollection`.
- Pinned `Freight` deleted or garbage-collected → hold persists; UI reflects
  missing `Freight`.
- `Stage`'s requested `Freight` drops the held origin → hold cleared; if the
  origin is later re-added, auto-promotion resumes with a fresh candidate.
- Queued auto-created `Promotion` targeting a now-held origin → aborted with
  `SupersededByAutoPromotionHold`.
- Promotion controller receives a pending auto-created `Promotion` targeting a
  held origin → aborts it before running any steps.
- In-flight auto-created `Promotion` when a hold is created → runs to terminal
  state; hold blocks the next iteration.
- Stale `expectedAutoCandidate` on promote request → `409 Conflict`; no
  `Promotion` or hold created.
- **Phase-out coexistence**: while the existing "never auto-promote a `Freight`
  that already has a `Promotion` for this `Stage`" guard still ships alongside
  holds, the two mechanisms do not conflict. After a hold is cleared and the
  pinned `Freight` is no longer the latest, the next auto-promotion target is
  whatever the resolver picks — even if intermediate `Freight` were previously
  blocked by the broad guard, the relaxed-guard flow still selects forward
  correctly.
- **Resume on Pending**: API resume against an origin whose hold is `Pending`
  returns `409 Conflict`; the hold remains; the rollback `Promotion` continues
  to terminal state.
- **Resume requires origin**: missing or malformed origin returns
  `400 Bad Request`; origin-scoped resume clears only the requested origin.
- **Installed RBAC**: the API server ServiceAccount can patch
  `stages/status`, because holds live in status while the user-facing action is
  still authorized as a `promote` operation.
- **Source marker spoofing**: a non-controller subject submitting a `Promotion`
  with `source: auto` is rejected by the admission webhook; missing source
  defaults to `nonAuto`.
- **Future automation composition**: a non-human actor using the same hold +
  rollback `Promotion` sequence gets the same Pending → Active/cleared
  lifecycle as a human rollback path. A different future design remains valid if
  it preserves the same "rollback is not immediately auto-promoted away" safety
  property.

## Rejected alternatives

- **Disallow manual promotions when auto-promotion is enabled.** This avoids
  the footgun, but blocks the rollback and hotfix workflows users need most.
- **Stage-wide pause.** Too coarse for multi-origin `Stage`s. A rollback of one
  origin should not stop unrelated origins from moving.
- **History-based suppression.** "Never re-promote Freight in history" blocks
  `B`, but still allows break-forward through `C/D/E`. It also turns audit
  history into control state.
- **Rejected Freight list.** This can remember `B`, but it is harder to
  explain, needs GC, and still does not express the current operator intent as
  clearly as a hold.
- **Auto-resume when newer Freight appears.** This solves the happy path where
  `C` is the fix, but fails the incident path where `C/D/E` carry the same
  regression.

## Open questions

- **Declarative preemptive holds.** Out of scope for this RFD. Revisit as a
  separate spec-level feature if users ask for it.
- **TTL on holds.** Recommended against. Incident windows are unbounded; a
  TTL forces a choice between renewal (a rediscovered footgun) and silent
  expiry (the exact failure mode this RFD prevents). Revisit only on concrete
  user demand.
- **Events and metrics.** Recommended to emit:
  - `Event`s on the `Stage` for hold creation, update, and clear, including
    the `Actor` and `Reason` so the audit trail is complete without joining
    other resources. `HoldAbandoned` should be conspicuous because abandoned
    pending-hold cleanup can resume automation after an attempted rollback never
    produced a `Promotion`.
  - Metrics:
    - `kargo_auto_promotion_holds_active` — single cluster-wide gauge of
      currently-active holds.
    - `kargo_auto_promotion_holds_created_total` /
      `kargo_auto_promotion_holds_cleared_total` — counters labeled by a bounded
      `cause` enum (e.g. `rollback`, `stage_origin_dropped`). Do not label
      metrics with the free-form hold `Reason`.
    - `kargo_promotions_aborted_total{reason="SupersededByAutoPromotionHold"}`
      — counter.
    Per-`Stage` gauges are deliberately avoided: hold cardinality should not
    grow with project size.
- **Policy-driven hold resume.** Future automated rollback policy could
  plausibly clear a hold when a newer `Freight` verifies clean. That policy
  owns the extra context and risk decision. The OSS rule "no auto-resume on
  newer Freight" remains correct for this RFD; policy-driven resume can layer on
  top by calling the resume endpoint deliberately.
