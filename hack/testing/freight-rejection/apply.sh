#!/usr/bin/env bash
set -euo pipefail

PROJECT=freight-rejection
MANUAL_ORIGIN_KEY=Warehouse/manual
FRONTEND_ORIGIN_KEY=Warehouse/frontend
MANUAL_IMAGE_REPO=public.ecr.aws/docker/library/busybox
FRONTEND_IMAGE_REPO=public.ecr.aws/nginx/nginx
MANUAL_GOOD=803bc6d85d2e67d2f4ae8d5377557697026acf66
MANUAL_APPROVED=ccac9166d8776fe89c7d56130e4e4eac43d67880
FRONTEND_CURRENT=0044230c2e53ad2f5512a139d080f3370f4e6f84
FRONTEND_FALLBACK=fb3165f8155d864b6133171bb4e1dcb10611575a
FRONTEND_REJECTED=fb556073349ba968d34d884ce7e3c47b3ee407a1
LEGACY_FRONTEND_CURRENT=5fbee33ae2e0e1d65272b75902651b684179b3bf
LEGACY_FRONTEND_FALLBACK=0a36b7d53f06329dd9324396579e798d5a866a91
LEGACY_FRONTEND_REJECTED=fae10101d9d57a526c8c26a436d0f5f80cd5b7d0
PENDING_PROMOTION=pending-promo-lab.rejected-freight

SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
NOW=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
REFRESH_TOKEN=$(date -u +"%Y%m%d%H%M%S")

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
kubectl delete stage manual-lab reject-lab hold-resume-lab pending-promo-lab \
  --namespace "${PROJECT}" \
  --ignore-not-found=true \
  --wait=true
kubectl delete freight \
  "${MANUAL_GOOD}" \
  "${MANUAL_APPROVED}" \
  "${FRONTEND_CURRENT}" \
  "${FRONTEND_FALLBACK}" \
  "${FRONTEND_REJECTED}" \
  "${LEGACY_FRONTEND_CURRENT}" \
  "${LEGACY_FRONTEND_FALLBACK}" \
  "${LEGACY_FRONTEND_REJECTED}" \
  --namespace "${PROJECT}" \
  --ignore-not-found=true \
  --wait=true
kubectl delete warehouse manual frontend \
  --namespace "${PROJECT}" \
  --ignore-not-found=true \
  --wait=true

kubectl apply -f "${SCRIPT_DIR}/resources.yaml"

for freight in \
  "${MANUAL_GOOD}" \
  "${MANUAL_APPROVED}" \
  "${FRONTEND_CURRENT}" \
  "${FRONTEND_FALLBACK}" \
  "${FRONTEND_REJECTED}"; do
  kubectl patch freight "${freight}" \
    --namespace "${PROJECT}" \
    --subresource=status \
    --type=json \
    --patch='[{"op":"replace","path":"/status","value":{}}]' \
    >/dev/null || true
done

kubectl patch freight "${MANUAL_APPROVED}" \
  --namespace "${PROJECT}" \
  --subresource=status \
  --type=merge \
  --patch "$(cat <<JSON
{
  "status": {
    "approvedFor": {
      "manual-lab": {"approvedAt": "${NOW}"}
    }
  }
}
JSON
)"

kubectl patch freight "${FRONTEND_CURRENT}" \
  --namespace "${PROJECT}" \
  --subresource=status \
  --type=merge \
  --patch "$(cat <<JSON
{
  "status": {
    "currentlyIn": {
      "reject-lab": {"since": "${NOW}"},
      "hold-resume-lab": {"since": "${NOW}"},
      "pending-promo-lab": {"since": "${NOW}"}
    },
    "verifiedIn": {
      "reject-lab": {"verifiedAt": "${NOW}"},
      "hold-resume-lab": {"verifiedAt": "${NOW}"},
      "pending-promo-lab": {"verifiedAt": "${NOW}"}
    }
  }
}
JSON
)"

kubectl patch stage pending-promo-lab \
  --namespace "${PROJECT}" \
  --subresource=status \
  --type=json \
  --patch "$(cat <<JSON
[
  {
    "op": "replace",
    "path": "/status",
    "value": {
      "conditions": [
        {
          "type": "Ready",
          "status": "False",
          "reason": "Verifying",
          "message": "Current Freight is pending verification",
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
          "status": "Unknown",
          "reason": "VerificationInProgress",
          "message": "Freight is pending verification",
          "lastTransitionTime": "${NOW}"
        }
      ],
      "freightSummary": "frontend-v001-current",
      "health": {"status": "Healthy"},
      "freightHistory": [
        {
          "id": "pending-promo-lab-current",
          "items": {
            "${FRONTEND_ORIGIN_KEY}": {
              "name": "${FRONTEND_CURRENT}",
              "origin": {"kind": "Warehouse", "name": "frontend"},
              "images": [{"repoURL": "${FRONTEND_IMAGE_REPO}", "tag": "1.26.31"}]
            }
          }
        }
      ]
    }
  }
]
JSON
)"

kubectl apply -f - <<YAML
apiVersion: kargo.akuity.io/v1alpha1
kind: Promotion
metadata:
  name: ${PENDING_PROMOTION}
  namespace: ${PROJECT}
