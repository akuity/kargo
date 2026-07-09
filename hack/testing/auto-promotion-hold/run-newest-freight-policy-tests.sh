#!/usr/bin/env bash
# End-to-end test runner for the NewestFreight auto-promotion hold scenarios.
#
# Prerequisites: harness applied via apply-newest-freight-policy-harness.sh
# Run from the repo root.
set -uo pipefail

NS="auto-promotion-hold"
PASS=0
FAIL=0

# Build the CLI once so repeated calls use the binary rather than recompiling.
CLI_BIN=$(mktemp)
trap 'rm -f "${CLI_BIN}"' EXIT
echo "Building CLI..."
go build -o "${CLI_BIN}" ./cmd/cli/
echo "Done."
echo ""

# ── Helpers ──────────────────────────────────────────────────────────────────

GREEN='\033[0;32m'
RED='\033[0;31m'
BOLD='\033[1m'
NC='\033[0m'

log_pass() {
  echo -e "  ${GREEN}✓ PASS${NC} $1"
  PASS=$((PASS + 1))
}

log_fail() {
  echo -e "  ${RED}✗ FAIL${NC} $1${2:+  ($2)}"
  FAIL=$((FAIL + 1))
}

# Promote and print the created Promotion name to stdout; status to stderr.
do_promote() {
  local output
  output=$("${CLI_BIN}" promote "$@" 2>&1)
  local name
  name=$(echo "$output" | grep 'promotion created' \
    | grep -oE 'kargo\.akuity\.io/\S+' \
    | sed 's|kargo\.akuity\.io/||' \
    | tr -d ' ')
  echo "  → ${name}" >&2
  echo "${name}"
}

# Poll until terminal phase or timeout (default 120 s).
wait_for_promotion() {
  local name="$1"
  local timeout="${2:-120}"
  local elapsed=0
  echo -n "  Waiting for ${name}" >&2
  while [[ ${elapsed} -lt ${timeout} ]]; do
    local phase
    phase=$(kubectl get promotion "${name}" -n "${NS}" \
      -o jsonpath='{.status.phase}' 2>/dev/null || true)
    case "${phase}" in
      Succeeded)
        echo " ✓" >&2
        return 0
        ;;
      Failed|Errored|Aborted)
        echo " ✗ ${phase}" >&2
        return 1
        ;;
    esac
    echo -n "." >&2
    sleep 3
    elapsed=$((elapsed + 3))
  done
  echo " timed out" >&2
  return 1
}

get_holds() {
  kubectl get stage "$1" -n "${NS}" \
    -o jsonpath='{.status.autoPromotionHolds}' 2>/dev/null \
    | jq -c . 2>/dev/null || echo "null"
}

holds_empty() {
  local h
  h=$(get_holds "$1")
  [[ -z "${h}" || "${h}" == "null" || "${h}" == "{}" ]]
}

holds_has_key() {
  local h
  h=$(get_holds "$1")
  echo "${h}" | jq -e --arg k "$2" '.[$k] != null' > /dev/null 2>&1
}

holds_count() {
  local h
  h=$(get_holds "$1")
  if [[ -z "${h}" || "${h}" == "null" ]]; then echo "0"; return; fi
  echo "${h}" | jq 'length'
}

promo_count_for_stage() {
  kubectl get promotions -n "${NS}" -o json \
    | jq --arg s "$1" '[.items[] | select(.spec.stage == $s)] | length'
}

# ── Scenarios ────────────────────────────────────────────────────────────────

echo -e "${BOLD}════════════════════════════════════════════${NC}"
echo -e "${BOLD}  NewestFreight Policy E2E Tests${NC}"
echo -e "${BOLD}════════════════════════════════════════════${NC}"
echo ""

# ── H1 ───────────────────────────────────────────────────────────────────────
echo -e "${BOLD}H1${NC}: Auto-promotion has no hold annotation"
if holds_empty "single-origin-hold"; then
  log_pass "H1 — no active hold on fresh cluster"
else
  log_fail "H1 — unexpected hold" "$(get_holds single-origin-hold)"
fi
echo ""

