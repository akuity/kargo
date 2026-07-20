#!/usr/bin/env bash
set -euo pipefail

# Regenerates the Go client under pkg/x/client/generated from the repo's
# swagger.json using openapi-generator. The pkg/x/ path signals no stability
# promise: the published swagger.json is the contract; consumers should
# generate their own clients from it and use ours at their own risk.
#
# WHY THE WORKAROUNDS BELOW ARE TOLERABLE: swagger.json itself is sound; a
# faithful generator could consume it verbatim. Every accommodation in this
# script compensates for a verified quirk of openapi-generator -- not of our
# spec -- so anyone generating a Go client with the same tool needs the same
# fixes, and anyone with a better tool needs none. We hold no advantage over
# other consumers of the published spec.

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
OUT_DIR="${REPO_ROOT}/pkg/x/client/generated"
JQ="${SCRIPT_DIR}/../bin/jq"
GENERATOR_JAR="${SCRIPT_DIR}/../bin/openapi-generator-cli.jar"

if [[ ! -x "$JQ" ]]; then
    echo "Error: jq not found at $JQ -- run 'make install-jq' first" >&2
    exit 1
fi
if [[ ! -f "$GENERATOR_JAR" ]]; then
    echo "Error: openapi-generator-cli.jar not found at $GENERATOR_JAR -- run 'make install-openapi-generator-cli' first" >&2
    exit 1
fi

WORK_DIR="$(mktemp -d)"
trap 'rm -rf "${WORK_DIR}"' EXIT

# --- Step 1: drop all but the first tag from each operation -------------------
#
# Quirk 1: the generator emits one API file per tag, duplicating operations that
# carry several tags. The duplicate request types do not compile. Tags are
# grouping metadata only; removing them changes no schema semantics.
# swagger.json itself is never modified.
"$JQ" '
  .paths |= map_values(map_values(
    if (type == "object" and .tags?) then .tags |= [.[0]] else . end
  ))
' "${REPO_ROOT}/swagger.json" > "${WORK_DIR}/swagger-go-gen.json"

# --- Step 2: generate ---------------------------------------------------------
#
# Quirk 2 (the type mapping): some fields hold arbitrary JSON and are correctly
# typeless in the spec, but the generator infers `object` for them and would
# emit map[string]interface{} -- which rejects the arrays and scalars those
# fields really carry in some cases (e.g. Health.output is an array on the
# wire). The mapping forces `any`, which round-trips any JSON.
#
# apiTests/modelTests/apiDocs/modelDocs=false: suppress the generator's
# standalone-repo scaffolding (per-operation test stubs, per-model/API
# markdown docs) -- pure diff noise that duplicates the Go doc comments
# already on the generated types and the real fidelity suite in
# pkg/x/client/fidelity, referenced nowhere in this repo.
rm -rf "${OUT_DIR}"
java -jar "$GENERATOR_JAR" generate \
  -i "${WORK_DIR}/swagger-go-gen.json" \
  -g go \
  -o "${OUT_DIR}" \
  --git-user-id akuity --git-repo-id "kargo/pkg/x/client/generated" \
  --type-mappings 'object=any' \
  --additional-properties=packageName=generated,withGoMod=false,generateInterfaces=false,enumClassPrefix=true \
  --global-property apiTests=false,modelTests=false,apiDocs=false,modelDocs=false \
  --skip-validate-spec

# --- Step 3: fix a generator template bug -------------------------------------
#
# Quirk 3: templates render a type's zero value textually as `<type>{}`.
# That is valid Go for maps and structs but not for `any`, so the
# *Ok() getters of the fields mapped in step 2 come out uncompilable. The
# zero value the template meant is nil.
LC_ALL=C find "${OUT_DIR}" -name 'model_*.go' \
  -exec sed -i.bak 's/return any{}, false/return nil, false/g' {} +
find "${OUT_DIR}" -name 'model_*.go.bak' -delete

