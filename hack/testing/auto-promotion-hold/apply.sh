#!/usr/bin/env bash
set -euo pipefail

PROJECT=auto-promotion-hold
ORIGIN_KEY=Warehouse/auto-hold
API_ORIGIN_KEY=Warehouse/auto-hold-api
IMAGE_REPO=public.ecr.aws/nginx/nginx
API_IMAGE_REPO=public.ecr.aws/docker/library/redis
FREIGHT_V001=39f8209d87b5222d5dbebf5e6f1d9a54fe7d7b52
FREIGHT_V002=7d96255278537d99b0c35445c3da426147d990bf
FREIGHT_V003=3966bf28d5d67698bfd4816aecafd66d96a4226c
API_FREIGHT_V001=10e608d617ce292f14398c07bcfadedd27b1ae6c
API_FREIGHT_V002=d2f95df42a8f2ed206d7f4c15c5c5888454633eb

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
kubectl delete stage rollback-demo active-hold pending-hold slow-pending-hold \
  single-origin-hold multi-origin-holds \
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

kubectl patch stage single-origin-hold \
  --namespace "${PROJECT}" \
  --subresource=status \
  --type=json \
  --patch='[{"op":"replace","path":"/status","value":{}}]' \
  >/dev/null || true

for freight in \
  "${FREIGHT_V001}" \
  "${FREIGHT_V002}" \
  "${FREIGHT_V003}" \
  "${API_FREIGHT_V001}" \
  "${API_FREIGHT_V002}"; do
  kubectl patch freight "${freight}" \
    --namespace "${PROJECT}" \
    --subresource=status \
    --type=json \
    --patch='[{"op":"replace","path":"/status","value":{}}]' \
    >/dev/null || true
done

kubectl patch freight "${FREIGHT_V001}" \
  --namespace "${PROJECT}" \
  --subresource=status \
  --type=merge \
  --patch "$(cat <<JSON
{
  "status": {
    "currentlyIn": {
      "multi-origin-holds": {"since": "${NOW}"}
    }
  }
}
JSON
)"

kubectl patch freight "${FREIGHT_V003}" \
  --namespace "${PROJECT}" \
  --subresource=status \
  --type=merge \
  --patch "$(cat <<JSON
{
  "status": {
    "currentlyIn": {
      "single-origin-hold": {"since": "${NOW}"}
    }
  }
}
JSON
)"

kubectl patch freight "${API_FREIGHT_V001}" \
  --namespace "${PROJECT}" \
  --subresource=status \
  --type=merge \
  --patch "$(cat <<JSON
{
  "status": {
    "currentlyIn": {
      "multi-origin-holds": {"since": "${NOW}"}
    }
  }
}
JSON
)"

kubectl patch stage single-origin-hold \
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
          "id": "single-origin-hold-v003",
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

kubectl patch stage multi-origin-holds \
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
      "freightSummary": "2/2 origins fulfilled",
      "health": {"status": "Healthy"},
      "freightHistory": [
        {
          "id": "multi-origin-holds-current",
          "items": {
            "${ORIGIN_KEY}": {
              "name": "${FREIGHT_V001}",
              "origin": {"kind": "Warehouse", "name": "auto-hold"},
              "images": [{"repoURL": "${IMAGE_REPO}", "tag": "1.26.0"}]
            },
            "${API_ORIGIN_KEY}": {
              "name": "${API_FREIGHT_V001}",
              "origin": {"kind": "Warehouse", "name": "auto-hold-api"},
              "images": [{"repoURL": "${API_IMAGE_REPO}", "tag": "7.2.4"}]
            }
          },
          "verificationHistory": [
            {
              "id": "manual-seed",
              "phase": "Successful",
              "message": "Seeded verified multi-origin Freight for manual UX review.",
              "startTime": "${NOW}",
              "finishTime": "${NOW}"
            }
          ]
        }
      ],
      "autoPromotionHolds": {
        "${ORIGIN_KEY}": {
          "freight": {
            "name": "${FREIGHT_V001}",
            "origin": {"kind": "Warehouse", "name": "auto-hold"}
          },
          "state": "Active",
          "promotionName": "multi-origin-holds.frontend-rollback",
          "promotionUID": "11111111-1111-1111-1111-111111111111",
          "actor": "user:demo@example.com",
          "reason": "Demo active hold after rollback",
          "createdAt": "${NOW}"
        },
        "${API_ORIGIN_KEY}": {
          "freight": {
            "name": "${API_FREIGHT_V001}",
            "origin": {"kind": "Warehouse", "name": "auto-hold-api"}
          },
          "state": "Active",
          "promotionName": "multi-origin-holds.api-rollback",
          "promotionUID": "22222222-2222-2222-2222-222222222222",
          "actor": "user:demo@example.com",
          "reason": "Demo active hold after API rollback",
          "createdAt": "${NOW}"
        }
      }
    }
  }
]
JSON
)"

kubectl apply -f "${SCRIPT_DIR}/project-config.yaml"

kubectl get project "${PROJECT}"
kubectl get warehouse,stage,freight,promotion --namespace "${PROJECT}"
