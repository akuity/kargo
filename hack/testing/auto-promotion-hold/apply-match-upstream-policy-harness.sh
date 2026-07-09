#!/usr/bin/env bash
set -euo pipefail

PROJECT=auto-promotion-hold
FREIGHT_V001=39f8209d87b5222d5dbebf5e6f1d9a54fe7d7b52   # alias: frontend-v001 (nginx 1.26.0)
API_FREIGHT_V001=10e608d617ce292f14398c07bcfadedd27b1ae6c  # alias: api-v001 (redis 7.2.4)

SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)

kubectl apply -f "${SCRIPT_DIR}/project.yaml"

echo "Waiting for namespace ${PROJECT}..."
for _ in {1..90}; do
  kubectl get namespace "${PROJECT}" >/dev/null 2>&1 && break
  sleep 1
done
kubectl get namespace "${PROJECT}" >/dev/null

# Delete prior state for the MatchUpstream stages only. The NewestFreight
# stages (single-origin-hold, multi-origin-holds) are left untouched so both
# harnesses can coexist in the same project.
kubectl delete promotion --all \
  --namespace "${PROJECT}" \
  --ignore-not-found=true \
  --wait=true
kubectl delete stage upstream downstream-single-origin upstream-multi downstream-multi-origin \
  --namespace "${PROJECT}" \
  --ignore-not-found=true \
  --wait=true

kubectl apply -f "${SCRIPT_DIR}/project-config.yaml"
kubectl apply -f "${SCRIPT_DIR}/warehouses.yaml"
kubectl apply -f "${SCRIPT_DIR}/freight.yaml"
kubectl apply -f "${SCRIPT_DIR}/stages-match-upstream.yaml"

# Wait for both upstream stages to complete their initial auto-promotion
# (set-metadata only, completes in seconds).
echo "Waiting for upstream to be Ready (initial auto-promotion)..."
kubectl wait stage upstream \
  --namespace "${PROJECT}" \
  --for=condition=Ready \
  --timeout=120s

echo "Waiting for upstream-multi to be Ready (initial auto-promotion)..."
kubectl wait stage upstream-multi \
  --namespace "${PROJECT}" \
  --for=condition=Ready \
  --timeout=120s

# downstream-single-origin uses an HTTP step (10s delay) to ensure the
# reconciler can reliably observe it as Running before it completes.
echo "Waiting for downstream-single-origin to be Ready (initial auto-promotion, ~15s)..."
kubectl wait stage downstream-single-origin \
  --namespace "${PROJECT}" \
  --for=condition=Ready \
  --timeout=120s

echo "Waiting for downstream-multi-origin to be Ready (initial auto-promotion)..."
kubectl wait stage downstream-multi-origin \
  --namespace "${PROJECT}" \
  --for=condition=Ready \
  --timeout=120s

# Approve the non-candidate Freight for each downstream stage. The MatchUpstream
# candidate is whatever the upstream stage currently has (frontend-v002 /
# api-v002). Freight is only available to a MatchUpstream stage if it has been
# verified in the upstream OR directly approved; since the upstream never
# promoted these older versions, we approve them directly.
echo "Approving non-candidate Freight for downstream stages..."
kargo approve \
  --project "${PROJECT}" \
  --freight "${FREIGHT_V001}" \
  --stage downstream-single-origin
kargo approve \
  --project "${PROJECT}" \
  --freight "${FREIGHT_V001}" \
  --stage downstream-multi-origin
kargo approve \
  --project "${PROJECT}" \
  --freight "${API_FREIGHT_V001}" \
  --stage downstream-multi-origin

# Create hold-intent Promotions for the downstream stages.
#
# The webhook stamps kargo.akuity.io/auto-promotion-hold on each because
# auto-promotion is enabled and the promoted Freight is not the MatchUpstream
# candidate: frontend-v001 ≠ frontend-v002 (what upstream has); api-v001 ≠
# api-v002 (what upstream-multi has).
echo "Creating hold-intent Promotion for downstream-single-origin..."
FRONTEND_HOLD_PROMO=$(kubectl create \
  --output 'jsonpath={.metadata.name}' \
  -f - <<YAML
apiVersion: kargo.akuity.io/v1alpha1
kind: Promotion
metadata:
  name: placeholder
  namespace: ${PROJECT}
spec:
  stage: downstream-single-origin
  freight: ${FREIGHT_V001}
YAML
)
echo "  frontend hold (single): ${FRONTEND_HOLD_PROMO}"

