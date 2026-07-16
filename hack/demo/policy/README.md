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
| `kargo.lib.helpers` | -- | Building blocks for custom policies (e.g. `is_semver_patch`) |

A project (`ProjectConfig spec.customPolicy`) and the operator
(`ClusterConfig spec.customPolicy`, applied to every project) may extend
the default policy with custom rules that **compose into** -- never
replace -- it. A custom policy contains *only rules*: the package
declaration and the standard library imports are prepended automatically,
so pasting a single rule just works. The default policy gathers any
`violation` a custom policy contributes, and the exclusions block consults
its `exclusions_bypass(e)` predicate. See the commented examples in
`10-projectconfig.yaml` and `40-clusterconfig.yaml`, and the
schema/authoring reference in `pkg/promotion/dispatch/policy/README.md`.

Promotion classes are inferred per Promotion: `auto-forward` (created by the
system), `manual-forward` (created by a user), and `rollback` (annotated
`kargo.akuity.io/rollback: "true"`). There is no built-in hotfix concept at
all -- what counts as a hotfix is defined by whoever writes the custom
policy (typically the operator, cluster-wide), with the standard library
supplying only the semver building block (`helpers.is_semver_patch`).

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

The ClusterConfig lists have server-side-apply merge semantics (keyed by
`name`), so every ClusterConfig change in this demo uses
`kubectl apply --server-side` with a per-concern field manager -- entries
merge into any existing ClusterConfig and each manager owns (and can
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
governs it, though a rate limit does -- see Scenario 4). Once freight is
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
target a particular destination server -- see Scenario 5.

## Scenario 3 -- custom policies: an operator hotfix lane through the freeze

With the freeze from Scenario 2 still active, let the **operator** open a
hotfix lane cluster-wide: uncomment the `customPolicy:` block in
`40-clusterconfig.yaml` and re-apply it (same command as Setup). The rules
compose into every project's dispatch decision (the `kargo.cluster`
package and standard imports are prepended automatically); nothing is
replaced. The policy defines hotfix semantics *in the operator's own
terms* -- every image shared with what the Stage last promoted is a semver
patch-only increment (`helpers.is_semver_patch`) -- and overrides
`exclusions_bypass(e)` with it. There is no hotfix concept in the standard
library to fight with.

The **project** can contribute its own rules independently: uncomment the
`customPolicy:` block in `10-projectconfig.yaml` and re-apply. It adds a
data-driven `violation`: because the Project is labeled `compliance: pci`
(see `00-project.yaml`), manual promotions must carry a `change-ticket`
annotation -- without one they park with the rule's message.

Now promote a **patch-only** bump of what `uat` is currently running (e.g.
`1.29.1` over `1.29.0`), with a change ticket -- it dispatches through the
freeze. A minor bump (e.g. `1.29.x` over `1.28.x`) stays blocked by the
freeze even with a ticket:

```shell
# Create a manual promotion for the patch-bump freight (see Scenario 2 for
# the kubectl create pattern; no rollback annotation this time). Include
# the change-ticket annotation to satisfy the PCI rule:
#   metadata:
#     annotations:
#       change-ticket: CHG-1234
```

Lift the freeze when done -- re-apply as the same field manager with the
entry omitted, and server-side apply removes what that manager owned:

```shell
kubectl apply --server-side --field-manager=incident-freeze -f - <<EOF
apiVersion: kargo.akuity.io/v1alpha1
kind: ClusterConfig
metadata:
  name: cluster
EOF
```

## Scenario 4 -- rolling-window rate limit

`test` allows at most 1 automatic dispatch per 2 minutes. Make the Warehouse
produce two pieces of freight in quick succession (create freight from the
UI for two discovered tags, or tighten/loosen `semverConstraint` and refresh
the Warehouse). The first auto-promotion dispatches; the second parks with a
rate-limit message and dispatches on its own when the first ages out of the
2-minute window. Manual promotions and rollbacks are never rate-limited.

## Scenario 5 -- cluster maintenance freezes only affected Stages

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
promote normally. End the maintenance and the held promotion dispatches:

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
