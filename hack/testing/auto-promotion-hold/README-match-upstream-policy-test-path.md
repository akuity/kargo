# Auto-Promotion Hold — MatchUpstream Policy Harness

Manual test harness for the auto-promotion hold feature under the
**MatchUpstream** selection policy.

See [README-newest-freight-policy-test-path.md](README-newest-freight-policy-test-path.md)
for the NewestFreight equivalent.

## Prerequisites

A running Tilt dev environment (`make hack-tilt-up`).

## Setup

```shell
./hack/testing/auto-promotion-hold/apply-match-upstream-policy-harness.sh
```

Creates four Stages in the `auto-promotion-hold` project:

- **`upstream`** — single-origin NewestFreight Stage. Auto-promotes to the
  candidate and acts as the source of truth for `downstream-single-origin`.
- **`downstream-single-origin`** — single-origin MatchUpstream Stage seeded
  with an active hold. The candidate is whatever `upstream` currently has.
- **`upstream-multi`** — multi-origin NewestFreight Stage. Acts as the source
  of truth for `downstream-multi-origin`.
- **`downstream-multi-origin`** — multi-origin MatchUpstream Stage seeded with
  active holds on both origins.

These Stages share the `auto-promotion-hold` project and Freight with the
NewestFreight harness. Running this script does not disturb the NewestFreight
Stages.

## Freight reference

Under MatchUpstream the candidate for each downstream origin is whatever the
upstream Stage currently has after initial auto-promotion.

| Alias | Hash (prefix) | Role |
|-------|--------------|------|
| `frontend-v002` | `7d96255...` | candidate (what `upstream` / `upstream-multi` promotes to) |
| `frontend-v001` | `39f8209...` | non-candidate |
| `api-v002` | `d2f95df...` | candidate |
| `api-v001` | `10e608d...` | non-candidate |

## Scenarios

### A — Hold Establishment

#### A2 · promote non-upstream freight → hold established

The apply script seeded `downstream-single-origin` with a hold by promoting
`frontend-v001` while `upstream` is at `frontend-v002`. Verify the hold is
present:

```shell
kubectl get stage downstream-single-origin -n auto-promotion-hold \
  -o jsonpath='{.status.autoPromotionHolds}' | jq .
```

Expected: `Warehouse/auto-hold` hold with `freightName: 39f8209...`.

To re-establish the hold after clearing it, approve and promote the
non-candidate freight:

```shell
kargo approve --project auto-promotion-hold \
  --freight-alias frontend-v001 --stage downstream-single-origin
kargo promote --project auto-promotion-hold \
  --stage downstream-single-origin --freight-alias frontend-v001
```

---

### B — Auto-Promotion Blocked

#### B2 · upstream promotes new freight; downstream hold persists

With the A2 hold active, `downstream-single-origin` must NOT follow `upstream`
even when upstream has new freight. Confirm no new auto-promotion Promotion
fires for the downstream:

```shell
kubectl get promotions -n auto-promotion-hold \
  --sort-by=.metadata.creationTimestamp | grep downstream-single
```

Expected: only the initial auto-promotion and the hold-intent Promotion appear.

#### B4 · hold persists after establishing Promotion is deleted

Delete the Promotion that established the hold and confirm `autoPromotionHolds`
is unchanged.

```shell
hold_promo=$(kubectl get promotions -n auto-promotion-hold -o json | \
  jq -r '
    .items[]
    | select(
        .metadata.annotations["kargo.akuity.io/auto-promotion-hold"] != null
        and .spec.stage == "downstream-single-origin"
        and .status.phase == "Succeeded"
      )
    | .metadata.name' | head -1)
echo "Deleting: ${hold_promo}"
kubectl delete promotion "${hold_promo}" -n auto-promotion-hold
kubectl get stage downstream-single-origin -n auto-promotion-hold \
  -o jsonpath='{.status.autoPromotionHolds}' | jq .
```

Expected: `Warehouse/auto-hold` hold still present.

---

### C — Hold Cleared (Resume)

#### C2 · promote upstream freight by name → hold cleared

With the A2 hold active, promote the freight currently in `upstream`
(`frontend-v002`) to `downstream-single-origin` by name:

```shell
kargo promote --project auto-promotion-hold \
  --stage downstream-single-origin --freight-alias frontend-v002
```

Verify:

```shell
kubectl get stage downstream-single-origin -n auto-promotion-hold \
  -o jsonpath='{.status.autoPromotionHolds}' | jq .
```

Expected: empty — the webhook detected `frontend-v002` as the MatchUpstream
candidate and stamped resume intent.

#### C3 · promote by warehouse → hold cleared (race-free path)

Re-establish the hold first (re-run A2), then use `--warehouse` so the server
resolves the current upstream candidate server-side:

```shell
# Re-establish hold
kargo approve --project auto-promotion-hold \
  --freight-alias frontend-v001 --stage downstream-single-origin
kargo promote --project auto-promotion-hold \
  --stage downstream-single-origin --freight-alias frontend-v001

# Clear via upstream resolution
kargo promote --project auto-promotion-hold \
  --stage downstream-single-origin --warehouse auto-hold
```

Verify: `status.autoPromotionHolds` is empty.

---

### D — Auto-Promotion Resumes

#### D2 · auto-promotion resumes after hold cleared

After clearing the hold (run C2 or C3), the downstream should auto-promote to
follow `upstream` on the next reconcile. Since the candidate (`frontend-v002`)
was just manually promoted, the reconciler sees no new freight and stays idle.

```shell
kubectl get promotions -n auto-promotion-hold \
  --sort-by=.metadata.creationTimestamp | grep downstream-single
```

Expected: no new auto-promotion Promotion created after the hold was cleared.
The downstream is already at the upstream candidate.

---

### E — Multiple Origins

#### E1 · per-origin isolation on multi-origin downstream

`downstream-multi-origin` starts with holds on both origins. Release only
`Warehouse/auto-hold` and confirm `Warehouse/auto-hold-api` remains held:

```shell
kargo promote --project auto-promotion-hold \
  --stage downstream-multi-origin --warehouse auto-hold
```

Verify:

```shell
kubectl get stage downstream-multi-origin -n auto-promotion-hold \
  -o jsonpath='{.status.autoPromotionHolds}' | jq .
```

Expected: only `Warehouse/auto-hold-api` remains.

---

### G — Promote by Origin

#### G1 · promote by warehouse when no upstream candidate → admission denied

This fires when the upstream Stage is mid-promotion (its freight is transiently
ambiguous) or when it has no verified freight yet. Trigger it by promoting by
warehouse immediately after restarting `upstream` or while it is actively
promoting:

```shell
kargo promote --project auto-promotion-hold \
  --stage downstream-single-origin --warehouse auto-hold
```

Expected: admission denied with a message like
`"no auto-promotion candidate found for origin"`.

---

## Coverage notes

Scenarios that depend on a reliably-failing Promotion step (F1, F2) are
covered by unit tests in `pkg/controller/stages` and
`pkg/webhook/kubernetes/promotion`.
