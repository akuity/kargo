#!/usr/bin/env bash

set -euo pipefail

# flatten-nullable-refs.sh
#
# Produces a Go-client-only derivative of swagger.json that makes optional
# object-typed fields generate as pointers in the go-swagger client.
#
# Background: swag wraps any $ref'd property that has a doc comment in
# `allOf: [{"$ref": ...}]`, because raw JSON Schema forbids sibling keywords
# (like "description") next to "$ref". go-swagger renders allOf-wrapped
# properties as value-typed embedded structs regardless of x-nullable, so
# Go's encoding/json -- which never treats a non-pointer struct as "empty" --
# always marshals these fields, producing phantom empty objects (e.g.
# "status":{}) even when the field was never set.
#
# This script flattens `{"allOf":[{"$ref":X}],"description":D}` down to
# `{"$ref":X,"description":D}` for every property that is NOT in its
# definition's `required` array. go-swagger then generates a pointer for
# that property (its default behavior for a bare $ref), fixing the
# phantom-empty-object bug for optional fields. Required fields are left
# untouched, matching today's behavior.
#
# The output of this script is used ONLY to generate the Go client. The
# canonical swagger.json (produced by fix-swagger-spec.sh) is left
# unmodified, since it also feeds the UI's TypeScript client via orval, and
# orval -- unlike go-swagger -- follows the strict OpenAPI rule that
# sibling keywords next to "$ref" are ignored, so flattening it there would
# silently drop field doc comments from the generated TypeScript models.

if [[ $# -ne 2 ]]; then
    echo "Usage: flatten-nullable-refs.sh <input-swagger.json> <output-path>" >&2
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

# --- Pass 1: Flatten single-$ref allOf wrappers on optional properties ------

"$JQ" '
  .definitions |= map_values(
    (.required // []) as $req |
    if has("properties") then
      .properties |= with_entries(
        (.key) as $k |
        if (.value.allOf? != null) and
           ((.value.allOf | length) == 1) and
           ((.value.allOf[0] | keys) == ["$ref"]) and
           ((.value | keys - ["allOf","description"]) == []) and
           ($req | index($k) == null)
        then .value = (
          {"$ref": .value.allOf[0]["$ref"]} +
          (if .value.description then {description: .value.description} else {} end)
        )
        else . end
      )
    else . end
  )
' "$INPUT_FILE" > "${OUTPUT_FILE}.tmp"

# --- Pass 2: Validate no broken $ref pointers remain -------------------------

"$JQ" -e '
  [.. | objects | select(has("$ref")) | .["$ref"] |
   select(startswith("#/definitions/")) |
   split("/") | last] as $refs |
  (.definitions | keys) as $defs |
  ($refs - $defs) | if length > 0 then
    error("Broken $ref pointers found: \(join(", "))")
  else true end
' "${OUTPUT_FILE}.tmp" > /dev/null

mv "${OUTPUT_FILE}.tmp" "$OUTPUT_FILE"

echo "Wrote Go-client-only swagger spec (nullable optional refs flattened) to $OUTPUT_FILE"