# --- Step 3b: redact credentials from Debug-mode logging ---------------------
#
# Quirk 4 (security): the generator's callAPI template dumps the FULL raw
# request and response -- headers AND body -- via log.Printf whenever
# Configuration.Debug is true. Request bodies for credential endpoints carry
# real secrets (CreateRepoCredentialsRequest.Password,
# CreateGenericCredentialsRequest.Data, etc.) and SecretKeyRef-shaped model
# fields, so enabling Debug would log them in cleartext -- flagged by CodeQL
# on PR #6647. The Authorization header carries a second class of secret: the
# admin-login flow (pkg/cli/cmd/login/login.go) submits the raw admin
# password as a Bearer credential. Kargo doesn't currently expose a way to set
# Debug=true, but this is a defect in the generator's own template that every
# consumer using this generator inherits, and static analysis correctly flags
# it regardless of current reachability.
#
# Fix combines two layers, matching GitHub's Copilot Autofix suggestions for
# this alert (offered across two review passes):
#  1. Exclude the body from both dumps (`false` instead of `true`). This is
#     the load-bearing fix: Password/SecretKeyRef only ever appear in the
#     body, so excluding it removes the tainted source data before it's ever
#     captured into `dump`, which is what actually satisfies CodeQL's static
#     analysis -- a regex-based sanitizer alone does not, because CodeQL
#     can't verify what a ReplaceAllString call removes and keeps flagging
#     the flow through it regardless.
#  2. Still pass the (now header-only) dump through sanitizeHTTPDump, which
#     regex-redacts any Authorization header line (the admin-login flow
#     submits the raw admin password as a Bearer credential) and, as
#     defense-in-depth, any "password"/"secretKeyRef" JSON field value in
#     case a future header ever carries one.
# awk (portable across BSD/GNU, unlike some sed extensions) inserts the two
# new regexp vars and the sanitizeHTTPDump function; sed then flips the two
# dump calls to exclude the body and wraps both log.Printf calls.
awk '
  /^var \($/ { in_var = 1; print; next }
  in_var && /queryDescape[[:space:]]*= strings\.NewReplacer/ {
    print $0
    print ""
    print "\tsensitiveJSONFieldRegex  = regexp.MustCompile(`(?i)\"(password|secretKeyRef)\"\\s*:\\s*(\"[^\"]*\"|\\{[^}]*\\}|null|[^,\\r\\n}]+)`)"
    print "\tauthorizationHeaderRegex = regexp.MustCompile(`(?im)^(Authorization:\\s*)(.+)$`)"
    next
  }
  in_var && /^\)$/ {
    in_var = 0
    print $0
    print ""
    print "// sanitizeHTTPDump redacts sensitive values (password/secretKeyRef JSON"
    print "// fields, the Authorization header) from a raw HTTP request/response dump"
    print "// before it is logged in Debug mode."
    print "func sanitizeHTTPDump(dump string) string {"
    print "\tredacted := sensitiveJSONFieldRegex.ReplaceAllString(dump, `\"$1\":\"[REDACTED]\"`)"
    print "\tredacted = authorizationHeaderRegex.ReplaceAllString(redacted, `${1}[REDACTED]`)"
    print "\treturn redacted"
    print "}"
    next
  }
  { print }
' "${OUT_DIR}/client.go" > "${OUT_DIR}/client.go.tmp"
mv "${OUT_DIR}/client.go.tmp" "${OUT_DIR}/client.go"

sed -i.bak \
  -e 's/httputil\.DumpRequestOut(request, true)/httputil.DumpRequestOut(request, false)/' \
  -e 's/httputil\.DumpResponse(resp, true)/httputil.DumpResponse(resp, false)/' \
  -e 's/log\.Printf("\\n%s\\n", string(dump))/log.Printf("\\n%s\\n", sanitizeHTTPDump(string(dump)))/g' \
  "${OUT_DIR}/client.go"
rm -f "${OUT_DIR}/client.go.bak"

# --- Step 4: write our own go.mod (withGoMod=false above) and tidy -----------
cat > "${OUT_DIR}/go.mod" <<'EOF'
module github.com/akuity/kargo/pkg/x/client/generated

go 1.26.0
EOF
(cd "${OUT_DIR}" && go mod tidy && go build ./...)

echo "OK: pkg/x/client/generated regenerated."