spec:
  stage: pending-promo-lab
  freight: ${FRONTEND_REJECTED}
  source: nonAuto
  steps:
  - uses: set-metadata
    config:
      updates:
      - kind: Stage
        name: pending-promo-lab
        values:
          manualTest: freight-rejection
          lastStory: pending-promo-lab
YAML

kubectl patch freight "${FRONTEND_REJECTED}" \
  --namespace "${PROJECT}" \
  --subresource=status \
  --type=merge \
  --patch "$(cat <<JSON
{
  "status": {
    "approvedFor": {
      "reject-lab": {"approvedAt": "${NOW}"}
    },
    "rejected": {
      "rejectedAt": "${NOW}",
      "actor": "user:fixture@example.com",
      "reason": "Known-bad build seeded by hack/testing/freight-rejection."
    }
  }
}
JSON
)"

kubectl patch stage pending-promo-lab \
  --namespace "${PROJECT}" \
  --subresource=status \
  --type=merge \
  --patch "$(cat <<JSON
{
  "status": {
    "currentPromotion": {
      "name": "${PENDING_PROMOTION}",
      "freight": {
        "name": "${FRONTEND_REJECTED}",
        "origin": {"kind": "Warehouse", "name": "frontend"},
        "images": [{"repoURL": "${FRONTEND_IMAGE_REPO}", "tag": "1.26.95"}]
      }
    }
  }
}
JSON
)"

kubectl patch stage manual-lab \
  --namespace "${PROJECT}" \
  --subresource=status \
  --type=json \
  --patch "$(cat <<JSON
[
  {
    "op": "replace",
    "path": "/status",
    "value": {
      "conditions": [
        {
          "type": "Ready",
          "status": "True",
          "reason": "Ready",
          "message": "Seeded and ready for manual Freight rejection testing",
          "lastTransitionTime": "${NOW}"
        },
        {
          "type": "Healthy",
          "status": "True",
          "reason": "Healthy",
          "message": "Stage is healthy (performed 0 health checks)",
          "lastTransitionTime": "${NOW}"
        }
      ],
      "health": {"status": "Healthy"}
    }
  }
]
JSON
)"

kubectl patch stage reject-lab \
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
      "freightSummary": "frontend-v001-current",
      "health": {"status": "Healthy"},
      "freightHistory": [
        {
          "id": "reject-lab-current",
          "items": {
            "${FRONTEND_ORIGIN_KEY}": {
              "name": "${FRONTEND_CURRENT}",
              "origin": {"kind": "Warehouse", "name": "frontend"},
              "images": [{"repoURL": "${FRONTEND_IMAGE_REPO}", "tag": "1.26.31"}]
            }
          },
          "verificationHistory": [
            {
              "id": "manual-seed",
              "phase": "Successful",
              "message": "Seeded verified Freight for rejection testing.",
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

kubectl patch stage hold-resume-lab \
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
      "freightSummary": "frontend-v001-current",
      "health": {"status": "Healthy"},
      "freightHistory": [
        {
          "id": "hold-resume-lab-current",
          "items": {
            "${FRONTEND_ORIGIN_KEY}": {
              "name": "${FRONTEND_CURRENT}",
              "origin": {"kind": "Warehouse", "name": "frontend"},
              "images": [{"repoURL": "${FRONTEND_IMAGE_REPO}", "tag": "1.26.31"}]
            }
          },
          "verificationHistory": [
            {
              "id": "manual-seed",
              "phase": "Successful",
              "message": "Seeded verified Freight for hold resume testing.",
              "startTime": "${NOW}",
              "finishTime": "${NOW}"
            }
          ]
        }
      ],
      "autoPromotionHolds": {
        "${FRONTEND_ORIGIN_KEY}": {
          "freight": {
            "name": "${FRONTEND_CURRENT}",
            "origin": {"kind": "Warehouse", "name": "frontend"},
            "images": [{"repoURL": "${FRONTEND_IMAGE_REPO}", "tag": "1.26.31"}]
          },
          "state": "Active",
          "promotionName": "hold-resume-lab.rollback-demo",
          "promotionUID": "33333333-3333-3333-3333-333333333333",
          "actor": "user:fixture@example.com",
          "reason": "Seeded active hold to test resume-auto-promotion with rejected Freight present.",
          "createdAt": "${NOW}"
        }
      }
    }
  }
]
)"

kubectl apply -f "${SCRIPT_DIR}/project-config.yaml"

kubectl annotate stage reject-lab hold-resume-lab pending-promo-lab \
  --namespace "${PROJECT}" \
  kargo.akuity.io/refresh="${REFRESH_TOKEN}" \
  --overwrite \
  >/dev/null
kubectl annotate promotion "${PENDING_PROMOTION}" \
  --namespace "${PROJECT}" \
  kargo.akuity.io/refresh="${REFRESH_TOKEN}" \
  --overwrite \
  >/dev/null

for _ in {1..10}; do
  phase=$(kubectl get promotion "${PENDING_PROMOTION}" \
    --namespace "${PROJECT}" \
    -o jsonpath='{.status.phase}' 2>/dev/null || true)
  if [[ "${phase}" == "Aborted" ]]; then
    break
  fi
  sleep 1
done

kubectl get project "${PROJECT}"
kubectl get warehouse,stage,freight,promotion --namespace "${PROJECT}"
