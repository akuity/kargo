# Fix Phantom Empty Objects in Generated Go Client Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Stop the go-swagger-generated REST client (`pkg/client/generated/models`) from marshaling optional object-typed struct fields as phantom empty objects (e.g. `"status":{}` on a `Freight` that was never given a status).

**Architecture:** `swag` wraps any `$ref`'d property that carries a doc comment in `allOf: [{"$ref": X}]`, because raw `$ref` forbids sibling keywords like `description`. `go-swagger` renders `allOf`-wrapped properties as value-typed embedded structs no matter what (`x-nullable` has no effect on them), and Go's `encoding/json` never treats a non-pointer struct as "empty", so `omitempty` never omits it. A bare `$ref` property, by contrast, already generates as a pointer by default. The fix flattens `allOf: [{"$ref": X}], description: D` down to `{"$ref": X, description: D}` for properties that are *not* in their definition's `required` array, but only in a **separate, Go-client-only derivative of `swagger.json`** — the canonical committed `swagger.json` (which also feeds the UI's orval-generated TypeScript client) is left untouched, because orval follows strict OpenAPI semantics and drops sibling keywords next to `$ref`, which would silently strip field doc comments from ~83 generated TS model files if the canonical file were flattened directly.

**Tech Stack:** Bash + `jq` (spec post-processing), `go-swagger` (Go client codegen), Go + testify (regression test), `make`/`Makefile` (build wiring).

## Global Constraints

- Errors: stdlib `errors` package only; wrap with `fmt.Errorf` + `%w`.
- Go tests: testify, prefer `require`, table-driven with `t.Run()` subtests where more than one case exists.
- All commits require DCO sign-off (`git commit -s`).
- `pkg/client/generated/` is its own Go module (has its own `go.mod`/`go.sum`, replaced locally from the root `go.mod`). Regenerating it must never delete or modify `go.mod`/`go.sum` in that directory.
- Do not modify the canonical `swagger.json` post-processing (`hack/codegen/fix-swagger-spec.sh`) or its output — the UI's TypeScript client and docs depend on it staying exactly as today.
- Line length: soft limit 80, hard limit 120.

---

### Task 1: Add `hack/codegen/flatten-nullable-refs.sh`

**Files:**
- Create: `hack/codegen/flatten-nullable-refs.sh`

**Interfaces:**
- Produces: an executable script invoked as `hack/codegen/flatten-nullable-refs.sh <input-swagger.json> <output-path>`. Reads `<input-swagger.json>` (already processed by `fix-swagger-spec.sh`), writes a transformed copy to `<output-path>`, and leaves `<input-swagger.json>` untouched. Exits non-zero with a message on any broken `$ref` pointer in the output.
- Consumes: `hack/bin/jq` (built by `make install-jq`), same convention `hack/codegen/fix-swagger-spec.sh` already uses (`SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"; JQ="${SCRIPT_DIR}/../bin/jq"`).

- [ ] **Step 1: Ensure the jq tool is built**

Run: `make install-jq`
Expected: exits 0; `hack/bin/jq` exists and is executable (`ls -l hack/bin/jq` shows an executable file).

- [ ] **Step 2: Write the script**

Create `hack/codegen/flatten-nullable-refs.sh` with this exact content:

```bash
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
```

- [ ] **Step 3: Make it executable**

Run: `chmod +x hack/codegen/flatten-nullable-refs.sh`
Expected: exits 0.

- [ ] **Step 4: Smoke-test on a fixture — optional allOf ref gets flattened, required allOf ref is untouched**

Run:

```bash
cat > /tmp/flatten-fixture.json <<'EOF'
{
  "swagger": "2.0",
  "info": {"title": "test", "version": "1.0"},
  "paths": {},
  "definitions": {
    "Inner": {
      "type": "object",
      "properties": {"foo": {"type": "string"}}
    },
    "Outer": {
      "type": "object",
      "required": ["requiredField"],
      "properties": {
        "optionalField": {
          "description": "an optional field",
          "allOf": [{"$ref": "#/definitions/Inner"}]
        },
        "requiredField": {
          "description": "a required field",
          "allOf": [{"$ref": "#/definitions/Inner"}]
        },
        "plainField": {"type": "string"}
      }
    }
  }
}
EOF
hack/codegen/flatten-nullable-refs.sh /tmp/flatten-fixture.json /tmp/flatten-fixture-out.json
hack/bin/jq '.definitions.Outer.properties.optionalField' /tmp/flatten-fixture-out.json
hack/bin/jq '.definitions.Outer.properties.requiredField' /tmp/flatten-fixture-out.json
hack/bin/jq '.definitions.Outer.properties.plainField' /tmp/flatten-fixture-out.json
```

