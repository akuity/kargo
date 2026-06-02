#!/usr/bin/env bash
set -euo pipefail

PROJECT=auto-promotion-hold
ORIGIN_KEY=Warehouse/auto-hold
IMAGE_REPO=public.ecr.aws/nginx/nginx
FREIGHT_V001=39f8209d87b5222d5dbebf5e6f1d9a54fe7d7b52
FREIGHT_V002=7d96255278537d99b0c35445c3da426147d990bf
FREIGHT_V003=3966bf28d5d67698bfd4816aecafd66d96a4226c

SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
NOW=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

kubectl apply -f "${SCRIPT_DIR}/project.yaml"

for _ in {1..90}; do
  if kubectl get namespace "${PROJECT}" >/dev/null 2>&1; then
    break
  fi
  sleep 1
done
kubectl get namespace "${PROJECT}" >/dev/null

kubectl delete projectconfig "${PROJECT}" \
  --namespace "${PROJECT}" \
  --ignore-not-found=true \
  --wait=true
kubectl delete promotion --all \
  --namespace "${PROJECT}" \
  --ignore-not-found=true \
  --wait=true
kubectl delete stage rollback-demo active-hold pending-hold \
  --namespace "${PROJECT}" \
  --ignore-not-found=true \
  --wait=true
kubectl delete deployment slow-promotion-endpoint \
  --namespace "${PROJECT}" \
  --ignore-not-found=true \
  --wait=true
kubectl delete service slow-promotion-endpoint \
  --namespace "${PROJECT}" \
  --ignore-not-found=true \
  --wait=true

kubectl apply -f "${SCRIPT_DIR}/resources.yaml"

kubectl patch stage slow-pending-hold \
  --namespace "${PROJECT}" \
  --subresource=status \
  --type=json \
  --patch='[{"op":"replace","path":"/status","value":{}}]' \
  >/dev/null || true

for freight in "${FREIGHT_V001}" "${FREIGHT_V002}" "${FREIGHT_V003}"; do
  kubectl patch freight "${freight}" \
    --namespace "${PROJECT}" \
    --subresource=status \
    --type=json \
    --patch='[{"op":"replace","path":"/status","value":{}}]' \
    >/dev/null || true
done

kubectl patch freight "${FREIGHT_V003}" \
  --namespace "${PROJECT}" \
  --subresource=status \
  --type=merge \
  --patch "$(cat <<JSON
{
  "status": {
    "currentlyIn": {
      "slow-pending-hold": {"since": "${NOW}"}
    }
  }
}
JSON
)"

kubectl patch stage slow-pending-hold \
  --namespace "${PROJECT}" \
  --subresource=status \
  --type=json \
  --patch "$(cat <<JSON
[
  {
    "op": "replace",
    "path": "/status",
    "value": {
      "autoPromotionEnabled": true,
      "conditions": [
        {
          "type": "Ready",
          "status": "True",
          "reason": "Verified",
          "message": "Freight has been verified",
          "lastTransitionTime": "${NOW}"
        },
        {
          "type": "Healthy",
          "status": "True",
          "reason": "Healthy",
          "message": "Stage is healthy (performed 0 health checks)",
          "lastTransitionTime": "${NOW}"
        },
        {
          "type": "Verified",
          "status": "True",
          "reason": "Verified",
          "message": "Freight has been verified",
          "lastTransitionTime": "${NOW}"
        }
      ],
      "freightSummary": "frontend-v003",
      "health": {"status": "Healthy"},
      "freightHistory": [
        {
          "id": "slow-pending-hold-v003",
          "items": {
            "${ORIGIN_KEY}": {
              "name": "${FREIGHT_V003}",
              "origin": {"kind": "Warehouse", "name": "auto-hold"},
              "images": [{"repoURL": "${IMAGE_REPO}", "tag": "1.26.2"}]
            }
          },
          "verificationHistory": [
            {
              "id": "manual-seed",
              "phase": "Successful",
              "message": "Seeded verified current Freight for manual UX review.",
              "startTime": "${NOW}",
              "finishTime": "${NOW}"
            }
          ]
        }
      ]
    }
  }
]
JSON
)"

kubectl apply -f "${SCRIPT_DIR}/project-config.yaml"

kubectl get project "${PROJECT}"
kubectl get warehouse,stage,freight,promotion --namespace "${PROJECT}"
