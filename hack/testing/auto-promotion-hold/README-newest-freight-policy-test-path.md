# Auto-Promotion Hold — NewestFreight Policy Harness

Manual test harness for the auto-promotion hold feature under the
**NewestFreight** selection policy.

See [README-match-upstream-policy-test-path.md](README-match-upstream-policy-test-path.md)
for the MatchUpstream equivalent.

## Prerequisites

A running Tilt dev environment (`make hack-tilt-up`).

## Setup

```shell
./hack/testing/auto-promotion-hold/apply-newest-freight-policy-harness.sh
```

Creates two Stages in the `auto-promotion-hold` project:

- **`single-origin-hold`** — single-origin Stage with auto-promotion enabled
  and no active hold. Use this to trigger holds, resume them, and observe
  auto-promotion behavior.
- **`multi-origin-holds`** — multi-origin Stage seeded with active holds on
  both origins. Use this to verify per-origin isolation.

All Freight shares the same creation timestamp, so the candidate is determined
by lexical name order (descending hash):

| Alias | Hash (prefix) | Role |
|-------|--------------|------|
| `frontend-v002` | `7d96255...` | candidate (lexically largest) |
| `frontend-v001` | `39f8209...` | non-candidate |
| `frontend-v003` | `3966bf2...` | non-candidate (lexically smallest) |
| `api-v002` | `d2f95df...` | candidate |
| `api-v001` | `10e608d...` | non-candidate |

## Scenarios

### A — Hold Establishment

#### A1 · promote non-candidate by name → hold established

Promote `frontend-v001` (non-candidate) to `single-origin-hold`. The webhook
stamps `kargo.akuity.io/auto-promotion-hold`; after the Promotion succeeds the
Stage controller records the hold in `status.autoPromotionHolds`.

```shell
kargo promote --project auto-promotion-hold --stage single-origin-hold \
  --freight-alias frontend-v001
```

Verify:

```shell
kubectl get stage single-origin-hold -n auto-promotion-hold \
  -o jsonpath='{.status.autoPromotionHolds}' | jq .
```

Expected: `Warehouse/auto-hold` entry present with `freightName: 39f8209...`.

---

### B — Auto-Promotion Blocked

#### B1 · hold blocks auto-promotion for held origin

With the hold from A1 active, confirm that no new auto-promotion Promotion
fires even though newer Freight is available.

```shell
kubectl get promotions -n auto-promotion-hold \
  --sort-by=.metadata.creationTimestamp
```

Expected: no new Promotion created after the hold-intent one; the stage stays
at `frontend-v001`.

#### B4 · hold persists after establishing Promotion is deleted

Delete the Promotion that established the hold and confirm `autoPromotionHolds`
is unchanged (hold data lives in Stage status, not in the Promotion object).

```shell
hold_promo=$(kubectl get promotions -n auto-promotion-hold -o json | \
  jq -r '
    .items[]
    | select(
        .metadata.annotations["kargo.akuity.io/auto-promotion-hold"] != null
        and .status.phase == "Succeeded"
      )
    | .metadata.name' | head -1)
echo "Deleting: ${hold_promo}"
kubectl delete promotion "${hold_promo}" -n auto-promotion-hold
kubectl get stage single-origin-hold -n auto-promotion-hold \
  -o jsonpath='{.status.autoPromotionHolds}' | jq .
```

Expected: `Warehouse/auto-hold` hold still present.

---

### C — Hold Cleared (Resume)

#### C1 · promote candidate by freight name → hold cleared

With the hold from A1 active, promote the candidate by hash. The webhook
detects candidate Freight and stamps the resume annotation.

```shell
kargo promote --project auto-promotion-hold --stage single-origin-hold \
  --freight-alias frontend-v002
```

Verify:

```shell
kubectl get stage single-origin-hold -n auto-promotion-hold \
  -o jsonpath='{.status.autoPromotionHolds}' | jq .
```

Expected: empty — hold cleared.

#### C3 · promote by warehouse → hold cleared (race-free path)

Re-establish the hold first (re-run A1), then use `--warehouse` so the server
resolves the current candidate and stamps the resume annotation server-side.
This is the intended ergonomic path.

```shell
# Re-establish hold
kargo promote --project auto-promotion-hold --stage single-origin-hold \
  --freight-alias frontend-v001

# Clear via warehouse resolution
kargo promote --project auto-promotion-hold --stage single-origin-hold \
  --warehouse auto-hold
```

Verify:

