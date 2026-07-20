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

# --- Step 4: write our own go.mod (withGoMod=false above) and tidy -----------
cat > "${OUT_DIR}/go.mod" <<'EOF'
module github.com/akuity/kargo/pkg/x/client/generated

go 1.26.0
EOF
(cd "${OUT_DIR}" && go mod tidy && go build ./...)

echo "OK: pkg/x/client/generated regenerated."
