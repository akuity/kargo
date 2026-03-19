#!/usr/bin/env bash

set -euo pipefail

# fix-swagger-spec.sh
#
# Post-processes swagger.json to fix type representations that swag gets wrong
# when parsing Go types from Kubernetes libraries.
#
# The main issue is []byte fields: Go's []byte is JSON-serialized as a
# base64-encoded string, but swag expands it to {"type":"array","items":
# {"type":"integer","format":"int32"}} because byte is an alias for uint8.
#
# This script rewrites those fields to {"type":"string","format":"byte"}
# so that go-swagger generates the correct Go types.

SWAGGER_FILE="${1:?Usage: fix-swagger-spec.sh <swagger.json>}"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
JQ="${SCRIPT_DIR}/../bin/jq"

if [[ ! -x "$JQ" ]]; then
    echo "Error: jq not found at $JQ -- run 'make install-jq' first" >&2
    exit 1
fi

# Fix map[string][]byte fields (e.g. Secret.data, ConfigMap.binaryData)
# These appear as additionalProperties with array-of-int32; they should be
# additionalProperties with string format byte.
"$JQ" '
  # Fix Secret.data
  (if .definitions["v1.Secret"].properties.data.additionalProperties then
    .definitions["v1.Secret"].properties.data.additionalProperties = {"type": "string", "format": "byte"}
  else . end) |
  # Fix ConfigMap.binaryData
  (if .definitions["v1.ConfigMap"].properties.binaryData.additionalProperties then
    .definitions["v1.ConfigMap"].properties.binaryData.additionalProperties = {"type": "string", "format": "byte"}
  else . end) |
  # Fix AnalysisRun MetricProvider.plugin
  (if .definitions["github_com_akuity_kargo_api_stubs_rollouts_v1alpha1.MetricProvider"].properties.plugin.additionalProperties then
    .definitions["github_com_akuity_kargo_api_stubs_rollouts_v1alpha1.MetricProvider"].properties.plugin.additionalProperties = {"type": "string", "format": "byte"}
  else . end)
' "$SWAGGER_FILE" > "${SWAGGER_FILE}.tmp" && mv "${SWAGGER_FILE}.tmp" "$SWAGGER_FILE"