```shell
kubectl get stage single-origin-hold -n auto-promotion-hold \
  -o jsonpath='{.status.autoPromotionHolds}' | jq .
```

Expected: empty.

#### C4 · candidate rotated since hold was established → new candidate clears hold

Establish a hold, then promote the current candidate. The hold clears
regardless of which Freight originally established it.

```shell
# Establish hold with frontend-v001
kargo promote --project auto-promotion-hold --stage single-origin-hold \
  --freight-alias frontend-v001

# Clear with the current candidate (frontend-v002)
kargo promote --project auto-promotion-hold --stage single-origin-hold \
  --freight-alias frontend-v002
```

Verify: `status.autoPromotionHolds` is empty after the second Promotion
succeeds.

---

### D — Auto-Promotion Resumes

#### D1 · auto-promotion fires after hold is cleared

After clearing a hold (run C1 or C3 first), confirm the reconciler does not
create a duplicate auto-promotion Promotion for the candidate that was just
manually promoted.

```shell
kubectl get promotions -n auto-promotion-hold \
  --sort-by=.metadata.creationTimestamp | tail -5
```

Expected: no new auto-promotion Promotion for `frontend-v002`. If new Freight
were to appear in the Warehouse, an auto-promotion Promotion would fire within
one reconcile.

---

### E — Multiple Origins

#### E1 · releasing one origin does not affect the other

`multi-origin-holds` starts with holds on both `Warehouse/auto-hold` and
`Warehouse/auto-hold-api`. Release only `auto-hold` and confirm `auto-hold-api`
remains held.

```shell
kargo promote --project auto-promotion-hold --stage multi-origin-holds \
  --warehouse auto-hold
```

Verify:

```shell
kubectl get stage multi-origin-holds -n auto-promotion-hold \
  -o jsonpath='{.status.autoPromotionHolds}' | jq .
```

Expected: only `Warehouse/auto-hold-api` remains.

#### E2 · clear one origin, establish hold on another in one reconcile

With `Warehouse/auto-hold-api` still held (from E1), promote the candidate
for that origin to clear it, then immediately promote a non-candidate for
`auto-hold`. Both changes land in the same reconcile pass.

```shell
# Clear auto-hold-api hold
kargo promote --project auto-promotion-hold --stage multi-origin-holds \
  --warehouse auto-hold-api

# Establish hold on auto-hold
kargo promote --project auto-promotion-hold --stage multi-origin-holds \
  --freight-alias frontend-v001
```

Wait for both Promotions to complete, then verify:

```shell
kubectl get stage multi-origin-holds -n auto-promotion-hold \
  -o jsonpath='{.status.autoPromotionHolds}' | jq .
```

Expected: `Warehouse/auto-hold` hold present, `Warehouse/auto-hold-api` absent.

---

### G — Promote by Origin

#### G2 · promote by warehouse with no active hold → no hold established

When no hold exists and the candidate is promoted via `--warehouse`, the webhook
stamps the resume annotation but no hold is created.

```shell
# Ensure no hold is active first
kubectl get stage single-origin-hold -n auto-promotion-hold \
  -o jsonpath='{.status.autoPromotionHolds}' | jq .

kargo promote --project auto-promotion-hold --stage single-origin-hold \
  --warehouse auto-hold
```

Verify after the Promotion succeeds:

```shell
kubectl get stage single-origin-hold -n auto-promotion-hold \
  -o jsonpath='{.status.autoPromotionHolds}' | jq .
```

Expected: empty — no hold created by a candidate promotion.

---

### H — System-generated Promotions

#### H1 · auto-promotion Promotion succeeds without establishing a hold

Confirm that Promotions created by the controller do not have the
`kargo.akuity.io/auto-promotion-hold` annotation and do not add entries to
`status.autoPromotionHolds`.

```shell
kubectl get promotions -n auto-promotion-hold -o json | \
  jq '[.items[] | select(.metadata.annotations["kargo.akuity.io/auto-promotion-hold"] == null) | .metadata.name]'
```

Expected: the initial auto-promotion Promotions appear in this list. Cross-check:

```shell
kubectl get stage single-origin-hold -n auto-promotion-hold \
  -o jsonpath='{.status.autoPromotionHolds}' | jq .
```

Expected: empty (assuming no manual hold-intent Promotion has run).

---

## Coverage notes

Scenarios that depend on a reliably-failing Promotion step (F1, F2) are
covered by unit tests in `pkg/controller/stages` and
`pkg/webhook/kubernetes/promotion`.
