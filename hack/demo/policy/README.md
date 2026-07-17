# Promotion Dispatch Policy Demo (OPA spike)

This demo exercises the OPA-driven **promotion dispatch gate**: Promotions
are created eagerly (auto-promotion or manual) and accumulate in a per-Stage
queue in the `Pending` phase, but the Stage controller consults a dispatch
policy before acknowledging one to run. A held Promotion stays `Pending`,
carries a `PromotionBlocked` event explaining why, and dispatches on its own
when the policy allows (the policy returns the "when" as a requeue hint).

The default policy is composed from standard, data-driven Rego blocks:

| Block | Data source | Behavior |
|---|---|---|
| `kargo.lib.windows` | `ProjectConfig` `spec.promotionWindows` | Forward promotions to a governed Stage dispatch only inside a recurring (RRULE) window |
| `kargo.lib.exclusions` | `ClusterConfig` `spec.promotionExclusions` | System-wide blackouts, scoped by promotion class (`no-promotions`, `no-forward`, `no-auto`) and optionally by Argo CD destination server |
| `kargo.lib.ratelimit` | `ProjectConfig` `spec.rateLimits` | Rolling window: at most N automatic dispatches per trailing window |
| `kargo.lib` | -- | Building blocks for custom policies (`kargo.is_forward`, `kargo.is_semver_patch`) |

A project (`ProjectConfig spec.customPolicy`) and the operator
(`ClusterConfig spec.customPolicy`, applied to every project) may extend
the default policy with custom rules that **compose into** -- never
replace -- it. A custom policy contains *only rules*: the package
declaration and the standard library imports are prepended automatically,
so pasting a single rule just works. The default policy gathers any
`violation` a custom policy contributes, and the exclusions block consults
its `exclusions_bypass(e)` predicate. See the schema/authoring reference in
`pkg/promotion/dispatch/policy/README.md`.

This demo ships **one custom rule active from the start**: the operator's
`ClusterConfig` policy (`40-clusterconfig.yaml`) requires a `change-ticket`
annotation on manual promotions in PCI-labeled projects. It is a standing
compliance guardrail, so custom-policy composition is in force throughout;
Scenario 5 shows it biting on its own. The commented example blocks in
`10-projectconfig.yaml` and `40-clusterconfig.yaml` layer on further rules
in Scenarios 6-7.

Promotion classes are inferred per Promotion: `auto-forward` (created by the
system), `manual-forward` (created by a user), and `rollback` (annotated
`kargo.akuity.io/rollback: "true"`). There is no built-in hotfix concept at
all -- what counts as a hotfix is defined by whoever writes the custom
policy (typically the operator, cluster-wide), with the standard library
supplying only the semver building block (`kargo.is_semver_patch`).

The commented example blocks -- and the active PCI rule -- are kept working
by `demo_test.go`, which extracts them and runs them against the real policy
engine (`go test ./hack/demo/policy`).

## Prerequisites

A running Tilt dev environment (`make hack-tilt-up`) built from this branch:

```shell
# After (re)building: apply the regenerated CRDs, then rebuild the controller.
kubectl apply --server-side -f charts/kargo/resources/crds/kargo.akuity.io_projectconfigs.yaml
kubectl apply --server-side -f charts/kargo/resources/crds/kargo.akuity.io_clusterconfigs.yaml
hack/bin/tilt trigger back-end-compile && hack/bin/tilt wait --for=condition=Ready uiresource/back-end-compile
hack/bin/tilt trigger controller && hack/bin/tilt wait --for=condition=Ready uiresource/controller
```

## Setup

```shell
kubectl apply -f hack/demo/policy/00-project.yaml
kubectl apply -f hack/demo/policy/10-projectconfig.yaml
kubectl apply -f hack/demo/policy/20-warehouse.yaml
kubectl apply -f hack/demo/policy/30-stages.yaml
```

The hotfix scenario seeds one extra Freight of its own (`15-freight.yaml`);
setup deliberately leaves it out so Scenarios 1-5 run on only the Warehouse's
discovered freight.

The `ClusterConfig` ships the always-on PCI rule (`spec.customPolicy`) and
an inert holiday freeze. Its lists have server-side-apply merge semantics
(keyed by `name`), so every `ClusterConfig` change in this demo uses
`kubectl apply --server-side` with a per-concern field manager -- entries
merge into any existing `ClusterConfig` and each manager owns (and can
remove) only its own:

```shell
kubectl apply --server-side --field-manager=policy-demo -f hack/demo/policy/40-clusterconfig.yaml
```

Watch the action from two terminals:

```shell
kubectl get promotions -n policy-demo -w
kubectl get events -n policy-demo --field-selector reason=PromotionBlocked -w
```

## Scenario 1 -- promotion window holds an auto-promotion

The Warehouse discovers nginx images; `test` auto-promotes freely (no window
governs it, though a rate limit does -- see Scenario 3). Once freight is
verified in `test`, auto-promotion creates a Promotion for `uat` -- which is
governed by the `uat-evenings` window (weekdays 18:00-23:00 US Pacific).
Outside that window the Promotion parks:

```shell
kubectl get promotions -n policy-demo
# NAME       SHARD   STAGE   FREIGHT   PHASE     AGE
# uat.01...          uat     2f0f...   Pending   1m

kubectl describe stage uat -n policy-demo | grep -A4 'Type:.*Promoting'
# Status: False, Reason: DispatchBlocked,
# Message: outside all promotion windows; next window opens at ...
```

Now open the window by editing it to span the present (or just watch at
18:00 Pacific):

```shell
kubectl edit projectconfig policy-demo -n policy-demo
# e.g. change start/end to bracket the current time in UTC:
#   recurrence: FREQ=DAILY
#   start: "00:00"
#   end: "23:59"
#   location: UTC
```

The ProjectConfig watch re-enqueues the Stage immediately and the held
Promotion dispatches. (Even without the watch, the controller requeues
itself at the window boundary the policy reported.)

## Scenario 2 -- system-wide freeze; rollback passes through

Activate a `no-forward` exclusion spanning now (merged in alongside any
other exclusions, under its own field manager):

```shell
kubectl apply --server-side --field-manager=incident-freeze -f - <<EOF
apiVersion: kargo.akuity.io/v1alpha1
kind: ClusterConfig
metadata:
  name: cluster
spec:
  promotionExclusions:
  - name: incident-freeze
    start: "2026-01-01T00:00:00Z"
    end: "2036-01-01T00:00:00Z"
    scope: no-forward
EOF
```

Every forward promotion in every project now parks (auto and manual alike),
with events explaining the freeze. Recovery is exempt: a rollback dispatches
right through it. Promote a previously-verified (older) piece of freight as
a rollback:

```shell
FREIGHT=$(kubectl get freight -n policy-demo --sort-by=.metadata.creationTimestamp -o jsonpath='{.items[0].metadata.name}')
cat <<EOF | kubectl create -f -
apiVersion: kargo.akuity.io/v1alpha1
kind: Promotion
metadata:
  generateName: rollback-
  namespace: policy-demo
  annotations:
    kargo.akuity.io/rollback: "true"
spec:
  stage: uat
  freight: ${FREIGHT}
  steps:
  - uses: compose-output
    as: note
    config:
      promoted: rollback
EOF
```

To freeze *everything* (an incident-forensics freeze that even blocks
rollbacks), use `scope: no-promotions`. To pause only automation while
humans retain control, use `scope: no-auto` and promote manually.

An exclusion can also be narrowed to Stages whose Argo CD Applications
target a particular destination server -- see Scenario 4.

Lift the freeze before moving on -- re-apply as the same field manager
with the entry omitted, and server-side apply removes what that manager
owned:

```shell
kubectl apply --server-side --field-manager=incident-freeze -f - <<EOF
apiVersion: kargo.akuity.io/v1alpha1
kind: ClusterConfig
metadata:
  name: cluster
EOF
```

## Scenario 3 -- rolling-window rate limit

`test` allows at most 1 automatic dispatch per 2 minutes. Make the Warehouse
produce two pieces of freight in quick succession (create freight from the
UI for two discovered tags, or tighten/loosen `semverConstraint` and refresh
the Warehouse). The first auto-promotion dispatches; the second parks with a
rate-limit message and dispatches on its own when the first ages out of the
2-minute window. Manual promotions and rollbacks are never rate-limited.

## Scenario 4 -- cluster maintenance freezes only affected Stages

An exclusion narrowed with `argocdServers` freezes only Stages whose
referenced Argo CD Applications target one of the named destination servers
(by URL or name) -- the "this cluster is under maintenance" use case. The
linkage is the `kargo.akuity.io/authorized-stage` annotation on the
Application.

Create an Application for the `prod` Stage targeting the local cluster:

```shell
kubectl apply -f hack/demo/policy/50-application.yaml
```

