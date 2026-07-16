#!/usr/bin/env bash

set -euo pipefail

# fix-swagger-spec.sh
#
# Post-processes swagger.json to:
#
# 1. Rename verbose definition keys (derived from full Go package paths) to
#    short, ergonomic names and update all $ref pointers accordingly. This
#    benefits both the generated Go client (go-swagger) and the TypeScript
#    client (orval).
#
# 2. Fix []byte field representations. Go's []byte is JSON-serialized as a
#    base64-encoded string, but swag expands it to {"type":"array","items":
#    {"type":"integer","format":"int32"}} because byte is an alias for uint8.
#    This script rewrites those fields to {"type":"string","format":"byte"}
#    so that go-swagger generates the correct Go types.
#
# 3. Add `required` arrays derived from +kubebuilder:validation:Required
#    markers. swag copies these markers into description strings but does not
#    translate them into the OpenAPI `required` array, causing generated
#    TypeScript clients to treat all fields as optional.
#
# If a second argument is given, it's treated as an output path for a
# Go-client-only derivative of the spec (see Pass 5 below) -- swagger.json
# itself is left as produced by Passes 1-4 either way.

SWAGGER_FILE="${1:?Usage: fix-swagger-spec.sh <swagger.json> [go-client-output-path]}"
GO_CLIENT_FILE="${2:-}"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
JQ="${SCRIPT_DIR}/../bin/jq"

if [[ ! -x "$JQ" ]]; then
    echo "Error: jq not found at $JQ -- run 'make install-jq' first" >&2
    exit 1
fi

# --- Pass 1: Rename verbose definition keys and update $ref pointers ----------