Expected output:

```
Wrote Go-client-only swagger spec (nullable optional refs flattened) to /tmp/flatten-fixture-out.json
{
  "$ref": "#/definitions/Inner",
  "description": "an optional field"
}
{
  "description": "a required field",
  "allOf": [
    {
      "$ref": "#/definitions/Inner"
    }
  ]
}
{
  "type": "string"
}
```

`optionalField` became a bare `$ref` (description preserved); `requiredField` kept its `allOf` wrapper unchanged; `plainField` was untouched.

- [ ] **Step 5: Smoke-test the broken-`$ref` guard**

Run:

```bash
cat > /tmp/flatten-fixture-broken.json <<'EOF'
{
  "swagger": "2.0",
  "info": {"title": "test", "version": "1.0"},
  "paths": {},
  "definitions": {
    "Outer": {
      "type": "object",
      "properties": {
        "optionalField": {
          "description": "dangling ref",
          "allOf": [{"$ref": "#/definitions/DoesNotExist"}]
        }
      }
    }
  }
}
EOF
hack/codegen/flatten-nullable-refs.sh /tmp/flatten-fixture-broken.json /tmp/flatten-fixture-broken-out.json; echo "exit: $?"
```

Expected: exits non-zero (`exit: 5` from `jq -e`'s `error(...)`), stderr shows `jq: error (at <stdin>:0): Broken $ref pointers found: DoesNotExist`, and `/tmp/flatten-fixture-broken-out.json` is **not** created (only the `.tmp` file may exist, since the final `mv` never runs).

- [ ] **Step 6: Clean up fixtures**

Run: `rm -f /tmp/flatten-fixture*.json`

- [ ] **Step 7: Commit**

```bash
git add hack/codegen/flatten-nullable-refs.sh
git commit -s -m "$(cat <<'EOF'
feat(codegen): add script to flatten optional allOf-wrapped refs for go client

go-swagger renders allOf-wrapped $ref properties as non-pointer embedded
structs regardless of x-nullable, so Go's omitempty never omits them,
producing phantom empty objects in the generated client. This script
produces a Go-client-only derivative of swagger.json with those wrappers
flattened for optional fields only, leaving the canonical swagger.json
(and the TypeScript client it feeds) untouched.
EOF
)"
```

---

### Task 2: Wire the new script into `codegen-openapi`

**Files:**
- Modify: `Makefile:260-283`

**Interfaces:**
- Consumes: `hack/codegen/flatten-nullable-refs.sh <input> <output>` from Task 1.
- Produces: the `codegen-openapi` target now generates the Go client from a temp Go-client-only spec instead of the canonical `swagger.json`; the `pnpm --dir=ui run generate:api` step is unaffected (still reads `swagger.json`).

- [ ] **Step 1: Edit the Makefile target**

Current (`Makefile:260-283`):

```makefile
.PHONY: codegen-openapi
codegen-openapi: install-jq
	rm -f swagger.json
	find pkg/client/generated -mindepth 1 ! -name go.mod ! -name go.sum -exec rm -rf {} +
	rm -rf /tmp/swagger-build
	mkdir -p /tmp/swagger-build
	go tool swag init \
		--generalInfo pkg/server/rest_router.go \
		--output /tmp/swagger-build \
		--parseDependency \
		--parseInternal \
		--outputTypes json
	mv /tmp/swagger-build/swagger.json .
	rm -rf /tmp/swagger-build
	hack/codegen/fix-swagger-spec.sh swagger.json
	mkdir -p pkg/client/generated
	go tool swagger generate client \
		-f swagger.json \
		-t pkg \
		--client-package client/generated \
		--model-package client/generated/models \
		--skip-validation
	pnpm --dir=ui install --dev
	pnpm --dir=ui run generate:api
```

Replace with:

```makefile
.PHONY: codegen-openapi
codegen-openapi: install-jq
	rm -f swagger.json
	find pkg/client/generated -mindepth 1 ! -name go.mod ! -name go.sum -exec rm -rf {} +
	rm -rf /tmp/swagger-build
	mkdir -p /tmp/swagger-build
	go tool swag init \
		--generalInfo pkg/server/rest_router.go \
		--output /tmp/swagger-build \
		--parseDependency \
		--parseInternal \
		--outputTypes json
	mv /tmp/swagger-build/swagger.json .
	rm -rf /tmp/swagger-build
	hack/codegen/fix-swagger-spec.sh swagger.json
	hack/codegen/flatten-nullable-refs.sh swagger.json /tmp/swagger-go-client.json
	mkdir -p pkg/client/generated
	go tool swagger generate client \
		-f /tmp/swagger-go-client.json \
		-t pkg \
		--client-package client/generated \
		--model-package client/generated/models \
		--skip-validation
	rm -f /tmp/swagger-go-client.json
	pnpm --dir=ui install --dev
	pnpm --dir=ui run generate:api
```

Only two lines change meaningfully: a new `hack/codegen/flatten-nullable-refs.sh` invocation is inserted before the client generation, the `-f` flag now points at its output, and a cleanup `rm -f` line is added after.

- [ ] **Step 2: Verify the Makefile still parses**

Run: `make -n codegen-openapi`
Expected: prints the recipe's commands (dry run) with no `make: *** No rule to make target` or syntax errors, and the printed `go tool swagger generate client` line shows `-f /tmp/swagger-go-client.json`.

- [ ] **Step 3: Commit**

```bash
git add Makefile
git commit -s -m "$(cat <<'EOF'
build(codegen): generate go client from flattened swagger spec

Wires hack/codegen/flatten-nullable-refs.sh into codegen-openapi so the
go-swagger client is generated from a derivative spec with optional
allOf-wrapped refs flattened, fixing phantom empty objects on optional
struct fields. The canonical swagger.json used by the TypeScript client
is untouched.
EOF
)"
```

---

### Task 3: Add a failing regression test that demonstrates the bug

**Files:**
- Create: `pkg/cli/client/generated_models_test.go`

**Interfaces:**
- Consumes: `github.com/akuity/kargo/pkg/client/generated/models.Freight` (existing generated type; fields used: `Alias string`, `Status *FreightStatus` — after Task 4's regen; today it is still `Status struct { FreightStatus }`, so this test is expected to fail to compile/pass until Task 4 runs).
- Produces: `TestFreightMarshal_NoPhantomEmptyObjects` — a regression test other engineers can extend for other models.

- [ ] **Step 1: Write the failing test**

Create `pkg/cli/client/generated_models_test.go`:

```go
package client

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/akuity/kargo/pkg/client/generated/models"
)

func TestFreightMarshal_NoPhantomEmptyObjects(t *testing.T) {
	testCases := []struct {
		name   string
		assert func(*testing.T, []byte)
	}{
		{
			name: "unset optional status is omitted, not an empty object",
			assert: func(t *testing.T, b []byte) {
				require.NotContains(t, string(b), `"status":{}`)
			},
		},
		{
			name: "unset optional metadata is omitted, not an empty object",
			assert: func(t *testing.T, b []byte) {
				require.NotContains(t, string(b), `"metadata":{}`)
			},
		},
	}

	freight := models.Freight{Alias: "my-freight"}
	b, err := json.Marshal(freight)
	require.NoError(t, err)

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assert(t, b)
		})
	}
}
```

- [ ] **Step 2: Run it to confirm it currently fails**

Run: `go test ./pkg/cli/client/... -run TestFreightMarshal_NoPhantomEmptyObjects -v`
Expected: **FAIL** — output includes `"status":{}` in the marshaled JSON (visible in the `require.NotContains` failure message), because `pkg/client/generated/models.Freight.Status` is still today's non-pointer embedded struct. This confirms the test actually exercises the bug.

- [ ] **Step 3: Commit**

```bash
git add pkg/cli/client/generated_models_test.go
git commit -s -m "$(cat <<'EOF'
test(cli): add failing regression test for phantom empty objects

Demonstrates that marshaling a Freight with no status/metadata set today
produces phantom "status":{} / "metadata":{} in the generated client.
Task 4 regenerates the client and makes this pass.
EOF
)"
```

---

### Task 4: Regenerate the real Go client and confirm the fix

**Files:**
- Modify (generated, not hand-edited): `pkg/client/generated/**` (except `go.mod`/`go.sum`, which must not change)
- No changes expected: `ui/src/gen/**`, `pkg/client/generated/go.mod`, `pkg/client/generated/go.sum`

**Interfaces:**
- Consumes: Task 1's script (already wired into `codegen-openapi` by Task 2).
- Produces: a regenerated `pkg/client/generated` where `models.Freight.Status` is `*FreightStatus` and `models.Freight.Metadata` is `*V1ObjectMeta`, making Task 3's test pass.

- [ ] **Step 1: Snapshot current state for comparison**

Run: `git status --short` (expected: clean, aside from Tasks 1-3's already-committed changes) and note the current file `git rev-parse HEAD` in case you need to compare.

- [ ] **Step 2: Regenerate**

Run: `make codegen-openapi`
Expected: exits 0. Near the end of the output you should see `Wrote Go-client-only swagger spec (nullable optional refs flattened) to /tmp/swagger-go-client.json`, followed by go-swagger's `Generation completed!`, then pnpm installing and running `generate:api`.

- [ ] **Step 3: Confirm the generated submodule's `go.mod`/`go.sum` are untouched**

Run: `git status --short pkg/client/generated/go.mod pkg/client/generated/go.sum`
Expected: empty output (no changes). If either file shows as modified or deleted, stop and investigate — `codegen-openapi`'s `find ... -exec rm -rf {} +` step is supposed to exclude them (`! -name go.mod ! -name go.sum`); a diff here means the exclusion broke.

- [ ] **Step 4: Confirm the TypeScript client is unaffected**

Run: `git status --short ui/src/gen`
Expected: empty output. If any TS files under `ui/src/gen` show as changed, first rule out environment noise before treating it as a regression: stash the change (`git stash -- ui/src/gen`), re-run `pnpm --dir=ui run generate:api` against the **unmodified `swagger.json` from `git show HEAD:swagger.json`** in a scratch copy, and diff that output against the currently committed `ui/src/gen`. If that also differs (e.g. due to a locally installed `orval`/`pnpm` version that doesn't match whatever generated the committed files), the diff is pre-existing baseline noise unrelated to this fix — restore the stash and don't fold it into this change's commits. If the unmodified-spec regen matches committed `ui/src/gen` exactly but the fixed-spec regen doesn't, that's a real regression from this change — stop and investigate before proceeding.

- [ ] **Step 5: Confirm `swagger.json` itself is unchanged**

Run: `git status --short swagger.json`
Expected: empty output (Task 1/2's script only ever writes to a `/tmp` file, never back to the committed `swagger.json`).

- [ ] **Step 6: Spot-check the fixed fields**

Run: `grep -n "Status \|Metadata " pkg/client/generated/models/freight.go`
Expected:
```
	Metadata *V1ObjectMeta `json:"metadata,omitempty"`
	Status *FreightStatus `json:"status,omitempty"`
```
(both pointers now, `omitempty` present).

Run: `grep -n "Origin struct" pkg/client/generated/models/freight.go`
Expected: still present — `Origin struct {` — confirming the required field is untouched.

- [ ] **Step 7: Run the regression test — it should now pass**

Run: `go test ./pkg/cli/client/... -run TestFreightMarshal_NoPhantomEmptyObjects -v`
Expected: **PASS** for both subtests.

- [ ] **Step 8: Build everything that depends on the generated client**

Run: `go build ./pkg/... ./cmd/...`
Expected: exits 0, no compile errors.

- [ ] **Step 9: Commit the regenerated client**

```bash
git add pkg/client/generated
git commit -s -m "$(cat <<'EOF'
chore(codegen): regenerate go client with flattened optional refs

Regenerates pkg/client/generated from the Go-client-only spec produced
by hack/codegen/flatten-nullable-refs.sh. Optional object-typed fields
(e.g. Freight.Status, Freight.Metadata) are now pointers and correctly
omitted when unset instead of marshaling as phantom empty objects.
EOF
)"
```

---

### Task 5: Full verification and cleanup

**Files:** none (verification only)

- [ ] **Step 1: Run the full unit test suite**

Run: `make test-unit`
Expected: exits 0, all packages pass (this also re-runs Task 3/4's regression test as part of `./...`).

- [ ] **Step 2: Run Go linting**

Run: `make lint-go`
Expected: exits 0, no lint errors (covers the new `pkg/cli/client/generated_models_test.go`; `pkg/client/generated/**` is excluded from linting like other generated code, and `hack/codegen/*.sh` is shell, not linted by `lint-go`).

- [ ] **Step 3: Confirm the build is clean end-to-end**

Run: `go build ./... `
Expected: exits 0.

- [ ] **Step 4: Review the final diff for scope**

Run: `git log --oneline main..HEAD` and `git diff --stat main..HEAD`
Expected: only these paths changed: `hack/codegen/flatten-nullable-refs.sh` (new), `Makefile`, `pkg/cli/client/generated_models_test.go` (new), `pkg/client/generated/**` (excluding `go.mod`/`go.sum`). No changes to `swagger.json`, `ui/src/gen/**`, `pkg/client/generated/go.mod`, or `pkg/client/generated/go.sum`. If `make codegen-openapi` picked up unrelated `swag`-annotation drift in `swagger.json` from other recent commits, that would show up as an unstaged `swagger.json` diff at this point (Task 4 Step 5 already checked it was empty) — if it's not empty, treat it as a separate, pre-existing issue and do not fold it into this fix's commits.

- [ ] **Step 5: Push and open a PR (only if requested)**

Do not push or open a PR unless the user explicitly asks — follow the repo's standard PR flow (see `AGENTS.md`) once they do.