Declare maintenance on that cluster:

```shell
kubectl apply --server-side --field-manager=cluster-maintenance -f - <<EOF
apiVersion: kargo.akuity.io/v1alpha1
kind: ClusterConfig
metadata:
  name: cluster
spec:
  promotionExclusions:
  - name: cluster-maintenance
    start: "2026-01-01T00:00:00Z"
    end: "2036-01-01T00:00:00Z"
    scope: no-promotions
    argocdServers:
    - https://kubernetes.default.svc
EOF
```

A manual promotion to `prod` now parks (its Application targets the server
under maintenance), while `test` and `uat` -- which have no Applications --
promote normally. Because the manual promotion is `manual-forward`, the
always-on PCI rule (Scenario 5) also applies, so give it a `change-ticket`
annotation -- then the maintenance exclusion is the only thing left holding
it. End the maintenance and the held promotion dispatches:

```shell
kubectl apply --server-side --field-manager=cluster-maintenance -f - <<EOF
apiVersion: kargo.akuity.io/v1alpha1
kind: ClusterConfig
metadata:
  name: cluster
EOF
```

This scenario requires the controller's Argo CD integration (enabled by
default under Tilt); without it, `input.applications` is always empty and
server-scoped exclusions never match.

## Scenario 5 -- custom policy composition: the always-on PCI rule

The operator's `ClusterConfig` policy composes into every project's dispatch
decision (the `kargo.cluster` package and standard imports are prepended
automatically); nothing is replaced. This demo's baseline rule reads the
Project's own metadata -- because `policy-demo` is labeled `compliance: pci`
(see `00-project.yaml`), manual promotions must carry a `change-ticket`
annotation.

Create a manual promotion to `prod` **without** the annotation -- it parks
on the rule's message:

```shell
FREIGHT=$(kubectl get freight -n policy-demo --sort-by=.metadata.creationTimestamp -o jsonpath='{.items[-1].metadata.name}')
cat <<EOF | kubectl create -f -
apiVersion: kargo.akuity.io/v1alpha1
kind: Promotion
metadata:
  generateName: prod-noticket-
  namespace: policy-demo
spec:
  stage: prod
  freight: ${FREIGHT}
  steps:
  - uses: compose-output
    as: note
    config:
      promoted: no-ticket
EOF

kubectl describe stage prod -n policy-demo | grep -A2 'Type:.*Promoting'
# Message: ... PCI-compliant projects require a change-ticket annotation on
# manual promotions
```

Create the same promotion with `annotations: { change-ticket: CHG-1234 }`
and it dispatches. Auto-promotions and rollbacks are unaffected (the rule
matches `manual-forward` only), which is why Scenarios 1-2 never tripped it.

## Scenario 6 -- custom policies: a hotfix lane through the holiday freeze

A hotfix is freight that reaches `prod`, so first seed and stage it. Seed the
patch build and walk it up the pipeline to `uat` (where `prod` draws its
freight) -- Kargo only lets a Stage promote freight verified upstream, so a
hotfix has to travel the normal path before it can jump the freeze:

```shell
kubectl create -f hack/demo/policy/15-freight.yaml
PATCH=$(kubectl get freight -n policy-demo -o jsonpath='{range .items[?(@.images[0].tag=="1.31.4")]}{.metadata.name}{end}')

# Promote 1.31.4 through test then uat. These are manual-forward promotions,
# so the always-on PCI rule needs a change-ticket; uat is windowed, so its
# window must be open (as in Scenario 1).
for STAGE in test uat; do
cat <<EOF | kubectl create -f -
apiVersion: kargo.akuity.io/v1alpha1
kind: Promotion
metadata:
  generateName: ${STAGE}-hotfix-
  namespace: policy-demo
  annotations:
    change-ticket: CHG-9000
spec:
  stage: ${STAGE}
  freight: ${PATCH}
  steps:
  - uses: compose-output
    as: note
    config:
      promoted: hotfix-staging
EOF
done
```

Now bring the holiday freeze into effect: edit `start`/`end` of the
`holiday-freeze` exclusion in `40-clusterconfig.yaml` to bracket the
present, and re-apply it (same command as Setup). Every forward promotion
parks, as in Scenario 2.

Then let the **operator** extend its policy cluster-wide: replace the active
`customPolicy:` in `40-clusterconfig.yaml` with the expanded block currently
commented beneath it (PCI rule **plus** a hotfix bypass) and re-apply once
more. It adds:

- A **hotfix lane** through the holiday freeze: hotfix semantics defined
  *in the operator's own terms* -- every image shared with what the Stage
  last promoted is a semver patch-only increment
  (`kargo.is_semver_patch`) -- overriding `exclusions_bypass(e)`. The
  bypass names the `holiday-freeze` exclusion specifically: a *planned*
  freeze admits hotfixes, while an incident freeze like Scenario 2's would
  still hold everything. There is no hotfix concept in the standard
  library to fight with.
- The **PCI compliance mandate** from Scenario 5, retained.

With `prod` on `1.31.3` and the freeze active, promote the `1.31.4` patch
bump to `prod` with a change ticket -- it is a semver patch over what `prod`
last ran, so the hotfix lane dispatches it straight through the freeze:

```shell
cat <<EOF | kubectl create -f -
apiVersion: kargo.akuity.io/v1alpha1
kind: Promotion
metadata:
  generateName: prod-hotfix-
  namespace: policy-demo
  annotations:
    change-ticket: CHG-9001
spec:
  stage: prod
  freight: ${PATCH}
  steps:
  - uses: compose-output
    as: note
    config:
      promoted: hotfix
EOF
```

A *minor* bump would **not** be a hotfix and stays frozen even with a ticket;
`demo_test.go` verifies that contrast (and the incident-freeze case) directly
against the engine.

When done, restore the holiday freeze's future dates in
`40-clusterconfig.yaml` and re-apply (the `policy-demo` field manager owns
the entry, so the dates simply revert; the freeze goes inert again). You can
also revert the `customPolicy:` to the baseline PCI rule.

## Scenario 7 -- project self-service: a prod-approval rule

A **project** can contribute its own rules independently of the operator:
uncomment the `customPolicy:` block in `10-projectconfig.yaml` and re-apply.
It requires an `approved-by` annotation on any forward promotion to `prod`
(rollbacks are exempt via `kargo.is_forward`).

Create a forward promotion to `prod` with a change ticket (to satisfy the
operator's PCI rule) but **no** `approved-by` -- it parks on the project's
rule:

```shell
FREIGHT=$(kubectl get freight -n policy-demo --sort-by=.metadata.creationTimestamp -o jsonpath='{.items[-1].metadata.name}')
cat <<EOF | kubectl create -f -
apiVersion: kargo.akuity.io/v1alpha1
kind: Promotion
metadata:
  generateName: prod-unapproved-
  namespace: policy-demo
  annotations:
    change-ticket: CHG-1234
spec:
  stage: prod
  freight: ${FREIGHT}
  steps:
  - uses: compose-output
    as: note
    config:
      promoted: unapproved
EOF

kubectl describe stage prod -n policy-demo | grep -A2 'Type:.*Promoting'
# Message: ... prod promotions require an approved-by annotation
```

Add `approved-by: eron` alongside the change ticket and it dispatches. A
rollback needs neither annotation -- `kargo.is_forward` excludes it from the
rule.

## Troubleshooting

- A broken `customPolicy` (project- or cluster-level) **fails closed**:
  nothing dispatches, the Stage's `Promoting` condition reports
  `DispatchPolicyError`, and each held Promotion gets a
  `PromotionPolicyError` event. Fix or remove the rules. A source that
  declares its own `package` is rejected the same way (custom policies
  contain rules only). Compile-error line numbers are offset by the
  prepended header.
- The policy engine sees Argo CD Applications only when the controller's
  Argo CD integration is enabled; without it, server-scoped exclusions never
  match (unscoped exclusions still work).
- `kubectl create`-ing a Promotion requires `spec.steps` inline (the webhook
  does not inflate them from the Stage's promotionTemplate).

## Cleanup

```shell
kubectl delete -f hack/demo/policy/50-application.yaml --ignore-not-found
kubectl delete freight -n policy-demo -l kargo.akuity.io/alias=hotfix-build --ignore-not-found
kubectl delete -f hack/demo/policy/30-stages.yaml -f hack/demo/policy/20-warehouse.yaml -f hack/demo/policy/10-projectconfig.yaml -f hack/demo/policy/00-project.yaml
# Relinquish each demo field manager's ClusterConfig entries:
for fm in policy-demo incident-freeze cluster-maintenance; do
  kubectl apply --server-side --field-manager=$fm -f - <<EOF
apiVersion: kargo.akuity.io/v1alpha1
kind: ClusterConfig
metadata:
  name: cluster
EOF
done
```