echo "Creating hold-intent Promotions for downstream-multi-origin..."
FRONTEND_MULTI_HOLD_PROMO=$(kubectl create \
  --output 'jsonpath={.metadata.name}' \
  -f - <<YAML
apiVersion: kargo.akuity.io/v1alpha1
kind: Promotion
metadata:
  name: placeholder
  namespace: ${PROJECT}
spec:
  stage: downstream-multi-origin
  freight: ${FREIGHT_V001}
YAML
)
echo "  frontend hold (multi): ${FRONTEND_MULTI_HOLD_PROMO}"

API_MULTI_HOLD_PROMO=$(kubectl create \
  --output 'jsonpath={.metadata.name}' \
  -f - <<YAML
apiVersion: kargo.akuity.io/v1alpha1
kind: Promotion
metadata:
  name: placeholder
  namespace: ${PROJECT}
spec:
  stage: downstream-multi-origin
  freight: ${API_FREIGHT_V001}
YAML
)
echo "  api hold (multi): ${API_MULTI_HOLD_PROMO}"

echo "Waiting for hold-intent Promotions to succeed..."
for promo in "${FRONTEND_HOLD_PROMO}" "${FRONTEND_MULTI_HOLD_PROMO}" "${API_MULTI_HOLD_PROMO}"; do
  for _ in {1..60}; do
    phase=$(kubectl get promotion "${promo}" \
      --namespace "${PROJECT}" \
      --output 'jsonpath={.status.phase}' \
      2>/dev/null || echo "")
    if [[ "${phase}" == "Succeeded" || "${phase}" == "Failed" || "${phase}" == "Error" ]]; then
      echo "  ${promo}: ${phase}"
      break
    fi
    sleep 2
  done
done

echo "Waiting for holds to be established on downstream-single-origin..."
for _ in {1..30}; do
  holds=$(kubectl get stage downstream-single-origin \
    --namespace "${PROJECT}" \
    --output 'jsonpath={.status.autoPromotionHolds}' \
    2>/dev/null || echo "")
  if [[ -n "${holds}" && "${holds}" != "null" && "${holds}" != "{}" ]]; then
    echo "  holds: ${holds}"
    break
  fi
  sleep 2
done

echo "Waiting for holds to be established on downstream-multi-origin..."
holds_set=false
for _ in {1..30}; do
  holds=$(kubectl get stage downstream-multi-origin \
    --namespace "${PROJECT}" \
    --output 'jsonpath={.status.autoPromotionHolds}' \
    2>/dev/null || echo "")
  hold_count=$(echo "${holds}" | jq 'length' 2>/dev/null || echo "0")
  if [[ "${hold_count}" == "2" ]]; then
    echo "  holds: ${holds}"
    holds_set=true
    break
  fi
  sleep 2
done
if [[ "${holds_set}" != "true" ]]; then
  echo "  WARNING: both holds not established within timeout"
fi

echo ""
echo "Setup complete."
echo ""
echo "Stages:"
echo "  upstream               — NewestFreight, currently at frontend-v002"
echo "  downstream-single-origin — MatchUpstream from upstream, HOLD on Warehouse/auto-hold"
echo "  upstream-multi         — NewestFreight, currently at {frontend-v002, api-v002}"
echo "  downstream-multi-origin  — MatchUpstream from upstream-multi, HOLD on both origins"
echo ""
echo "Hold-intent Promotion names (needed for scenario B4):"
echo "  frontend (single): ${FRONTEND_HOLD_PROMO}"
echo "  frontend (multi):  ${FRONTEND_MULTI_HOLD_PROMO}"
echo "  api (multi):       ${API_MULTI_HOLD_PROMO}"
echo ""

kubectl get project "${PROJECT}"
kubectl get warehouse,stage,freight,promotion --namespace "${PROJECT}"