# ── A1 ───────────────────────────────────────────────────────────────────────
echo -e "${BOLD}A1${NC}: Promote non-candidate → hold established"
promo_a1=$(do_promote --project "${NS}" --stage single-origin-hold \
  --freight-alias frontend-v001)
if wait_for_promotion "${promo_a1}"; then
  if holds_has_key "single-origin-hold" "Warehouse/auto-hold"; then
    log_pass "A1 — Warehouse/auto-hold established"
  else
    log_fail "A1 — hold not found" "$(get_holds single-origin-hold)"
  fi
else
  log_fail "A1 — promotion did not succeed"
fi
echo ""

# ── B1 ───────────────────────────────────────────────────────────────────────
echo -e "${BOLD}B1${NC}: Hold blocks auto-promotion"
sleep 5
count=$(promo_count_for_stage "single-origin-hold")
if [[ "${count}" -eq 2 ]]; then
  log_pass "B1 — no spurious auto-promotion fired (count=2)"
else
  log_fail "B1 — expected 2 promotions, got ${count}"
fi
echo ""

# ── B4 ───────────────────────────────────────────────────────────────────────
echo -e "${BOLD}B4${NC}: Hold persists after Promotion deleted"
hold_promo=$(kubectl get promotions -n "${NS}" -o json | jq -r '
  .items[]
  | select(
      .metadata.annotations["kargo.akuity.io/auto-promotion-hold"] != null
      and .status.phase == "Succeeded"
      and .spec.stage == "single-origin-hold"
    )
  | .metadata.name' | head -1)
echo "  Deleting: ${hold_promo}"
kubectl delete promotion "${hold_promo}" -n "${NS}" > /dev/null 2>&1
sleep 3
if holds_has_key "single-origin-hold" "Warehouse/auto-hold"; then
  log_pass "B4 — hold persists after Promotion deleted"
else
  log_fail "B4 — hold disappeared" "$(get_holds single-origin-hold)"
fi
echo ""

# ── C1 ───────────────────────────────────────────────────────────────────────
echo -e "${BOLD}C1${NC}: Promote candidate by name → hold cleared"
promo_c1=$(do_promote --project "${NS}" --stage single-origin-hold \
  --freight-alias frontend-v002)
if wait_for_promotion "${promo_c1}"; then
  if holds_empty "single-origin-hold"; then
    log_pass "C1 — hold cleared after candidate promotion by name"
  else
    log_fail "C1 — hold still present" "$(get_holds single-origin-hold)"
  fi
else
  log_fail "C1 — promotion did not succeed"
fi
echo ""

# ── C3 ───────────────────────────────────────────────────────────────────────
echo -e "${BOLD}C3${NC}: Promote by warehouse → hold cleared (race-free path)"
echo "  Re-establishing hold..."
promo_c3a=$(do_promote --project "${NS}" --stage single-origin-hold \
  --freight-alias frontend-v001)
if wait_for_promotion "${promo_c3a}"; then
  echo "  Clearing via warehouse..."
  promo_c3b=$(do_promote --project "${NS}" --stage single-origin-hold \
    --warehouse auto-hold)
  if wait_for_promotion "${promo_c3b}"; then
    if holds_empty "single-origin-hold"; then
      log_pass "C3 — hold cleared via warehouse"
    else
      log_fail "C3 — hold still present" "$(get_holds single-origin-hold)"
    fi
  else
    log_fail "C3 — warehouse promotion failed"
  fi
else
  log_fail "C3 — re-establish promotion failed"
fi
echo ""

# ── C4 ───────────────────────────────────────────────────────────────────────
echo -e "${BOLD}C4${NC}: Rotated candidate clears hold"
echo "  Re-establishing hold..."
promo_c4a=$(do_promote --project "${NS}" --stage single-origin-hold \
  --freight-alias frontend-v001)
if wait_for_promotion "${promo_c4a}"; then
  echo "  Clearing with current candidate..."
  promo_c4b=$(do_promote --project "${NS}" --stage single-origin-hold \
    --freight-alias frontend-v002)
  if wait_for_promotion "${promo_c4b}"; then
    if holds_empty "single-origin-hold"; then
      log_pass "C4 — hold cleared by current candidate"
    else
      log_fail "C4 — hold still present" "$(get_holds single-origin-hold)"
    fi
  else
    log_fail "C4 — candidate promotion failed"
  fi
else
  log_fail "C4 — re-establish promotion failed"
fi
echo ""

# ── D1 ───────────────────────────────────────────────────────────────────────
echo -e "${BOLD}D1${NC}: No duplicate auto-promotion after hold cleared"
count_before=$(promo_count_for_stage "single-origin-hold")
sleep 10
count_after=$(promo_count_for_stage "single-origin-hold")
if [[ "${count_before}" -eq "${count_after}" ]]; then
  log_pass "D1 — no new auto-promotion fired (count stable at ${count_after})"
else
  log_fail "D1 — unexpected new promotions (${count_before} → ${count_after})"
fi
echo ""

# ── E1 ───────────────────────────────────────────────────────────────────────
echo -e "${BOLD}E1${NC}: Releasing one origin does not affect the other"
start_count=$(holds_count "multi-origin-holds")
if [[ "${start_count}" -eq 2 ]]; then
  promo_e1=$(do_promote --project "${NS}" --stage multi-origin-holds \
    --warehouse auto-hold)
  if wait_for_promotion "${promo_e1}"; then
    after_count=$(holds_count "multi-origin-holds")
    if [[ "${after_count}" -eq 1 ]] \
        && holds_has_key "multi-origin-holds" "Warehouse/auto-hold-api"; then
      log_pass "E1 — only Warehouse/auto-hold-api remains"
    else
      log_fail "E1 — unexpected hold state" "$(get_holds multi-origin-holds)"
    fi
  else
    log_fail "E1 — promotion failed"
  fi
else
  log_fail "E1 — expected 2 holds at start, got ${start_count}" \
    "$(get_holds multi-origin-holds)"
fi
echo ""

# ── E2 ───────────────────────────────────────────────────────────────────────
echo -e "${BOLD}E2${NC}: Clear one origin, establish hold on another"
promo_e2a=$(do_promote --project "${NS}" --stage multi-origin-holds \
  --warehouse auto-hold-api)
promo_e2b=$(do_promote --project "${NS}" --stage multi-origin-holds \
  --freight-alias frontend-v001)
e2_ok=true
wait_for_promotion "${promo_e2a}" || { log_fail "E2 — clear auto-hold-api failed"; e2_ok=false; }
wait_for_promotion "${promo_e2b}" || { log_fail "E2 — establish auto-hold failed"; e2_ok=false; }
if [[ "${e2_ok}" == "true" ]]; then
  e2_count=$(holds_count "multi-origin-holds")
  if [[ "${e2_count}" -eq 1 ]] \
      && holds_has_key "multi-origin-holds" "Warehouse/auto-hold"; then
    log_pass "E2 — Warehouse/auto-hold present, Warehouse/auto-hold-api cleared"
  else
    log_fail "E2 — unexpected hold state" "$(get_holds multi-origin-holds)"
  fi
fi
echo ""

# ── G2 ───────────────────────────────────────────────────────────────────────
echo -e "${BOLD}G2${NC}: Promote by warehouse with no active hold → no hold created"
promo_g2=$(do_promote --project "${NS}" --stage single-origin-hold \
  --warehouse auto-hold)
if wait_for_promotion "${promo_g2}"; then
  if holds_empty "single-origin-hold"; then
    log_pass "G2 — no hold created by candidate warehouse promotion"
  else
    log_fail "G2 — unexpected hold" "$(get_holds single-origin-hold)"
  fi
else
  log_fail "G2 — promotion failed"
fi
echo ""

# ── Summary ──────────────────────────────────────────────────────────────────
echo -e "${BOLD}════════════════════════════════════════════${NC}"
if [[ "${FAIL}" -eq 0 ]]; then
  echo -e "${BOLD}  ${GREEN}All ${PASS} scenarios passed${NC}${BOLD}.${NC}"
else
  echo -e "${BOLD}  ${GREEN}${PASS} passed${NC}${BOLD}, ${RED}${FAIL} failed${NC}${BOLD}.${NC}"
fi
echo -e "${BOLD}════════════════════════════════════════════${NC}"
[[ "${FAIL}" -eq 0 ]]
