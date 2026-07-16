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
| `kargo.lib.windows` | `ProjectConfig` `spec.policy.promotionWindows` | Forward promotions to a governed Stage dispatch only inside a recurring (RRULE) window |
| `kargo.lib.exclusions` | `ClusterConfig` `spec.promotionExclusions` | System-wide blackouts, scoped by promotion class (`no-promotions`, `no-forward`, `no-auto`) and optionally by Argo CD destination server |
| `kargo.lib.ratelimit` | `ProjectConfig` `spec.policy.rateLimits` | Rolling window: at most N automatic dispatches per trailing window |
| `kargo.lib.helpers` | -- | Building blocks for custom policies (e.g. `is_hotfix`) |

A project may replace the default with its own module
(`spec.policy.custom`), importing whichever blocks it wants and adding its
own rules. See the commented example in `10-projectconfig.yaml`.

Promotion classes are inferred per Promotion: `auto-forward` (created by the
system), `manual-forward` (created by a user), and `rollback` (annotated
`kargo.akuity.io/rollback: "true"`). There is no built-in hotfix class --
hotfix semantics are a custom-policy concern, built on `helpers.is_hotfix`
(every shared image is a semver patch-only increment over what the Stage
last promoted).

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

For the ClusterConfig: apply `40-clusterconfig.yaml` only if the cluster has
no ClusterConfig yet; otherwise patch the exclusions in:

```shell
kubectl apply -f hack/demo/policy/40-clusterconfig.yaml   # if none exists
# ...or...
kubectl patch clusterconfig cluster --type merge -p '
spec:
  promotionExclusions:
  - name: holiday-freeze
    start: "2026-12-20T00:00:00Z"
    end: "2027-01-02T00:00:00Z"
    scope: no-forward
'
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

Activate a `no-forward` exclusion spanning now:

```shell
kubectl patch clusterconfig cluster --type merge -p '
spec:
  promotionExclusions:
  - name: incident-freeze
    start: "2026-01-01T00:00:00Z"
    end: "2036-01-01T00:00:00Z"
    scope: no-forward
'
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
target a particular destination (`argocdServers: ["https://prod.example.com"]`);
Stage-to-Application linkage uses the `kargo.akuity.io/authorized-stage`
annotation on the Application.

## Scenario 3 -- custom policy: hotfixes bypass the freeze

With the freeze from Scenario 2 still active, enable the custom policy:
uncomment the `custom:` block in `10-projectconfig.yaml` and re-apply. The
custom module replaces the default, composes the same library blocks, but
filters exclusion violations through `helpers.is_hotfix`.

Now promote a **patch-only** bump of what `uat` is currently running (e.g.
`1.29.1` over `1.29.0`) -- it dispatches through the freeze. A minor bump
(e.g. `1.29.x` over `1.28.x`) stays blocked:

```shell
# Create a manual promotion for the patch-bump freight (see Scenario 2 for
# the kubectl create pattern; no rollback annotation this time).
```

The same custom module also refuses to auto-promote prerelease image tags
into `prod` -- a purely project-specific rule unioned into the same
violation set.

Lift the freeze when done:

```shell
kubectl patch clusterconfig cluster --type json -p '[{"op": "remove", "path": "/spec/promotionExclusions"}]'
```

## Scenario 4 -- rolling-window rate limit

`test` allows at most 1 automatic dispatch per 2 minutes. Make the Warehouse
produce two pieces of freight in quick succession (create freight from the
UI for two discovered tags, or tighten/loosen `semverConstraint` and refresh
the Warehouse). The first auto-promotion dispatches; the second parks with a
rate-limit message and dispatches on its own when the first ages out of the
2-minute window. Manual promotions and rollbacks are never rate-limited.

## Troubleshooting

- A broken `spec.policy.custom` **fails closed**: nothing dispatches, the
  Stage's `Promoting` condition reports `DispatchPolicyError`, and each held
  Promotion gets a `PromotionPolicyError` event. Fix or remove the module.
- The policy engine sees Argo CD Applications only when the controller's
  Argo CD integration is enabled; without it, server-scoped exclusions never
  match (unscoped exclusions still work).
- `kubectl create`-ing a Promotion requires `spec.steps` inline (the webhook
  does not inflate them from the Stage's promotionTemplate).

## Cleanup

```shell
kubectl delete -f hack/demo/policy/30-stages.yaml -f hack/demo/policy/20-warehouse.yaml -f hack/demo/policy/10-projectconfig.yaml -f hack/demo/policy/00-project.yaml
kubectl patch clusterconfig cluster --type json -p '[{"op": "remove", "path": "/spec/promotionExclusions"}]'
```
