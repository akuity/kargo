# margo + kargo 2.0 — Design Spine

**Status:** Exploratory design spine. Captures a conceptual foundation, not a
finalized architecture. Several decisions are *deliberately deferred* (see
[Open & Deferred Decisions](#7-open--deferred-decisions)).

**Codenames:**

- **margo** — the successor to / replacement for Argo CD (delivery/sync engine).
- **kargo 2.0** — the successor to Kargo (promotion/orchestration engine).

These are two *complementary* systems. This document defines what each is, the
line between them, the foundation they share, and the one genuinely hard
problem that sits between them.

---

## 1. Thesis — why replace rather than evolve

Both Argo CD and Kargo are over-fixated on Kubernetes. Kubernetes is made to
play **three** roles at once:

1. **Runtime** — where the controllers run. *Fine.*
2. **Deployment target** — what gets synced. *Fine, as **one** target.*
3. **System-of-record + API** — etcd-as-database, CRDs-as-API. **This is the
   sin.**

Conflating "we run on k8s" with "our API *is* k8s" poisons everything
downstream.

### Why it doesn't scale (the real reasons, not a slogan)

- **etcd is a coordination store, not a database** — no secondary indexes, no
  relational integrity, no real query layer, hard practical ceilings on object
  count and total size, and list/watch cost that grows with the whole working
  set rather than with your query.
- **The API is *derived from* the CRD model**, so every consumer inherits k8s
  semantics (resourceVersion, watch, label selectors as the only query
  dialect). You cannot offer a tenant/account model, cross-target federation,
  real audit history, or a decent query API without fighting the substrate the
  entire way.
- **One control plane ≈ one cluster.** Horizontal scale means *more clusters*,
  not *more capacity*, because the "database" is the cluster.

**The tell:** the thing that is genuinely hard in both products — "show me
everything, across everything, filtered how I actually think about it" — is hard
*because of* this choice, not despite it.

### The goal

Show what something better looks like. Kubernetes stays as a **runtime** and as
**a** deployment target — **not** the API, **not** the database, **not** the
only target.

---

## 2. The line — two complementary systems

The fundamental split is **"decide what/when" vs. "make it so":**

- **margo (A) — delivery/sync engine.** Reconciles declared desired state into
  actual state on targets. ("Make it so.")
- **kargo 2.0 (B) — promotion/orchestration engine.** Decides *what* the desired
  state should become and *when* it advances through a pipeline. ("Decide
  what/when.")

Argo CD collapses to a *single hardcoded point* in margo's design space; Kargo's
orchestration role is what kargo 2.0 generalizes.

---

## 3. Shared foundation

### 3.1 Persistence and API

- **Data belongs in a real database.** Relational vs. other is open — follow
  where the emerging design most strongly points.
- **Keep the reconciliation-loop model** (continuously syncing reality → desired
  state) as an *idea*, decoupled from k8s-the-implementation.
- **Right paradigm per concern — not one-size-fits-all.** Forcing a single
  paradigm onto everything is precisely the mistake already made. There are
  three distinct kinds of thing:

  | Kind | Example | Machinery |
  | --- | --- | --- |
  | **Records** | Freight | Store + query. No desired/actual, no loop. Where a real DB earns its keep. |
  | **Declarative state** | Stage; margo Application | Desired-vs-actual + reconcile loop. *The part that works — keep it.* |
  | **Imperatives** | Promotion; project creation; a verification run | A *verb*, not a resource. Decomposes into a **trigger/command** + an **execution record**. |

  Cramming all three into "CRD reconciled by a controller" is the specific
  mistake being unwound.

### 3.2 Reconciliation and topology

- **Reconciliation always runs at the agent.** The control plane collapses to
  **database + API + broker**. It scales precisely *because* it is no longer a
  swarm of watch-holding reconcile loops doing per-target work.
- **Agent — definition.** An agent is a placement of a syncer/executor inside
  the **trust/network boundary that holds the target's credentials and can reach
  the target**. Its defining quality is that it *can reach and administer
  everything its work touches*.
  - **k8s syncer:** that boundary *is* the cluster → agent runs in-cluster (the
    strongest form of co-location).
  - **tofu/cloud syncer:** there is no "inside the target" (the target is a
    cloud API) → the agent runs wherever the cloud credentials and network path
    legitimately live. It's co-located with the *authority over* the target, not
    the target.
- **Hybrid agent-based topology** (as pioneered by Akuity's Argo offering):
  agents are distributed and **phone home** to a centralized control plane. The
  three wins — none of them k8s-specific:
  1. **Single pane of glass** — agents aggregate status centrally.
  2. **Credential containment** — the control plane holds *no* target creds.
  3. **Outbound-only connections** — agents dial home; nothing dials into them.
- **margo agents and kargo agents are distinct roles.** A margo agent is placed
  to reach the *target*; a kargo agent is placed to reach everything the
  *promotion process* touches (git, registries, things behind a firewall). They
  are driven by different placement constraints and won't always co-locate or be
  interchangeable.

### 3.3 Self-hosting — "gitops all the way down"

- The *declarative config* of **both** systems must itself be manageable the
  same way anything else is — i.e., synced by margo. "gitops" is in scare quotes
  because git need not be involved.
- Managing platform config is **just another syncer whose target is the
  platform's own API/DB** (a "control-plane syncer") — an instance of the
  general design, not a special mechanism.
- **Only declarative (bucket-2) things are gitopsable.** Freight (records) and
  promotions (imperatives) are *not* in a source of truth — they are runtime
  data and actions. The three-paradigm taxonomy *predicts* exactly what does and
  doesn't belong in a source of truth. (A coherence check that passes.)
- **Bootstrap floor:** first install is via some other mechanism (likely a Helm
  chart); a manually created "app-of-apps" then pulls in everything else,
  including whatever lets the platform take over managing itself. Same trick
  Argo CD uses. Not overthought.

### 3.4 Tenancy and config layering

- **Multi-tenant by default.** "Tenant" need not mean customer — think
  department or project.
- **Everything is a project.** A project is the *uniform, hard tenancy
  boundary*. The platform team's project and an app team's project differ only
  in (a) what source of truth they read and (b) what target they sync into —
  gated by permissions. The platform team's project just happens to point at the
  platform's own config and sync *into the control plane itself*.
- **Creating a project is an imperative** (a self-service verb; creator becomes
  admin) — *not* a resource an operator pre-declares. The operator only
  declares, broadly, *who may create projects*.
- **Two config tiers:**
  - **Deployment layer** — the chart + its values, in a source of truth. Carries
    the broadest, most privileged, system-level config (who may create projects;
    the catalog of available source/syncer types). Established at install and on
    an ongoing basis. Self-managed = "deploy my own chart with these values."
  - **Resource layer** — projects and their contents, synced per-project.
- **Free privilege separation:** crown-jewels permissions live at the
  *deployment* layer, so they're controlled by whoever can deploy the platform
  and are **not editable from within the running system's API.** You can't
  escalate "who can create projects" by poking a resource.

### 3.5 Authorization & identity

Kubernetes was **not** doing authz "for free" in 1.x. The "everything's a CRD →
use k8s RBAC" decision forced an **impedance boundary**: OIDC subjects mapped to
k8s ServiceAccounts, then — for performance — *not* using clients scoped to those
SAs, which forced re-implementing k8s permission *and anti-escalation* enforcement
inside the API server. That chain was the source of the system's most complex
code and a run of CVEs. The enemy was **complexity from straddling two systems**,
not hand-rolling per se.

**Principles:**

- **Authz is ours, decoupled from persistence** — purpose-built for our resources
  and principals, not inherited from whatever stores the data.
- **One enforcement chokepoint, no backdoor.** With no k8s-API to bypass to,
  there is exactly one path to the data — our API — so authz is enforced there
  unconditionally. (The thing 1.x/Argo CD fought for; here it is structural.)
- **Native principals.** The OIDC identity *is* the principal — no claims→SA
  mapping, no "assume this SA."
- **Execute as the principal** — operations run against our store with the
  caller's own permissions, never a broadly-permissioned "god client" after a
  separate check. This closes the escalation gap that bred the CVEs.
- **Conventional and minimal** — lean on models people already understand.

**The grant resource (one synthetic primitive).** Follows 1.x's "Kargo Role" but
*better*: in 1.x it abstracted *over* a real SA+Role+RoleBinding trio (which could
leak / be edited out-of-band); in 2.0 there is **nothing underneath** — the grant
*is* the primitive and the single source of truth. One object expresses:

- **WHO** — any mix of: individual identities (human or automation), claim/group
  selectors (e.g. `team=payments`), and **named bearer tokens minted on the grant
  itself** (the automation path, below). Collapses principal + permissions +
  binding into one resource.
- **WHAT** — verbs: CRUD *plus* first-class domain verbs (`promote`, `approve`,
  …). Because we own the verb vocabulary, domain verbs need no special machinery
  (no more "dolphin verbs").
- **WHICH** — resource *types* or specific *instances* ("promote *this* stage").
  Instance-level is first-class, not a `resourceNames` hack.
- **Scope** — a project (= tenancy = authz boundary); plus a system scope for
  operators.

**Named tokens for automation.** Rather than modeling a separate service-account
principal and referencing it, a grant **mints named bearer tokens directly** (as
1.x does). Possession of the token = acting as that grant. Named ⇒ individually
rotatable, revocable, and attributable in audit; want narrower authority ⇒ mint
on a narrower grant (no per-token subsetting to design). *Bearer-token hygiene
applies:* support expiry/rotation; a leak's blast radius is exactly that one
grant.

**Enforcement.** Evaluated **inline against our one store** — no separate
stateful authz service (a Zanzibar-style service reintroduces the very two-store
impedance boundary that caused 1.x's pain). A small evaluator, or an *embeddable*
library (Cedar/Casbin) fed our data, are both acceptable; a separate service is
not. *(Hand-rolled-vs-embeddable is the one remaining open sub-choice.)*

**Anti-escalation (strict, no escape hatch).** To create or modify a grant — or
mint a token on one — you need (a) permission to manage grants in that project
**and** (b) every permission conferred must be a subset of what you *currently
hold*. **You cannot confer what you do not have.** No k8s-style `escalate`/`bind`
escape verbs. The same inline evaluator performs this check at write time, so
there is no separate admission layer to drift. The **deployment layer** seeds
top-level permissions as an un-escalatable ceiling; every grant below is bounded
by its writer's holdings — auditable top to bottom. ("Manage grants" is itself
just a permission, so meta-escalation is bounded by the same rule.)

**Two trust classes:**

- **Tenant principals** — humans (via OIDC) and automation (via named tokens) —
  live in the grant model above.
- **Agents** are a *separate infrastructure trust class*, not tenant principals.
  They enroll (bootstrap token → rotating creds / mTLS) and their authority is
  inherent to their role (pull desired state for their project, report status),
  not expressed as tenant grants. Keeping infra trust out of the tenant model
  preserves the boundary.

---

## 4. margo — delivery / sync engine

### 4.1 Two pluggable interfaces

margo is fundamentally about connecting **sources of truth** to **syncers**:

- **Source of truth** — a *versioned content provider*: "give me the current
  desired content, tell me its revision, notify me when it changes." It is
  deliberately **format-blind**. Reference impl: **git**. Others possible: OCI,
  S3.
- **Syncer (state reconciliation)** — an *interpreter + applier*: "given this
  content, reconcile my target to it, report drift/health." The syncer is the
  only side that understands the payload. Reference impl: **Kubernetes**. Others
  possible: **Terraform/OpenTofu**.

margo is the broker that wires a source to a syncer and runs the loop. Argo CD =
the single hardcoded pairing (source = git, syncer = k8s). margo makes both axes
pluggable — an N×M matrix.

### 4.2 The seam

- The source emits `(revision, opaque content bundle)` and is format-blind. Only
  the syncer understands the bytes. This is what makes it a true N×M matrix.
- **Rendering lives upstream of the rendezvous.** The source of truth holds
  *final, rendered* state; templates (Helm/Kustomize/…) are resolved by the
  producer *before* deposit. **Syncers never render** — they only apply + detect
  drift + report health. (A big simplification vs. Argo CD's repo-server.)

### 4.3 The source of truth is the rendezvous point

- **margo and kargo 2.0 never talk to each other directly.** kargo 2.0 writes
  final desired state *into a source of truth*; margo's syncer reads it out. The
  source is the neutral meeting ground — exactly the role git plays between Kargo
  and Argo CD today, now generalized into *an interface* rather than
  git-specifically.

### 4.4 The Application

- margo's core declarative resource is the **Application** — a generalization of
  Argo CD's Application. "Connect this source of truth to this execution
  environment," where **both endpoints are now pluggable** instead of hardwired
  git→cluster.
- It still carries **status** (sync + health) as a first-class property.
- **Single pane of glass is nothing exotic:** list applications, read their
  status; comparable across everything *because everything is an Application*.
  Pluggability widens the endpoints; it does not change the shape. No novel
  "status contract" to invent — the status vocabulary can stay close to what
  Argo CD already proved works.

---

## 5. kargo 2.0 — promotion / orchestration engine

### 5.1 Freight

A piece of **Freight** is an immutable **box of references to versioned
artifacts** (e.g., *this* git commit + *these two* container images). Every
unique combination of artifact revisions is a unique piece of freight with a
**deterministically derived thumbprint**.

- The artifact revisions within a single piece of freight **move through the
  pipeline as a unit.**
- Artifacts (or groups) that must move at **different cadences** belong in
  **separate boxes**.

Conceptually **unchanged from Kargo 1.x** apart from the storage medium. (Don't
fix what isn't broken.)

### 5.2 Stage = slots + instructions

A Stage is:

- **Slots** — "I require these N different kinds of freight."
- **Instructions** — "here is a process that constructs desired state (for
  something else to sync) *from* those N pieces of freight."

Its declarative desired state is really *the tuple of which freight occupies
each slot*; **promoting = changing a slot's occupant.**

- A Stage is a **promotion target, not a deployment target.** It need **not**
  represent an environment capable of hosting a workload. Examples: an
  assembly/aggregation stage (builds a release bundle/SBOM), a publish/release
  waypoint (cuts a GitHub release, pushes a chart, promotes an image tag), a
  gate/checkpoint stage, an infra stage (emits tofu desired state). **This
  flexibility already exists in Kargo 1.x — it is to be preserved, not
  introduced.**

Irreducible core of a Stage: **slots + instructions + whatever realizes/observes
it.** Everything else 1.x layers on (verification config, freight-sourcing
rules, history) sits on top and is not part of the essence.

### 5.3 How slots get filled — sources and origins

Each slot is fed by an **upstream source**, of two categories:

- **Origins** — where freight is *born*. Pluggable in *kind*:
  - **Warehouse** — external artifacts → freight. The one reference kind today.
  - **Junction** (envisioned) — freight from several sources → a **new *kind* of
    freight**. The resolution of the freight cadence rule: keep separately
    cadenced artifacts in different boxes early, then *fuse* them into one unit
    late in the pipeline for the final run to prod.
  - Others possible.
- **Upstream stages** — where freight *propagates* from another stage's output.

kargo's **origins are pluggable** the way margo's sources/syncers are — the same
philosophy (a small set of reference kinds, extensible later) showing up on both
sides.

**Pipeline** = the composition (inferred *or* formally declared) of stages plus
their slot-source wiring. In 1.x it is *emergent* — a DAG inferred from stages
referencing other nodes; there is no concrete Pipeline resource. Whether it
becomes an explicit first-class concept in 2.0 is **deferred** — contours are
identical either way; only the authoring surface differs.

### 5.4 Promotion dynamics (preserved from 1.x)

Freight fills a slot when a **promotion** effects it ("this freight needs to go
to this stage"). Promotions happen two ways:

1. **Manual** — a user says "I want this to go here." (Always the floor.)
2. **Auto-promotion** — during stage reconciliation, the loop looks for *new*
   freight *available* to the stage and, if found, creates a promotion.

**Availability.** Freight from an origin is available to a stage if:

- (a) the stage *wants* freight from that origin, **and**
- (b) the freight met its criteria — usually **verified** upstream of the target
  (optionally with a **soak time**), *or* an authorized user **approved** it for
  the target (clearance to bypass the usual criteria).

**Auto-promotion policies** (when enabled): promote the *newest available*, or
*match what's in use immediately upstream* (the "I'll have what she's having"
flavor). Typical shape: auto-promotion on the left (near origins), human-gated
manual promotion at prod on the right.

*New-substrate note:* a stage's reconciliation splits into **two concerns, two
homes** — *availability/orchestration* (pure control-plane data → the agent
co-located with the control plane) vs. *promotion execution* (the process itself
→ an agent holding the process's creds). Both "at an agent," different agents.

### 5.5 Promotion vs. verification — separate by design

- **Promotion is a state change. Verification is state validation.**
- Current state is **re-verifiable at any time without a new state change.**
  Verification can flip **pass → fail with no promotion at all** (a dependency
  rots, an SLO breaches).
- The common "promotion → verification" adjacency is *coincidence*: verification
  ran because the new state wasn't validated *yet*, **not** because it belongs
  to the promotion. They must remain separate — and making that separation more
  *visible* to users would be a welcome improvement over 1.x.

### 5.6 The execution model — the open frontier

- **The "ugly" in 1.x:** verification was built on Argo Rollouts' analysis
  framework (reuse-to-inherit-integrations). This **couples Kargo tightly to
  Argo Rollouts** — importing exactly the k8s-CRD dependence this whole project
  exists to cure — and the premise looks empirically weak: users lean on the
  k8s-Job escape hatch, not the inherited integrations.
  - **Kernel worth keeping:** the *integrations/providers* (Prometheus, Datadog,
    …) were genuinely worth wanting. The *vehicle* (the framework/CRDs) was
    wrong.
  - **Synthesis:** keep the **capability** (verification can query external
    validation providers) as **stdlib functions**; drop the **framework**.
- **Direction (Kent's preference):** promotions and verifications stay separate
  processes but should share a **consistent language/style**. If promotion moves
  away from YAML steps and toward **scripts over a "kargo stdlib,"**
  verification should move to the same model.
- **The actual hard problem is durable execution — not YAML-vs-script.** The
  YAML step engine's real superpower is **suspend/resume across arbitrary time**
  (e.g., "wait for this PR to merge," which may take days or weeks). A
  straight-line script cannot `await` for weeks without either blocking a
  process forever or a **durable-execution runtime** (durable history +
  checkpoint/replay — the Temporal / durable-functions pattern), which in turn
  constrains how scripts are written (determinism, replay-safety, side effects
  behind the runtime).
  - **Precedent to generalize (with a caveat):** Kargo 1.x's YAML step engine is
    a hand-rolled step engine with *resumption* (step position + accumulated
    outputs are checkpointed) — but **not** true *durability*. It leans on an
    **ephemeral working filesystem**; losing that filesystem forces a restart,
    because working state lives outside the recorded history. That gap is the
    line between "an orchestrator that checkpoints some things" and a genuinely
    durable engine. The design lesson for 2.0: **nothing load-bearing may live
    only on ephemeral disk** — every intermediate must be either *replayable*
    (re-derivable from recorded activity results) or *persisted*. This bites
    hardest in the long-wait case: a promotion parked for weeks on "wait for PR
    merge" will outlive its hosting agent/pod, so "it's still in the working
    directory" is not a survivable assumption. Kargo 1.x EE's "custom steps"
    system
    (long-running promotion pods with an orchestrator delegating to per-step
    containers, because arbitrary user logic must never run inside a controller
    container) is a *k8s-flavored* precedent. The generalization is an
    **agent-hosted durable workflow engine.**
  - **Crux:** *durable, suspendable execution, hosted on agents.* Script-vs-steps
    then becomes mostly an authoring surface layered on top.

#### Filesystem strategy (decided)

Durable-execution engines are replay/journal-based and **never** persist local
disk across steps or suspends; the accepted pattern is to externalize workspace
state and pass references. This becomes *entirely practical* — not a compromise —
once processes are written in a real language rather than YAML:

- **Lump all filesystem-dependent work into a single, synchronous durable step:**
  bare clone, check out branches from several repos, shuffle files, exec out to
  Helm/Kustomize, make and push commits — all at once. The filesystem never
  crosses a durable boundary; if the step fails it re-runs from a clean clone.
- **Steps that follow do the long-running, non-FS work** (wait for a PR to merge
  or close), which needs no working directory.
- The durable output of the FS burst is what gets **pushed to a source of truth**
  (a branch, a commit SHA). In other words, **the source of truth *is* the
  durable, externalized filesystem.** Push the branch, *then* wait for merge.
- Kargo 1.x's "every step is a step" YAML model made this awkward; a real
  language makes composing the coarse FS step natural, and it also makes that
  step trivially unit-testable (see local testability, below).

#### Durable execution engine (research findings; choice deferred)

**No engine solves our two hardest requirements** — a working directory shared
across steps/suspends (handled by the filesystem strategy above) and sandboxing
untrusted user Python (handled by running each step in an isolated container).
Both are ours to build regardless of engine. Engine choice is therefore only
about the durability substrate. License-filtered shortlist (Python-first,
self-hostable, OSS-friendly):

- **Temporal** (MIT) — most proven; its worker-poll model *is* the agent
  phone-home topology; heavy ops (a separate stateful cluster); Python
  worker-affinity is DIY.
- **Hatchet** (MIT) — Postgres-only, distributed workers, genuine durable
  execution; matches the "lightweight / one real DB" instinct; youngest and
  least battle-tested.
- **DBOS** (MIT) — Postgres-only, expressive Python; but an **in-process
  library**, not a distributed worker system — fights the agent model; must be
  wrapped.
- Ruled out on license for an OSS core: **Restate** (BSL), **Inngest** (SSPL),
  **Windmill / LittleHorse** (AGPL).

**Working recommendation:** prototype against **Temporal** (fastest proven path
to a vertical slice; the slice's real content is engine-independent), keep the
engine behind a **thin seam**, and treat **Hatchet** as the target-state
front-runner to beat it.

#### Requirement — local testability (hard requirement)

Scripted promotion **and** verification processes MUST be runnable and testable
**locally**, without rolling them out to a live control plane. "Deploy to find
out if it works" is unacceptable. Implications:

- **stdlib functions are plain, directly-callable units** usable in a local
  harness / ordinary unit tests.
- **The execution engine must offer a local or time-skipping test mode** — this
  is now a real *selection criterion*, favoring engines with strong test
  frameworks or library-style local execution.
- **Side effects support a dry-run / fixture mode** (git push, etc.) so a process
  can be exercised against throwaway repos without mutating anything real.
- The single-synchronous-FS-step model *helps*: an FS step is just a function you
  can call in a test with a temp dir and a throwaway repo.

#### Operational advantage — clean, per-process observability (major selling point)

A promotion process is a **discrete, isolated execution** — its own sandboxed
container (mandatory anyway for untrusted code) tracked as its own workflow by
the durable engine. Three wins fall out **for free** from decisions already made:

- **Uncluttered logs.** Each promotion has its *own* log stream. Contrast 1.x,
  where every promotion's logs interleave with countless others inside a shared
  controller container and isolating one promotion's output is log-correlation
  archaeology. Here, a promotion's logs are simply *that promotion's logs*.
- **First-class execution history.** Beyond raw logs, the durable engine records
  each step, its inputs/outputs, timing, and retries — so troubleshooting is
  "read the execution," not "grep the fleet."
- **Reproducibility.** Because the same process runs the same way locally (the
  local-testability requirement), the logs and behavior you debug on a laptop
  *match* production.

This is the **same core sin, inverted:** 1.x observability is poor *because*
everything is crammed into shared controllers; decoupling execution into isolated
per-process units makes troubleshooting substantially easier — at no extra cost,
since it falls straight out of the sandbox requirement plus the durable engine.
*Honest caveat:* isolated logs must still be shipped/aggregated to the control
plane for the single pane of glass — standard, but not free.

---

## 6. The hard problem — status *between* systems

Two questions, kept strictly apart:

- **Status *within* one system** — solved. An Application (or Stage) has status;
  it can work however we want.
- **Status *between* systems** (margo → kargo 2.0) — genuinely hard.

### 6.1 Why it's hard

- **The forward path is anonymous; the backward path is addressed.** kargo
  writes desired state to a source of truth and forgets — it neither knows nor
  cares which margo Applications consume it. A margo Application syncs content
  without knowing which Stage produced it. The rendezvous (repo) works *because*
  the forward direction needs no addressing.
- But **status is meaningless without addressing** — it must be *about* a
  specific Application and *for* a specific interested Stage. The rendezvous, by
  design, **destroys the very relationship that status requires** (mutual
  blindness).
- **Implication:** addressed between-system status has an **irreducible coupling
  cost** — *someone* must hold the Stage↔Application relationship, and holding it
  *is* coupling. Kargo 1.x pays this by having kargo know its Argo apps (the
  inverse is not true).

### 6.2 Working answer — `reply-to` + `correlation-id`

kargo writes, *alongside* the desired state, a callback instruction:
essentially, *"whoever syncs this, reach me **here** about token **T** when
you're done."* This is the `reply-to` + `correlation-id` pattern from decoupled
messaging (AMQP/JMS).

- **It dissolves the blindness** rather than accepting it: each realizer
  *discovers the return address in the content it already reads* and announces
  itself back against T. margo still has zero concept of a "Stage" — it just
  honors a generic "if there's a reply-to, report status to it" contract. The
  correlation rides *with the payload*, so neither system models the other's
  domain.
- Coupling drops from **entity-level knowledge** (kargo knows apps) to a
  **generic protocol contract** (both honor reply-to).
- **Bonus:** fan-out discovers itself — if three Applications sync the same
  desired state, three reports come back on T and kargo aggregates. kargo need
  not know *how many* realizers exist.

**Friction (flagged, not solved):**

- **Honored at the margo *control plane*, not the agent** — agents are
  firewalled / outbound-only; the control plane already collects their status
  and can relay control-plane-to-control-plane.
- **It is sync-*metadata*, not content** — the reply-to must ride *beside*
  desired state in a channel the syncer reads but never *applies* ("call kargo
  here" must not land in the cluster).
- **Reachability fallback** — if margo can't reach kargo (firewalls between
  control planes), degrade to *kargo polls margo for "status of token T."* Still
  better than 1.x: query by token, no app topology needed.
- **Security** — a reply-to is a "phone this address" instruction sitting in a
  repo. Needs signing both ways, or it's an SSRF lever (margo POSTing anywhere)
  and a status-spoofing lever (forged reports to kargo).

### 6.3 The "one roof" alternative

If margo and kargo 2.0 are vertically integrated (option (c) below), the problem
collapses from a **network protocol across a trust boundary** to an **internal
correlation within shared state.** Both sides' data already lands in the same
store, so the control plane can **join on `(repo, revision)`** — which works
*because* rendering is upstream of the rendezvous, so the revision an Application
reports is exactly the revision kargo wrote. The `reply-to` earns its keep only
when a system boundary severs that shared visibility.

> **One roof → correlation is a join; two roofs → correlation is a protocol.**
> Same problem, wildly different difficulty. This is a legitimate gravitational
> pull toward (c) — to weigh, not to decide prematurely.

---

## 7. Open & Deferred Decisions

Deliberately unresolved, to let the answer **emerge from the design** rather than
constrain it:

1. **margo ↔ kargo 2.0 relationship:**
   - **(a)** one platform, shared control-plane/runtime, both as modules (risk:
     rebuilding a monolith);
   - **(b)** two independent systems, same architecture, interoperating *only*
     through the rendezvous (+ margo managing kargo's config);
   - **(c)** margo as the substrate, kargo 2.0 built as an application on top of
     margo's primitives.
   - The distinct-agent-roles insight nudges away from a naive (a); the
     between-systems status problem exerts a pull toward (c). Not decided.
2. **Pipeline: explicit first-class concept vs. inferred/emergent DAG.**
3. **Persistence technology** — relational vs. other; follow the design.
4. **Between-systems status — final mechanism.** `reply-to`/`correlation-id` is
   the working answer; it may simplify to an internal join under (c).
5. **Execution model — durable execution engine choice.** Filesystem strategy
   (single synchronous FS-burst step; source-of-truth as the durable externalized
   filesystem) and local testability are now *decided requirements* (§5.6). What
   remains open is the **engine**: prototype against Temporal, keep it behind a
   thin seam, treat Hatchet as the target-state front-runner. The scripted-stdlib
   authoring surface still needs detailed design.
6. **Prototype scope** — what the first prototype must *prove* is not yet
   decided.

Additionally, these areas are **identified but not yet designed** — surfaced here
so we don't lose sight of them:

1. **Authorization & identity** — *designed; see §3.5* (grant primitive, native
   principals, single chokepoint, inline evaluation, strict no-escape-hatch
   anti-escalation). Remaining sub-choices: the inline evaluator (small
   hand-rolled vs. embeddable library like Cedar/Casbin) and agent-enrollment
   mechanics.
2. **Query / visibility model** — the API/query experience that finally delivers
   cross-everything visibility (the payoff of ditching etcd).
3. **Credentials & secrets** — how target and promotion-process credentials are
   provisioned, scoped per project, and delivered to the agents that need them.
4. **stdlib authoring surface** — the concrete Python API a promotion or
   verification process is written against.

---

## 8. Glossary

- **margo** — successor to Argo CD; the delivery/sync engine.
- **kargo 2.0** — successor to Kargo; the promotion/orchestration engine.
- **Source of truth** — a format-blind, versioned content provider (git, OCI,
  S3, …). The rendezvous point between the two systems.
- **Syncer** — interprets + applies content to a target and reports drift/health
  (k8s, tofu, …).
- **Application** (margo) — declarative resource binding a source of truth to an
  execution environment; carries status.
- **Agent** — a placement of a syncer/executor inside the trust/network boundary
  that holds the target's credentials and can reach it. Reconciliation and
  execution always run here.
- **Freight** — an immutable, thumbprinted box of versioned-artifact references
  that moves through the pipeline as a unit.
- **Stage** — slots (N kinds of required freight) + instructions (a process
  producing desired state). A promotion target, not necessarily an environment.
- **Origin** — where freight is born: warehouse (reference), junction
  (envisioned), others.
- **Junction** — an origin kind that repackages freight from several sources into
  a new *kind* of freight.
- **Promotion** — an imperative state change that fills a stage's slot(s).
- **Verification** — state *validation*, separate from promotion; re-runnable at
  any time.
- **Durable workflow engine** — a runtime that persists execution state
  continuously so a single logical execution survives crashes/restarts and
  suspends/resumes across arbitrary time.
