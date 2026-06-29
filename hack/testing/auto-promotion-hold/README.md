# Auto-Promotion Hold Demo

A manual test harness for the auto-promotion hold feature. Seeds two Stages
to exercise and review hold behavior end-to-end.

## Prerequisites

A running Tilt dev environment (`make hack-tilt-up`).

## Setup

```shell
./hack/testing/auto-promotion-hold/apply.sh
```

Re-run the same command to reset between scenario runs. It deletes all
Promotions, Stages, and ProjectConfig before re-seeding.

This creates the `auto-promotion-hold` project with two Stages:

- **`single-origin-hold`** — single-origin Stage with auto-promotion enabled
  and no active hold. Use this to trigger a hold manually and observe the
  promote-by-origin resume path.
- **`multi-origin-holds`** — multi-origin Stage seeded with active holds on
  both origins. Use this to review how the DAG and Stage detail views render
  held origins, and to verify per-origin isolation.

## Scenarios

### 1. Trigger a hold (single-origin-hold)

Promote a non-candidate Freight. The webhook stamps
`kargo.akuity.io/auto-promotion-hold` on the Promotion. When the Promotion
succeeds, the Stage controller records the hold in `status.autoPromotionHolds`
and auto-promotion for that origin stops.

Because all Freight in this harness share the same creation timestamp, the
candidate is determined by lexical name order (descending hash). `frontend-v001`
(`39f8209...`) is always a safe non-candidate choice.

```shell
# Via CLI (using alias)
kargo promote --project auto-promotion-hold --stage single-origin-hold \
  --freight-alias frontend-v001

# Via kubectl (using content-addressed name; Kargo enforces SHA-1 naming)
kubectl apply -f - <<YAML
apiVersion: kargo.akuity.io/v1alpha1
kind: Promotion
metadata:
  name: my-hold-test
  namespace: auto-promotion-hold
spec:
  stage: single-origin-hold
  freight: 39f8209d87b5222d5dbebf5e6f1d9a54fe7d7b52
YAML
```

Verify the hold appears:

```shell
kubectl get stage single-origin-hold -n auto-promotion-hold \
  -o jsonpath='{.status.autoPromotionHolds}' | jq .
```

### 2. Resume via promote-by-origin (race-free)

Promoting by origin resolves to the current candidate server-side, stamps
`kargo.akuity.io/auto-promotion-release`, and clears the hold when the
Promotion succeeds.

```shell
# Via CLI
kargo promote --project auto-promotion-hold --stage single-origin-hold \
  --origin Warehouse/auto-hold

# Via kubectl (spec.origin; webhook resolves to candidate Freight)
kubectl apply -f - <<YAML
apiVersion: kargo.akuity.io/v1alpha1
kind: Promotion
metadata:
  name: my-resume-test
  namespace: auto-promotion-hold
spec:
  stage: single-origin-hold
  origin:
    kind: Warehouse
    name: auto-hold
YAML
```

After the Promotion succeeds, `status.autoPromotionHolds` should be empty and
auto-promotion resumes.

### 3. Per-origin isolation (multi-origin-holds)

The `multi-origin-holds` Stage starts with holds on both origins. Releasing one
origin does not affect the other. Promote by origin for `Warehouse/auto-hold`
only and confirm `Warehouse/auto-hold-api` remains held:

```shell
kargo promote --project auto-promotion-hold --stage multi-origin-holds \
  --origin Warehouse/auto-hold
```

### 4. Verify hold survives Promotion GC

Delete the establishing Promotion and confirm the hold persists in status.
`apply.sh` prints the actual Promotion names at the end; copy the frontend
name from there, or look it up by annotation:

```shell
hold_promo=$(kubectl get promotions -n auto-promotion-hold -o json | \
  jq -r '
    .items[]
    | select(
        .metadata.annotations["kargo.akuity.io/auto-promotion-hold"] == "Warehouse/auto-hold"
        and .status.phase == "Succeeded"
      )
    | .metadata.name' | head -1)
kubectl delete promotion "${hold_promo}" -n auto-promotion-hold
kubectl get stage multi-origin-holds -n auto-promotion-hold \
  -o jsonpath='{.status.autoPromotionHolds}' | jq .
```