"$JQ" '
  # Compute a short name for a swagger definition key.
  def rename_key:
    if startswith("github_com_akuity_kargo_api_v1alpha1.") then
      split(".") | last
    elif startswith("github_com_akuity_kargo_api_stubs_rollouts_v1alpha1.") then
      "Rollouts" + (split(".") | last)
    elif startswith("github_com_akuity_kargo_api_rbac_v1alpha1.") then
      "Rbac" + (split(".") | last)
    elif startswith("k8s_io_api_core_v1.") then
      "V1" + (split(".") | last)
    elif startswith("v1.") then
      "V1" + .[3:]
    elif startswith("intstr.") then
      split(".") | last
    elif startswith("resource.") then
      split(".") | last
    else
      .
    end;

  # Build old_name -> new_name mapping.
  (.definitions | keys | map({(.) : (. | rename_key)}) | add) as $map |

  # Collision detection: ensure all new names are unique.
  ([$map | to_entries[].value] | unique | length) as $unique_count |
  (if ($map | length) != $unique_count then
    # Find and report the collisions.
    [$map | to_entries[] | .value] | group_by(.) | map(select(length > 1) | .[0]) |
    error("Definition name collision detected for: \(join(", "))")
  else . end) |

  # Rename definition keys.
  .definitions |= (
    to_entries | map(.key = $map[.key]) | from_entries
  ) |

  # Rewrite all $ref pointers to use new names.
  walk(
    if type == "object" and has("$ref") and (.["$ref"] | startswith("#/definitions/")) then
      .["$ref"] |= (
        split("/") | last | . as $old |
        "#/definitions/" + ($map[$old] // $old)
      )
    else . end
  )
' "$SWAGGER_FILE" > "${SWAGGER_FILE}.tmp" && mv "${SWAGGER_FILE}.tmp" "$SWAGGER_FILE"

echo "Renamed swagger definitions to short names."

# --- Pass 2: Fix map[string][]byte fields ------------------------------------
#
# Fix Secret.data, ConfigMap.binaryData, and AnalysisRun MetricProvider.plugin.
# These appear as additionalProperties with array-of-int32; they should be
# additionalProperties with string format byte.

"$JQ" '
  # Fix Secret.data
  (if .definitions["V1Secret"].properties.data.additionalProperties then
    .definitions["V1Secret"].properties.data.additionalProperties = {"type": "string", "format": "byte"}
  else . end) |
  # Fix ConfigMap.binaryData
  (if .definitions["V1ConfigMap"].properties.binaryData.additionalProperties then
    .definitions["V1ConfigMap"].properties.binaryData.additionalProperties = {"type": "string", "format": "byte"}
  else . end) |
  # Fix AnalysisRun MetricProvider.plugin
  (if .definitions["RolloutsMetricProvider"].properties.plugin.additionalProperties then
    .definitions["RolloutsMetricProvider"].properties.plugin.additionalProperties = {"type": "string", "format": "byte"}
  else . end)
' "$SWAGGER_FILE" > "${SWAGGER_FILE}.tmp" && mv "${SWAGGER_FILE}.tmp" "$SWAGGER_FILE"

# --- Pass 3: Add `required` arrays from kubebuilder validation markers -------
#
# swag copies +kubebuilder:validation:Required into description strings but
# does not translate them into the OpenAPI `required` array. Derive the
# required array from descriptions automatically.

"$JQ" '
  .definitions |= map_values(
    . as $def |
    (
      $def.properties // {} |
      to_entries |
      map(select(
        .value.description != null and
        (.value.description | contains("+kubebuilder:validation:Required"))
      )) |
      map(.key)
    ) as $required |
    if ($required | length) > 0
    then .required = $required
    else . end
  )
' "$SWAGGER_FILE" > "${SWAGGER_FILE}.tmp" && mv "${SWAGGER_FILE}.tmp" "$SWAGGER_FILE"

echo "Added required arrays from kubebuilder validation markers."

# --- Pass 4: Validate no broken $ref pointers remain -------------------------

"$JQ" -e '
  [.. | objects | select(has("$ref")) | .["$ref"] |
   select(startswith("#/definitions/")) |
   split("/") | last] as $refs |
  (.definitions | keys) as $defs |
  ($refs - $defs) | if length > 0 then
    error("Broken $ref pointers found: \(join(", "))")
  else true end
' "$SWAGGER_FILE" > /dev/null

echo "Swagger spec post-processing complete."

# --- Pass 5: Produce a Go-client-only derivative with nullable refs ----------
#
# swag wraps any $ref'd property that has a doc comment in
# `allOf: [{"$ref": ...}]`, because raw JSON Schema forbids sibling keywords
# (like "description") next to "$ref". go-swagger renders allOf-wrapped
# properties as value-typed embedded structs regardless of x-nullable, so
# Go's encoding/json -- which never treats a non-pointer struct as "empty" --
# always marshals these fields, producing phantom empty objects (e.g.
# "status":{}) even when the field was never set.
#
# This flattens `{"allOf":[{"$ref":X}],"description":D}` down to
# `{"$ref":X,"description":D}` for every property that is NOT in its
# definition's `required` array. go-swagger then generates a pointer for
# that property (its default behavior for a bare $ref), fixing the
# phantom-empty-object bug for optional fields. Required fields are left
# untouched, matching today's behavior.
#
# This derivative is used ONLY to generate the Go client. swagger.json
# itself (produced by Passes 1-4 above) is left unmodified, since it also
# feeds the UI's TypeScript client via orval, and orval -- unlike
# go-swagger -- follows the strict OpenAPI rule that sibling keywords next
# to "$ref" are ignored, so flattening it there would silently drop field
# doc comments from the generated TypeScript models.

if [[ -n "$GO_CLIENT_FILE" ]]; then
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
    ' "$SWAGGER_FILE" > "${GO_CLIENT_FILE}.tmp"

    "$JQ" -e '
      [.. | objects | select(has("$ref")) | .["$ref"] |
       select(startswith("#/definitions/")) |
       split("/") | last] as $refs |
      (.definitions | keys) as $defs |
      ($refs - $defs) | if length > 0 then
        error("Broken $ref pointers found: \(join(", "))")
      else true end
    ' "${GO_CLIENT_FILE}.tmp" > /dev/null

    mv "${GO_CLIENT_FILE}.tmp" "$GO_CLIENT_FILE"

    echo "Wrote Go-client-only swagger spec (nullable optional refs flattened) to $GO_CLIENT_FILE"
fi
