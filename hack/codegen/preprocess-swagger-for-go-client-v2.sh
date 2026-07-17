#!/usr/bin/env bash

set -euo pipefail

# preprocess-swagger-for-go-client-v2.sh
#
# Produces an openapi-generator-only derivative of swagger.json. The
# canonical swagger.json (and the UI's orval-generated TypeScript client
# that reads it) is NEVER modified by this -- same discipline as
# hack/codegen/fix-swagger-spec.sh's Pass 5, which does the equivalent job
# for the (old, go-swagger-based) Go client.
#
# Three fixes, all scoped to this generator-input-only copy:
#
# 1. Keep only the first tag on each operation. 113 operations in our spec
#    carry multiple tags; openapi-generator generates one API file per tag
#    and duplicates/redeclares request types for multi-tagged operations,
#    and the output doesn't compile.
#
# 2. Blank the Quantity, IntOrString, and V1MicroTime definitions to
#    typeless (description-only) schemas. swag (which produced swagger.json)
#    reflected these Go structs directly and ignored their custom
#    MarshalJSON/UnmarshalJSON -- their real wire formats are scalars
#    ("100m", 5 or "20%", an RFC3339 string), not the objects the spec
#    describes. This is a PRE-EXISTING defect in swagger.json's description
#    of these three types (present for both the old and new Go clients, and
#    for the TypeScript client too) -- it silently corrupts Quantity values
#    and hard-fails to unmarshal IntOrString/V1MicroTime in the OLD
#    (go-swagger) client today. Typeless schemas render as interface{} in
#    openapi-generator's Go output, which round-trips any of these formats
#    losslessly.
#
# 3. Strip "type": "object" from bare object schemas that have neither
#    "properties" nor "additionalProperties". These are exactly the
#    apiextensions.JSON-style opaque fields (Health.output,
#    PromotionStep.config, PromotionStatus.state, etc.) -- Health.output in
#    particular carries a JSON ARRAY on the wire (see
#    pkg/health/aggregating_checker_test.go), so without this step
#    openapi-generator would infer map[string]interface{} for these fields,
#    which is a worse (and wrong) typing than the interface{} typeless
#    schemas render as.

if [[ $# -ne 2 ]]; then
    echo "Usage: preprocess-swagger-for-go-client-v2.sh <input-swagger.json> <output-path>" >&2
    exit 1
fi

INPUT_FILE="$1"
OUTPUT_FILE="$2"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
JQ="${SCRIPT_DIR}/../bin/jq"

if [[ ! -x "$JQ" ]]; then
    echo "Error: jq not found at $JQ -- run 'make install-jq' first" >&2
    exit 1
fi

"$JQ" '
  # --- 1. Keep only the first tag per operation -----------------------------
  .paths |= map_values(map_values(
    if (type == "object" and .tags?) then .tags |= [.[0]] else . end
  )) |

  # --- 2. Blank Quantity / IntOrString / V1MicroTime to typeless schemas ----
  .definitions.Quantity = {
    "description": "Kubernetes resource quantity; serializes as a string (e.g. \"100m\", \"128Mi\")"
  } |
  .definitions.IntOrString = {
    "description": "Serializes as a bare integer or string"
  } |
  .definitions.V1MicroTime = {
    "description": "Serializes as an RFC3339 string with microseconds"
  } |

  # --- 3. Strip "type":"object" from bare (opaque) object schemas ----------
  walk(
    if (type == "object" and .type? == "object"
        and (has("properties") | not)
        and (has("additionalProperties") | not))
    then del(.type)
    else . end
  )
' "$INPUT_FILE" > "$OUTPUT_FILE"

echo "Wrote openapi-generator-only swagger spec to $OUTPUT_FILE"
