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
MULTI_ORIGIN_SEED_PROMOTION=multi-origin-holds.00000000000000000000000000.seed
FRONTEND_ROLLBACK_PROMOTION=multi-origin-holds.00000000000000000000000001.rollback-frontend
API_ROLLBACK_PROMOTION=multi-origin-holds.00000000000000000000000002.rollback-api

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
kubectl delete stage single-origin-hold multi-origin-holds \
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

# Disable auto-promotion while seeding state to avoid races.
kubectl apply -f - <<YAML
apiVersion: kargo.akuity.io/v1alpha1
kind: ProjectConfig
metadata:
  name: ${PROJECT}
  namespace: ${PROJECT}
spec:
  promotionPolicies:
  - stageSelector:
      name: single-origin-hold
    autoPromotionEnabled: false
  - stageSelector:
      name: multi-origin-holds
    autoPromotionEnabled: false
YAML

kubectl apply -f "${SCRIPT_DIR}/resources.yaml"

# Seed the multi-origin-holds stage with a completed promotion so it shows
# current freight and has a verified state to start from.
kubectl apply -f - <<YAML
apiVersion: kargo.akuity.io/v1alpha1
kind: Promotion
metadata:
  name: ${MULTI_ORIGIN_SEED_PROMOTION}
  namespace: ${PROJECT}
spec:
  stage: multi-origin-holds
  freight: ${FREIGHT_V001}
  steps:
  - uses: set-metadata
    config:
      updates:
      - kind: Stage
        name: multi-origin-holds
        values:
          manualTest: auto-promotion-hold
          lastStory: multi-origin-holds
YAML

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

kubectl patch promotion "${MULTI_ORIGIN_SEED_PROMOTION}" \
  --namespace "${PROJECT}" \
  --subresource=status \
  --type=merge \
  --patch "$(cat <<JSON
{
  "status": {
    "phase": "Succeeded",
    "startedAt": "${NOW}",
    "finishedAt": "${NOW}",
    "freight": {
      "name": "${FREIGHT_V001}",
      "origin": {"kind": "Warehouse", "name": "auto-hold"},
      "images": [{"repoURL": "${IMAGE_REPO}", "tag": "1.26.0"}]
    },
    "freightCollection": {
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
      }
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
      "lastPromotion": {
        "name": "${MULTI_ORIGIN_SEED_PROMOTION}",
        "finishedAt": "${NOW}",
        "freight": {
          "name": "${FREIGHT_V001}",
          "origin": {"kind": "Warehouse", "name": "auto-hold"},
          "images": [{"repoURL": "${IMAGE_REPO}", "tag": "1.26.0"}]
        },
        "status": {
          "phase": "Succeeded",
          "startedAt": "${NOW}",
          "finishedAt": "${NOW}",
          "freight": {
            "name": "${FREIGHT_V001}",
            "origin": {"kind": "Warehouse", "name": "auto-hold"},
            "images": [{"repoURL": "${IMAGE_REPO}", "tag": "1.26.0"}]
          },
          "freightCollection": {
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
            }
          }
        }
      },
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
      ]
    }
  }
]
JSON
)"

# Create rollback Promotions so the Stage controller derives holds for both
# origins. These run immediately (set-metadata is fast) but auto-promotion
# is disabled above so there is no competing traffic.
kubectl apply -f - <<YAML
apiVersion: kargo.akuity.io/v1alpha1
kind: Promotion
metadata:
  name: ${FRONTEND_ROLLBACK_PROMOTION}
  namespace: ${PROJECT}
  annotations:
    kargo.akuity.io/rollback: "${ORIGIN_KEY}"
spec:
  stage: multi-origin-holds
  freight: ${FREIGHT_V001}
  steps:
  - uses: set-metadata
    config:
      updates:
      - kind: Stage
        name: multi-origin-holds
        values:
          manualTest: auto-promotion-hold
          lastStory: multi-origin-holds
---
apiVersion: kargo.akuity.io/v1alpha1
kind: Promotion
metadata:
  name: ${API_ROLLBACK_PROMOTION}
  namespace: ${PROJECT}
  annotations:
    kargo.akuity.io/rollback: "${API_ORIGIN_KEY}"
spec:
  stage: multi-origin-holds
  freight: ${API_FREIGHT_V001}
  steps:
  - uses: set-metadata
    config:
      updates:
      - kind: Stage
        name: multi-origin-holds
        values:
          manualTest: auto-promotion-hold
          lastStory: multi-origin-holds
YAML

# Wait for both rollback Promotions to succeed so holds are derived.
echo "Waiting for rollback Promotions to succeed..."
for promo in "${FRONTEND_ROLLBACK_PROMOTION}" "${API_ROLLBACK_PROMOTION}"; do
  for _ in {1..60}; do
    phase=$(kubectl get promotion "${promo}" -n "${PROJECT}" \
      -o jsonpath='{.status.phase}' 2>/dev/null || echo "")
    if [[ "${phase}" == "Succeeded" || "${phase}" == "Failed" || "${phase}" == "Error" ]]; then
      echo "  ${promo}: ${phase}"
      break
    fi
    sleep 2
  done
done

# Annotate the succeeded rollback Promotions so the Stage controller derives
# holds. The webhook only stamps kargo.akuity.io/auto-promotion-hold during
# Create when auto-promotion is enabled; since we seed with it disabled, we add
# the annotation manually here.
kubectl annotate promotion "${FRONTEND_ROLLBACK_PROMOTION}" \
  --namespace "${PROJECT}" \
  --overwrite \
  "kargo.akuity.io/auto-promotion-hold=${ORIGIN_KEY}"
kubectl annotate promotion "${API_ROLLBACK_PROMOTION}" \
  --namespace "${PROJECT}" \
  --overwrite \
  "kargo.akuity.io/auto-promotion-hold=${API_ORIGIN_KEY}"

# The Stage watches Promotions only on phase-change events. Since the rollback
# Promotions are already Succeeded (phase didn't change), we must force a Stage
# reconcile by setting the refresh annotation so the controller picks up the
# newly-added hold annotations.
kubectl annotate stage multi-origin-holds \
  --namespace "${PROJECT}" \
  --overwrite \
  "kargo.akuity.io/refresh=${NOW}"

# Wait for Stage controller to derive holds from the rollback Promotions.
echo "Waiting for auto-promotion holds to be set..."
holds_set=false
for _ in {1..60}; do
  holds=$(kubectl get stage multi-origin-holds -n "${PROJECT}" \
    -o jsonpath='{.status.autoPromotionHolds}' 2>/dev/null || echo "")
  if [[ -n "${holds}" && "${holds}" != "null" ]]; then
    echo "  holds set: ${holds}"
    holds_set=true
    break
  fi
  sleep 2
done
if [[ "${holds_set}" != "true" ]]; then
  echo "  WARNING: timed out waiting for holds; proceeding anyway"
fi

# Enable auto-promotion now that holds are in place. The holds will block
# auto-promotion for the held origins; single-origin-hold has no hold.
kubectl apply -f "${SCRIPT_DIR}/project-config.yaml"

kubectl get project "${PROJECT}"
kubectl get warehouse,stage,freight,promotion --namespace "${PROJECT}"
