#!/usr/bin/env bash
set -euo pipefail

PROJECT=auto-promotion-hold
FREIGHT_V001=39f8209d87b5222d5dbebf5e6f1d9a54fe7d7b52   # alias: frontend-v001 (nginx 1.26.0)
FREIGHT_V002=7d96255278537d99b0c35445c3da426147d990bf   # alias: frontend-v002 (nginx 1.26.1)
FREIGHT_V003=3966bf28d5d67698bfd4816aecafd66d96a4226c   # alias: frontend-v003 (nginx 1.26.2)
API_FREIGHT_V001=10e608d617ce292f14398c07bcfadedd27b1ae6c  # alias: api-v001 (redis 7.2.4)
API_FREIGHT_V002=d2f95df42a8f2ed206d7f4c15c5c5888454633eb  # alias: api-v002 (redis 7.2.5)

SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)

kubectl apply -f "${SCRIPT_DIR}/project.yaml"

echo "Waiting for namespace ${PROJECT}..."
for _ in {1..90}; do
  kubectl get namespace "${PROJECT}" >/dev/null 2>&1 && break
  sleep 1
done
kubectl get namespace "${PROJECT}" >/dev/null

# Delete prior state so each run starts fresh.
kubectl delete promotion --all \
  --namespace "${PROJECT}" \
  --ignore-not-found=true \
  --wait=true
kubectl delete stage single-origin-hold multi-origin-holds \
  --namespace "${PROJECT}" \
  --ignore-not-found=true \
  --wait=true
kubectl delete projectconfig "${PROJECT}" \
  --namespace "${PROJECT}" \
  --ignore-not-found=true \
  --wait=true

kubectl apply -f "${SCRIPT_DIR}/project-config.yaml"
kubectl apply -f "${SCRIPT_DIR}/warehouses.yaml"
kubectl apply -f "${SCRIPT_DIR}/freight.yaml"
kubectl apply -f "${SCRIPT_DIR}/stages.yaml"

# Wait for multi-origin-holds to complete its initial auto-promotion (only
# set-metadata steps, completes in seconds). This ensures both origins have
# current Freight before we create hold-intent Promotions; if we create them
# before the initial auto-promotion runs they could interleave in a way that
# leaves the Stage with stale current Freight.
echo "Waiting for multi-origin-holds to be Ready (initial auto-promotion)..."
kubectl wait stage multi-origin-holds \
  --namespace "${PROJECT}" \
  --for=condition=Ready \
  --timeout=120s

# Create hold-intent Promotions for multi-origin-holds.
#
# The Promotion webhook stamps kargo.akuity.io/auto-promotion-hold on each
# because auto-promotion is enabled (ProjectConfig) and both freights are
# non-candidates: frontend-v001 < frontend-v003 (candidate); api-v001 <
# api-v002 (candidate).
#
# The webhook also renames every Promotion to <stage>.<ulid>.<freight-7> via
# GeneratePromotionName, so we capture the actual names from the API response.
echo "Creating hold-intent Promotions for multi-origin-holds..."

FRONTEND_HOLD_PROMO=$(kubectl create \
  --output 'jsonpath={.metadata.name}' \
  -f - <<YAML
apiVersion: kargo.akuity.io/v1alpha1
kind: Promotion
metadata:
  name: placeholder
  namespace: ${PROJECT}
spec:
  stage: multi-origin-holds
  freight: ${FREIGHT_V001}
YAML
)
echo "  frontend hold: ${FRONTEND_HOLD_PROMO}"

API_HOLD_PROMO=$(kubectl create \
  --output 'jsonpath={.metadata.name}' \
  -f - <<YAML
apiVersion: kargo.akuity.io/v1alpha1
kind: Promotion
metadata:
  name: placeholder
  namespace: ${PROJECT}
spec:
  stage: multi-origin-holds
  freight: ${API_FREIGHT_V001}
YAML
)
echo "  api hold: ${API_HOLD_PROMO}"

echo "Waiting for hold-intent Promotions to succeed..."
for promo in "${FRONTEND_HOLD_PROMO}" "${API_HOLD_PROMO}"; do
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

echo "Waiting for auto-promotion holds to be established..."
holds_set=false
for _ in {1..30}; do
  holds=$(kubectl get stage multi-origin-holds \
    --namespace "${PROJECT}" \
    --output 'jsonpath={.status.autoPromotionHolds}' \
    2>/dev/null || echo "")
  if [[ -n "${holds}" && "${holds}" != "null" && "${holds}" != "{}" ]]; then
    echo "  holds: ${holds}"
    holds_set=true
    break
  fi
  sleep 2
done
if [[ "${holds_set}" != "true" ]]; then
  echo "  WARNING: holds not established within timeout"
fi

echo ""
echo "Setup complete. Hold-intent Promotion names (needed for scenario 4):"
echo "  frontend: ${FRONTEND_HOLD_PROMO}"
echo "  api:      ${API_HOLD_PROMO}"
echo ""

kubectl get project "${PROJECT}"
kubectl get warehouse,stage,freight,promotion --namespace "${PROJECT}"
