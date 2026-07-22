package dispatch

import (
	"context"
	"testing"
	"time"

	"github.com/open-policy-agent/opa/v1/rego"
	"github.com/stretchr/testify/require"
)

// testNow is a Wednesday, 15:00 UTC (07:00 in America/Los_Angeles).
const testNow = "2026-07-15T15:00:00Z"

// hotfixBypassRules is the canonical custom-policy pattern, typically a
// cluster (operator) policy: hotfixes bypass every freeze. Hotfix
// semantics are defined in the custom policy itself; the stdlib supplies
// only the semver building block. Rules only -- the engine prepends the
// package and imports.
const hotfixBypassRules = `freeze_bypass(f) if is_hotfix

is_hotfix if {
	count(shared_images) > 0
	every pair in shared_images {
		kargo.is_semver_patch(pair.old, pair.new)
	}
}

shared_images := [pair |
	some img in input.freight.images
	some last in input.stage.lastPromotion.freight.images
	img.repoURL == last.repoURL
	pair := {"old": last.tag, "new": img.tag}
]
`

func emptyData() map[string]any {
	return map[string]any{
		"windows":            []any{},
		"freezes":            []any{},
		"scopes":             defaultScopes,
		"rateLimit":          map[string]any{},
		"queue":              []any{},
		"currentFreight":     map[string]any{},
		"autoPromotionHolds": map[string]any{},
	}
}

func testInput(class string) map[string]any {
	return map[string]any{
		"promotion": map[string]any{
			"name":        "test-promo",
			"class":       class,
			"createdAt":   testNow,
			"actor":       "",
			"labels":      map[string]any{},
			"annotations": map[string]any{},
		},
		"freight": map[string]any{
			"name": "test-freight",
			"images": []any{
				map[string]any{"repoURL": "example/nginx", "tag": "1.2.3", "digest": ""},
			},
		},
		"stage": map[string]any{
			"name":          "prod",
			"project":       "test-project",
			"labels":        map[string]any{},
			"annotations":   map[string]any{},
			"lastPromotion": map[string]any{},
		},
		"project":      map[string]any{"labels": map[string]any{}, "annotations": map[string]any{}},
		"applications": []any{},
		"now":          testNow,
	}
}

// testOrigin is the Warehouse origin used by the Freight-ordering cases.
const testOrigin = "Warehouse/demo/nginx"

// orderingInput is a manual-forward candidate whose Freight carries an origin
// and a discovery time, for exercising the kargo.lib Freight-ordering helpers
// (advances / regresses) against data.currentFreight.
func orderingInput(freightName, discoveredAt string) map[string]any {
	in := testInput(ClassManualForward)
	in["freight"] = map[string]any{
		"name":         freightName,
		"origin":       testOrigin,
		"discoveredAt": discoveredAt,
	}
	return in
}

// The current Freight the ordering cases compare against: a candidate reusing
// this name is a re-promote, an earlier discoveredAt regresses, a later one
// advances.
const (
	testCurrentFreightName         = "freight-cur"
	testCurrentFreightDiscoveredAt = "2026-07-15T12:00:00Z"
)

// orderingData sets data.currentFreight for the candidate's origin.
func orderingData() map[string]any {
	data := emptyData()
	data["currentFreight"] = map[string]any{
		testOrigin: map[string]any{
			"name":         testCurrentFreightName,
			"discoveredAt": testCurrentFreightDiscoveredAt,
		},
	}
	return data
}

// autoOrderingInput is the auto-forward counterpart of orderingInput, for the
// built-in auto-forward guards (regression / auto-hold).
func autoOrderingInput(freightName, discoveredAt string) map[string]any {
	in := testInput(ClassAutoForward)
	in["freight"] = map[string]any{
		"name":         freightName,
		"origin":       testOrigin,
		"discoveredAt": discoveredAt,
	}
	return in
}

// scheduledInput carries a promote-after annotation, for the built-in
// scheduled rule.
func scheduledInput(promoteAfter string) map[string]any {
	in := testInput(ClassManualForward)
	promo, _ := in["promotion"].(map[string]any)
	promo["annotations"] = map[string]any{
		"kargo.akuity.io/promote-after": promoteAfter,
	}
	return in
}

// heldData sets data.autoPromotionHolds for the candidate's origin, on top of
// a data document that also has current Freight (so a held auto-forward can be
// one that advances -- the create-race the auto-hold rule exists to catch).
func heldData() map[string]any {
	data := orderingData()
	data["autoPromotionHolds"] = map[string]any{
		testOrigin: map[string]any{
			"freightName":   testCurrentFreightName,
			"promotionName": "prod.01",
			"actor":         "alice",
			"createdAt":     testCurrentFreightDiscoveredAt,
		},
	}
	return data
}

func TestEngineEvaluate(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name          string
		projectCustom string
		clusterCustom string
		input         map[string]any
		data          func() map[string]any
		assert        func(*testing.T, *Decision, error)
	}{
		{
			name:  "default policy allows when no config matches",
			input: testInput(ClassAutoForward),
			data:  emptyData,
			assert: func(t *testing.T, d *Decision, err error) {
				require.NoError(t, err)
				require.True(t, d.Allow)
				require.Equal(t, "within policy", d.Message)
				require.Zero(t, d.RequeueAfter)
			},
		},
		{
			name:  "active freeze denies with requeue until its end",
			input: testInput(ClassAutoForward),
			data: func() map[string]any {
				data := emptyData()
				data["freezes"] = []any{map[string]any{
					"name":          "holiday",
					"start":         "2026-07-15T00:00:00Z",
					"end":           "2026-07-16T00:00:00Z",
					"scope":         "no-promotions",
					"argocdServers": []any{},
				}}
				return data
			},
			assert: func(t *testing.T, d *Decision, err error) {
				require.NoError(t, err)
				require.False(t, d.Allow)
				require.Contains(t, d.Message, `frozen by freeze "holiday"`)
				// 9 hours from testNow to the end of the freeze.
				require.Equal(t, 9*time.Hour, d.RequeueAfter)
			},
		},
		{
			name:  "expired freeze does not deny",
			input: testInput(ClassAutoForward),
			data: func() map[string]any {
				data := emptyData()
				data["freezes"] = []any{map[string]any{
					"name":          "past",
					"start":         "2026-07-01T00:00:00Z",
					"end":           "2026-07-02T00:00:00Z",
					"scope":         "no-promotions",
					"argocdServers": []any{},
				}}
				return data
			},
			assert: func(t *testing.T, d *Decision, err error) {
				require.NoError(t, err)
				require.True(t, d.Allow)
			},
		},
		{
			name:  "no-forward freeze permits rollbacks",
			input: testInput(ClassRollback),
			data: func() map[string]any {
				data := emptyData()
				data["freezes"] = []any{map[string]any{
					"name":          "freeze",
					"start":         "2026-07-15T00:00:00Z",
					"end":           "2026-07-16T00:00:00Z",
					"scope":         "no-forward",
					"argocdServers": []any{},
				}}
				return data
			},
			assert: func(t *testing.T, d *Decision, err error) {
				require.NoError(t, err)
				require.True(t, d.Allow)
			},
		},
		{
			name:  "no-auto freeze permits manual promotions",
			input: testInput(ClassManualForward),
			data: func() map[string]any {
				data := emptyData()
				data["freezes"] = []any{map[string]any{
					"name":          "auto-off",
					"start":         "2026-07-15T00:00:00Z",
					"end":           "2026-07-16T00:00:00Z",
					"scope":         "no-auto",
					"argocdServers": []any{},
				}}
				return data
			},
			assert: func(t *testing.T, d *Decision, err error) {
				require.NoError(t, err)
				require.True(t, d.Allow)
			},
		},
		{
			name: "server-scoped freeze applies only to matching destinations",
			input: func() map[string]any {
				input := testInput(ClassAutoForward)
				input["applications"] = []any{map[string]any{
					"name":      "app",
					"namespace": "argocd",
					"destination": map[string]any{
						"server": "https://other.example.com",
						"name":   "other",
					},
				}}
				return input
			}(),
			data: func() map[string]any {
				data := emptyData()
				data["freezes"] = []any{map[string]any{
					"name":          "server-freeze",
					"start":         "2026-07-15T00:00:00Z",
					"end":           "2026-07-16T00:00:00Z",
					"scope":         "no-promotions",
					"argocdServers": []any{"https://prod.example.com"},
				}}
				return data
			},
			assert: func(t *testing.T, d *Decision, err error) {
				require.NoError(t, err)
				require.True(t, d.Allow)
			},
		},
		{
			name: "server-scoped freeze denies matching destinations",
			input: func() map[string]any {
				input := testInput(ClassAutoForward)
				input["applications"] = []any{map[string]any{
					"name":      "app",
					"namespace": "argocd",
					"destination": map[string]any{
						"server": "https://prod.example.com",
						"name":   "prod",
					},
				}}
				return input
			}(),
			data: func() map[string]any {
				data := emptyData()
				data["freezes"] = []any{map[string]any{
					"name":          "server-freeze",
					"start":         "2026-07-15T00:00:00Z",
					"end":           "2026-07-16T00:00:00Z",
					"scope":         "no-promotions",
					"argocdServers": []any{"https://prod.example.com"},
				}}
				return data
			},
			assert: func(t *testing.T, d *Decision, err error) {
				require.NoError(t, err)
				require.False(t, d.Allow)
			},
		},
		{
			name:  "outside promotion window denies forward with requeue to next open",
			input: testInput(ClassAutoForward),
			data: func() map[string]any {
				data := emptyData()
				// Weekdays 18:00-23:00 UTC; testNow is 15:00 UTC Wednesday.
				data["windows"] = []any{map[string]any{
					"name":       "evenings",
					"recurrence": "FREQ=WEEKLY;BYDAY=MO,TU,WE,TH,FR",
					"start":      "18:00",
					"end":        "23:00",
					"location":   "UTC",
				}}
				return data
			},
			assert: func(t *testing.T, d *Decision, err error) {
				require.NoError(t, err)
				require.False(t, d.Allow)
				require.Contains(t, d.Message, "outside all promotion windows")
				// Next open is 18:00 today, 3 hours from testNow.
				require.Equal(t, 3*time.Hour, d.RequeueAfter)
			},
		},
		{
			name:  "inside promotion window allows forward",
			input: testInput(ClassAutoForward),
			data: func() map[string]any {
				data := emptyData()
				// Weekdays 09:00-17:00 UTC; testNow is 15:00 UTC Wednesday.
				data["windows"] = []any{map[string]any{
					"name":       "business-hours",
					"recurrence": "FREQ=WEEKLY;BYDAY=MO,TU,WE,TH,FR",
					"start":      "09:00",
					"end":        "17:00",
					"location":   "UTC",
				}}
				return data
			},
			assert: func(t *testing.T, d *Decision, err error) {
				require.NoError(t, err)
				require.True(t, d.Allow)
			},
		},
		{
			name:  "promotion window does not hold rollbacks",
			input: testInput(ClassRollback),
			data: func() map[string]any {
				data := emptyData()
				data["windows"] = []any{map[string]any{
					"name":       "evenings",
					"recurrence": "FREQ=WEEKLY;BYDAY=MO,TU,WE,TH,FR",
					"start":      "18:00",
					"end":        "23:00",
					"location":   "UTC",
				}}
				return data
			},
			assert: func(t *testing.T, d *Decision, err error) {
				require.NoError(t, err)
				require.True(t, d.Allow)
			},
		},
		{
			name:  "rate limit under max allows",
			input: testInput(ClassAutoForward),
			data: func() map[string]any {
				data := emptyData()
				data["rateLimit"] = map[string]any{"prod": map[string]any{
					"max":    int64(2),
					"window": int64(30 * time.Minute),
					"dispatches": []any{
						mustParseNS("2026-07-15T14:50:00Z"),
					},
				}}
				return data
			},
			assert: func(t *testing.T, d *Decision, err error) {
				require.NoError(t, err)
				require.True(t, d.Allow)
			},
		},
		{
			name:  "rate limit at max denies until the oldest dispatch ages out",
			input: testInput(ClassAutoForward),
			data: func() map[string]any {
				data := emptyData()
				data["rateLimit"] = map[string]any{"prod": map[string]any{
					"max":    int64(2),
					"window": int64(30 * time.Minute),
					"dispatches": []any{
						mustParseNS("2026-07-15T14:40:00Z"),
						mustParseNS("2026-07-15T14:50:00Z"),
					},
				}}
				return data
			},
			assert: func(t *testing.T, d *Decision, err error) {
				require.NoError(t, err)
				require.False(t, d.Allow)
				require.Contains(t, d.Message, "rate limit")
				// Oldest in-window dispatch (14:40) + 30m window = 15:10,
				// which is 10 minutes from testNow.
				require.Equal(t, 10*time.Minute, d.RequeueAfter)
			},
		},
		{
			name:  "rate limit ignores dispatches outside the rolling window",
			input: testInput(ClassAutoForward),
			data: func() map[string]any {
				data := emptyData()
				data["rateLimit"] = map[string]any{"prod": map[string]any{
					"max":    int64(2),
					"window": int64(30 * time.Minute),
					"dispatches": []any{
						mustParseNS("2026-07-15T13:00:00Z"),
						mustParseNS("2026-07-15T14:50:00Z"),
					},
				}}
				return data
			},
			assert: func(t *testing.T, d *Decision, err error) {
				require.NoError(t, err)
				require.True(t, d.Allow)
			},
		},
		{
			name:  "rate limit does not hold manual promotions",
			input: testInput(ClassManualForward),
			data: func() map[string]any {
				data := emptyData()
				data["rateLimit"] = map[string]any{"prod": map[string]any{
					"max":    int64(1),
					"window": int64(30 * time.Minute),
					"dispatches": []any{
						mustParseNS("2026-07-15T14:50:00Z"),
					},
				}}
				return data
			},
			assert: func(t *testing.T, d *Decision, err error) {
				require.NoError(t, err)
				require.True(t, d.Allow)
			},
		},
		{
			name:  "multiple violations join messages and take the soonest requeue",
			input: testInput(ClassAutoForward),
			data: func() map[string]any {
				data := emptyData()
				data["freezes"] = []any{map[string]any{
					"name":          "holiday",
					"start":         "2026-07-15T00:00:00Z",
					"end":           "2026-07-16T00:00:00Z",
					"scope":         "no-promotions",
					"argocdServers": []any{},
				}}
				data["windows"] = []any{map[string]any{
					"name":       "evenings",
					"recurrence": "FREQ=WEEKLY;BYDAY=MO,TU,WE,TH,FR",
					"start":      "18:00",
					"end":        "23:00",
					"location":   "UTC",
				}}
				return data
			},
			assert: func(t *testing.T, d *Decision, err error) {
				require.NoError(t, err)
				require.False(t, d.Allow)
				require.Contains(t, d.Message, "frozen by freeze")
				require.Contains(t, d.Message, "outside all promotion windows")
				// Window opens at 18:00 (3h), freeze ends at 00:00 (9h);
				// the soonest boundary wins.
				require.Equal(t, 3*time.Hour, d.RequeueAfter)
			},
		},
		{
			name: "project violation composes into the default decision",
			projectCustom: `violation contains {"rule": "no", "msg": "computer says no", "requeue": 60} if {
	input.promotion.class == "auto-forward"
}
`,
			input: testInput(ClassAutoForward),
			data:  emptyData,
			assert: func(t *testing.T, d *Decision, err error) {
				require.NoError(t, err)
				require.False(t, d.Allow)
				require.Equal(t, "computer says no", d.Message)
				require.Equal(t, time.Minute, d.RequeueAfter)
			},
		},
		{
			name: "cluster violation composes into the default decision",
			clusterCustom: `violation contains {"rule": "ops", "msg": "cluster says no"} if {
	input.promotion.class == "auto-forward"
}
`,
			input: testInput(ClassAutoForward),
			data:  emptyData,
			assert: func(t *testing.T, d *Decision, err error) {
				require.NoError(t, err)
				require.False(t, d.Allow)
				require.Equal(t, "cluster says no", d.Message)
			},
		},
		{
			name: "project and cluster violations both contribute",
			projectCustom: `violation contains {"rule": "p", "msg": "project says no"} if {
	input.promotion.class == "auto-forward"
}
`,
			clusterCustom: `violation contains {"rule": "c", "msg": "cluster says no"} if {
	input.promotion.class == "auto-forward"
}
`,
			input: testInput(ClassAutoForward),
			data:  emptyData,
			assert: func(t *testing.T, d *Decision, err error) {
				require.NoError(t, err)
				require.False(t, d.Allow)
				require.Contains(t, d.Message, "project says no")
				require.Contains(t, d.Message, "cluster says no")
			},
		},
		{
			// A custom policy reads data.queue to yield when a rollback is
			// waiting behind the candidate under evaluation. Exercises that
			// the queue is threaded through and consumable. The built-in
			// yield-to-rollback rule now co-fires (same outcome), so the
			// message carries both -- assert on the custom substring.
			name: "custom policy yields to a queued rollback via data.queue",
			projectCustom: `violation contains {"rule": "yield", "msg": "yielding to queued rollback"} if {
	input.promotion.class == "auto-forward"
	some q in data.queue
	q.name != input.promotion.name
	q.class == "rollback"
}
`,
			input: testInput(ClassAutoForward),
			data: func() map[string]any {
				data := emptyData()
				data["queue"] = []any{
					map[string]any{"name": "test-promo", "class": "auto-forward", "createdAt": testNow},
					map[string]any{"name": "rb.01", "class": "rollback", "createdAt": testNow},
				}
				return data
			},
			assert: func(t *testing.T, d *Decision, err error) {
				require.NoError(t, err)
				require.False(t, d.Allow)
				require.Contains(t, d.Message, "yielding to queued rollback")
			},
		},
		{
			// The same policy allows when the queue holds no rollback,
			// confirming it is genuinely reading the queue contents.
			name: "custom policy allows when data.queue holds no rollback",
			projectCustom: `violation contains {"rule": "yield", "msg": "yielding to queued rollback"} if {
	input.promotion.class == "auto-forward"
	some q in data.queue
	q.name != input.promotion.name
	q.class == "rollback"
}
`,
			input: testInput(ClassAutoForward),
			data: func() map[string]any {
				data := emptyData()
				data["queue"] = []any{
					map[string]any{"name": "test-promo", "class": "auto-forward", "createdAt": testNow},
					map[string]any{"name": "fwd.01", "class": "auto-forward", "createdAt": testNow},
				}
				return data
			},
			assert: func(t *testing.T, d *Decision, err error) {
				require.NoError(t, err)
				require.True(t, d.Allow)
			},
		},
		{
			// A custom policy uses kargo.regresses (Freight clock, per origin)
			// to hold a candidate older than the origin's current Freight. The
			// built-in would-regress rule now co-fires for this manual-forward
			// (same outcome), so the message carries both -- assert on the
			// custom substring.
			name: "custom policy denies a regressing candidate via kargo.regresses",
			projectCustom: `violation contains {"rule": "regress", "msg": "would regress the stage"} if {
	kargo.regresses
}
`,
			input: orderingInput("freight-old", "2026-07-15T09:00:00Z"),
			data: func() map[string]any {
				return orderingData()
			},
			assert: func(t *testing.T, d *Decision, err error) {
				require.NoError(t, err)
				require.False(t, d.Allow)
				require.Contains(t, d.Message, "would regress the stage")
			},
		},
		{
			// The same policy allows a candidate newer than current: it
			// advances the Stage, so kargo.regresses is false.
			name: "custom policy allows an advancing candidate",
			projectCustom: `violation contains {"rule": "regress", "msg": "would regress the stage"} if {
	kargo.regresses
}
`,
			input: orderingInput("freight-new", "2026-07-15T14:00:00Z"),
			data: func() map[string]any {
				return orderingData()
			},
			assert: func(t *testing.T, d *Decision, err error) {
				require.NoError(t, err)
				require.True(t, d.Allow)
			},
		},
		{
			// A re-promote of the current Freight (equal discovery time, equal
			// name) is neither advance nor regress: freight_newer is strict.
			name: "custom policy allows a re-promote of the current Freight",
			projectCustom: `violation contains {"rule": "regress", "msg": "would regress the stage"} if {
	kargo.regresses
}
`,
			input: orderingInput("freight-cur", "2026-07-15T12:00:00Z"),
			data: func() map[string]any {
				return orderingData()
			},
			assert: func(t *testing.T, d *Decision, err error) {
				require.NoError(t, err)
				require.True(t, d.Allow)
			},
		},
		{
			// On a fresh origin (no current Freight) kargo.regresses is
			// undefined, so the candidate is allowed -- nothing to regress past.
			name: "custom policy allows on a fresh origin",
			projectCustom: `violation contains {"rule": "regress", "msg": "would regress the stage"} if {
	kargo.regresses
}
`,
			input: orderingInput("freight-1", "2026-07-15T09:00:00Z"),
			data:  emptyData,
			assert: func(t *testing.T, d *Decision, err error) {
				require.NoError(t, err)
				require.True(t, d.Allow)
			},
		},
		{
			// kargo.advances is the dual: a custom policy can require forward
			// motion. Here an equal-name/older candidate does not advance. The
			// built-in would-regress rule now co-fires for this manual-forward
			// (same outcome), so the message carries both -- assert on the
			// custom substring.
			name: "custom policy requires kargo.advances",
			projectCustom: `violation contains {"rule": "stale", "msg": "does not advance the stage"} if {
	kargo.current_freight
	not kargo.advances
}
`,
			input: orderingInput("freight-old", "2026-07-15T09:00:00Z"),
			data: func() map[string]any {
				return orderingData()
			},
			assert: func(t *testing.T, d *Decision, err error) {
				require.NoError(t, err)
				require.False(t, d.Allow)
				require.Contains(t, d.Message, "does not advance the stage")
			},
		},
		{
			// AX5: a forward candidate yields to a queued rollback so recovery
			// preempts change. The built-in fires with no custom policy.
			name:  "built-in yield-to-rollback holds an auto-forward behind a queued rollback",
			input: testInput(ClassAutoForward),
			data: func() map[string]any {
				data := emptyData()
				data["queue"] = []any{
					map[string]any{"name": "test-promo", "class": "auto-forward", "createdAt": testNow},
					map[string]any{"name": "rb.01", "class": "rollback", "createdAt": testNow},
				}
				return data
			},
			assert: func(t *testing.T, d *Decision, err error) {
				require.NoError(t, err)
				require.False(t, d.Allow)
				require.Equal(t, `yielding to queued rollback "rb.01"`, d.Message)
				require.Equal(t, 5*time.Second, d.RequeueAfter)
			},
		},
		{
			// A manual-forward also yields to a queued rollback.
			name:  "built-in yield-to-rollback holds a manual-forward behind a queued rollback",
			input: testInput(ClassManualForward),
			data: func() map[string]any {
				data := emptyData()
				data["queue"] = []any{
					map[string]any{"name": "test-promo", "class": "manual-forward", "createdAt": testNow},
					map[string]any{"name": "rb.01", "class": "rollback", "createdAt": testNow},
				}
				return data
			},
			assert: func(t *testing.T, d *Decision, err error) {
				require.NoError(t, err)
				require.False(t, d.Allow)
				require.Contains(t, d.Message, `yielding to queued rollback "rb.01"`)
			},
		},
		{
			// The rollback itself is never yielded -- it is not a forward
			// class, so no yield rule applies even with peers queued.
			name:  "built-in yield-to-rollback allows the rollback candidate",
			input: testInput(ClassRollback),
			data: func() map[string]any {
				data := emptyData()
				data["queue"] = []any{
					map[string]any{"name": "test-promo", "class": "rollback", "createdAt": testNow},
					map[string]any{"name": "rb.02", "class": "rollback", "createdAt": testNow},
				}
				return data
			},
			assert: func(t *testing.T, d *Decision, err error) {
				require.NoError(t, err)
				require.True(t, d.Allow)
			},
		},
		{
			// AX5: automation yields to an explicit human decision -- an
			// auto-forward defers while any manual-forward is queued.
			name:  "built-in yield-to-manual holds an auto-forward behind a queued manual",
			input: testInput(ClassAutoForward),
			data: func() map[string]any {
				data := emptyData()
				data["queue"] = []any{
					map[string]any{"name": "test-promo", "class": "auto-forward", "createdAt": testNow},
					map[string]any{"name": "m.01", "class": "manual-forward", "createdAt": testNow},
				}
				return data
			},
			assert: func(t *testing.T, d *Decision, err error) {
				require.NoError(t, err)
				require.False(t, d.Allow)
				require.Equal(t, `yielding to queued manual promotion "m.01"`, d.Message)
				require.Equal(t, 5*time.Second, d.RequeueAfter)
			},
		},
		{
			// Manual-forwards are never yielded to one another -- peers run
			// FIFO, so the gate does not hold a manual behind a manual.
			name:  "built-in yield-to-manual does not hold a manual behind a manual",
			input: testInput(ClassManualForward),
			data: func() map[string]any {
				data := emptyData()
				data["queue"] = []any{
					map[string]any{"name": "test-promo", "class": "manual-forward", "createdAt": testNow},
					map[string]any{"name": "m.02", "class": "manual-forward", "createdAt": testNow},
				}
				return data
			},
			assert: func(t *testing.T, d *Decision, err error) {
				require.NoError(t, err)
				require.True(t, d.Allow)
			},
		},
		{
			// AX6: an auto-forward that would not advance the Stage is stale.
			name:  "built-in regression holds a non-advancing auto-forward",
			input: autoOrderingInput("freight-old", "2026-07-15T09:00:00Z"),
			data:  orderingData,
			assert: func(t *testing.T, d *Decision, err error) {
				require.NoError(t, err)
				require.False(t, d.Allow)
				require.Equal(t, "auto-promotion would not advance the stage; stale", d.Message)
			},
		},
		{
			name:  "built-in regression allows an advancing auto-forward",
			input: autoOrderingInput("freight-new", "2026-07-15T14:00:00Z"),
			data:  orderingData,
			assert: func(t *testing.T, d *Decision, err error) {
				require.NoError(t, err)
				require.True(t, d.Allow)
			},
		},
		{
			// A fresh origin has no current Freight, so nothing is stale.
			name:  "built-in regression allows an auto-forward on a fresh origin",
			input: autoOrderingInput("freight-1", "2026-07-15T09:00:00Z"),
			data:  emptyData,
			assert: func(t *testing.T, d *Decision, err error) {
				require.NoError(t, err)
				require.True(t, d.Allow)
			},
		},
		{
			// AX6: a manual-forward strictly older than current is held for an
			// operator decision (re-issue as a rollback if intended).
			name:  "built-in would-regress holds a regressing manual-forward",
			input: orderingInput("freight-old", "2026-07-15T09:00:00Z"),
			data:  orderingData,
			assert: func(t *testing.T, d *Decision, err error) {
				require.NoError(t, err)
				require.False(t, d.Allow)
				require.Equal(
					t,
					`held: would regress the stage below current Freight "freight-cur"; re-issue as a rollback if intended`,
					d.Message,
				)
				require.Zero(t, d.RequeueAfter)
			},
		},
		{
			// A re-promote of the current Freight (equal, not older) is not
			// held: kargo.regresses is strict.
			name:  "built-in would-regress allows a re-promote of current",
			input: orderingInput("freight-cur", "2026-07-15T12:00:00Z"),
			data:  orderingData,
			assert: func(t *testing.T, d *Decision, err error) {
				require.NoError(t, err)
				require.True(t, d.Allow)
			},
		},
		{
			// The create-race (AX8/#3016): an auto-forward for a held origin is
			// denied even though it ADVANCES current (the anti-regression guard
			// cannot catch this -- only the hold can).
			name:  "built-in auto-hold denies an advancing auto-forward for a held origin",
			input: autoOrderingInput("freight-new", "2026-07-15T14:00:00Z"),
			data:  heldData,
			assert: func(t *testing.T, d *Decision, err error) {
				require.NoError(t, err)
				require.False(t, d.Allow)
				require.Equal(
					t,
					`auto-promotion held for origin "Warehouse/demo/nginx"; awaiting resume`,
					d.Message,
				)
				require.Zero(t, d.RequeueAfter)
			},
		},
		{
			name:  "built-in auto-hold allows an auto-forward for an unheld origin",
			input: autoOrderingInput("freight-new", "2026-07-15T14:00:00Z"),
			data:  orderingData,
			assert: func(t *testing.T, d *Decision, err error) {
				require.NoError(t, err)
				require.True(t, d.Allow)
			},
		},
		{
			// The hold is auto-only: a manual-forward for a held origin is not
			// held by it (an advancing manual is allowed).
			name:  "built-in auto-hold does not apply to a manual-forward",
			input: orderingInput("freight-new", "2026-07-15T14:00:00Z"),
			data:  heldData,
			assert: func(t *testing.T, d *Decision, err error) {
				require.NoError(t, err)
				require.True(t, d.Allow)
			},
		},
		{
			// The hold has no bypass hook: the canonical freeze-bypass custom
			// policy does not lift it (violation sets only union).
			name:          "built-in auto-hold cannot be bypassed by a custom policy",
			clusterCustom: hotfixBypassRules,
			input:         autoOrderingInput("freight-new", "2026-07-15T14:00:00Z"),
			data:          heldData,
			assert: func(t *testing.T, d *Decision, err error) {
				require.NoError(t, err)
				require.False(t, d.Allow)
				require.Contains(t, d.Message, `auto-promotion held for origin "Warehouse/demo/nginx"`)
			},
		},
		{
			// AX3: a promotion is held until its promote-after time, then
			// self-resumes via a requeue at that time.
			name:  "built-in scheduled holds a promotion until promote-after",
			input: scheduledInput("2026-07-15T18:00:00Z"),
			data:  emptyData,
			assert: func(t *testing.T, d *Decision, err error) {
				require.NoError(t, err)
				require.False(t, d.Allow)
				require.Equal(t, "scheduled; held until 2026-07-15T18:00:00Z", d.Message)
				// 15:00 -> 18:00 is three hours.
				require.Equal(t, 3*time.Hour, d.RequeueAfter)
			},
		},
		{
			name:  "built-in scheduled allows once promote-after has passed",
			input: scheduledInput("2026-07-15T12:00:00Z"),
			data:  emptyData,
			assert: func(t *testing.T, d *Decision, err error) {
				require.NoError(t, err)
				require.True(t, d.Allow)
			},
		},
		{
			name: "project violation unions with the standard blocks",
			projectCustom: `violation contains v if {
	input.project.labels.compliance == "pci"
	input.promotion.class == "manual-forward"
	not input.promotion.annotations["change-ticket"]
	v := {
		"rule": "pci-change-ticket",
		"msg": "PCI projects require a change-ticket annotation on manual promotions",
	}
}
`,
			input: func() map[string]any {
				input := testInput(ClassManualForward)
				input["project"] = map[string]any{
					"labels":      map[string]any{"compliance": "pci"},
					"annotations": map[string]any{},
				}
				return input
			}(),
			data: func() map[string]any {
				data := emptyData()
				data["freezes"] = []any{map[string]any{
					"name":          "freeze",
					"start":         "2026-07-15T00:00:00Z",
					"end":           "2026-07-16T00:00:00Z",
					"scope":         "no-forward",
					"argocdServers": []any{},
				}}
				return data
			},
			assert: func(t *testing.T, d *Decision, err error) {
				require.NoError(t, err)
				require.False(t, d.Allow)
				require.Contains(t, d.Message, "change-ticket")
				require.Contains(t, d.Message, `frozen by freeze "freeze"`)
			},
		},
		{
			name:          "cluster freeze_bypass admits a hotfix through an active freeze",
			clusterCustom: hotfixBypassRules,
			input: func() map[string]any {
				input := testInput(ClassManualForward)
				input["freight"] = map[string]any{
					"name": "test-freight",
					"images": []any{
						map[string]any{"repoURL": "example/nginx", "tag": "1.2.4", "digest": ""},
					},
				}
				input["stage"] = map[string]any{
					"name":        "prod",
					"project":     "test-project",
					"labels":      map[string]any{},
					"annotations": map[string]any{},
					"lastPromotion": map[string]any{
						"name": "promo-prev",
						"freight": map[string]any{
							"name": "prev-freight",
							"images": []any{
								map[string]any{"repoURL": "example/nginx", "tag": "1.2.3", "digest": ""},
							},
						},
					},
				}
				return input
			}(),
			data: func() map[string]any {
				data := emptyData()
				data["freezes"] = []any{map[string]any{
					"name":          "freeze",
					"start":         "2026-07-15T00:00:00Z",
					"end":           "2026-07-16T00:00:00Z",
					"scope":         "no-promotions",
					"argocdServers": []any{},
				}}
				return data
			},
			assert: func(t *testing.T, d *Decision, err error) {
				require.NoError(t, err)
				require.True(t, d.Allow)
			},
		},
		{
			name:          "cluster freeze_bypass does not admit a minor version bump",
			clusterCustom: hotfixBypassRules,
			input: func() map[string]any {
				input := testInput(ClassManualForward)
				input["freight"] = map[string]any{
					"name": "test-freight",
					"images": []any{
						map[string]any{"repoURL": "example/nginx", "tag": "1.3.0", "digest": ""},
					},
				}
				input["stage"] = map[string]any{
					"name":        "prod",
					"project":     "test-project",
					"labels":      map[string]any{},
					"annotations": map[string]any{},
					"lastPromotion": map[string]any{
						"name": "promo-prev",
						"freight": map[string]any{
							"name": "prev-freight",
							"images": []any{
								map[string]any{"repoURL": "example/nginx", "tag": "1.2.3", "digest": ""},
							},
						},
					},
				}
				return input
			}(),
			data: func() map[string]any {
				data := emptyData()
				data["freezes"] = []any{map[string]any{
					"name":          "freeze",
					"start":         "2026-07-15T00:00:00Z",
					"end":           "2026-07-16T00:00:00Z",
					"scope":         "no-promotions",
					"argocdServers": []any{},
				}}
				return data
			},
			assert: func(t *testing.T, d *Decision, err error) {
				require.NoError(t, err)
				require.False(t, d.Allow)
			},
		},
		{
			name:          "project freeze_bypass works the same way",
			projectCustom: hotfixBypassRules,
			input: func() map[string]any {
				input := testInput(ClassManualForward)
				input["freight"] = map[string]any{
					"name": "test-freight",
					"images": []any{
						map[string]any{"repoURL": "example/nginx", "tag": "1.2.4", "digest": ""},
					},
				}
				input["stage"] = map[string]any{
					"name":        "prod",
					"project":     "test-project",
					"labels":      map[string]any{},
					"annotations": map[string]any{},
					"lastPromotion": map[string]any{
						"name": "promo-prev",
						"freight": map[string]any{
							"name": "prev-freight",
							"images": []any{
								map[string]any{"repoURL": "example/nginx", "tag": "1.2.3", "digest": ""},
							},
						},
					},
				}
				return input
			}(),
			data: func() map[string]any {
				data := emptyData()
				data["freezes"] = []any{map[string]any{
					"name":          "freeze",
					"start":         "2026-07-15T00:00:00Z",
					"end":           "2026-07-16T00:00:00Z",
					"scope":         "no-promotions",
					"argocdServers": []any{},
				}}
				return data
			},
			assert: func(t *testing.T, d *Decision, err error) {
				require.NoError(t, err)
				require.True(t, d.Allow)
			},
		},
		{
			name:          "invalid custom policy surfaces a compile error",
			projectCustom: "this is not rego",
			input:         testInput(ClassAutoForward),
			data:          emptyData,
			assert: func(t *testing.T, d *Decision, err error) {
				require.Error(t, err)
				require.Nil(t, d)
				require.Contains(t, err.Error(), "preparing dispatch policy")
			},
		},
		{
			name: "custom source declaring its own package fails closed",
			projectCustom: `package kargo.dispatch

decision := {"allow": true, "message": "hijacked", "requeue_after": 0}
`,
			input: testInput(ClassAutoForward),
			data:  emptyData,
			assert: func(t *testing.T, d *Decision, err error) {
				require.Error(t, err)
				require.Nil(t, d)
				require.Contains(t, err.Error(), "contains only rules")
			},
		},
	}

	engine := NewEngine()
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			decision, err := engine.Evaluate(
				context.Background(),
				testCase.projectCustom,
				testCase.clusterCustom,
				testCase.input,
				testCase.data(),
			)
			testCase.assert(t, decision, err)
		})
	}
}

// reasonByRule returns the (first) reason with the given rule, or a zero
// Reason and false when absent.
func reasonByRule(reasons []Reason, rule string) (Reason, bool) {
	for _, r := range reasons {
		if r.Rule == rule {
			return r, true
		}
	}
	return Reason{}, false
}

// TestEngineEvaluateReasons covers the structured Decision.Reasons projection:
// each held violation surfaces its rule, and — where it has them — the queued
// Promotion it defers to (blocked_by) and the time it self-clears (until).
func TestEngineEvaluateReasons(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		input  map[string]any
		data   func() map[string]any
		assert func(*testing.T, *Decision)
	}{
		{
			name:  "freeze reason carries the rule and until",
			input: testInput(ClassAutoForward),
			data: func() map[string]any {
				data := emptyData()
				data["freezes"] = []any{map[string]any{
					"name":          "holiday",
					"start":         "2026-07-15T00:00:00Z",
					"end":           "2026-07-16T00:00:00Z",
					"scope":         "no-promotions",
					"argocdServers": []any{},
				}}
				return data
			},
			assert: func(t *testing.T, d *Decision) {
				require.Len(t, d.Reasons, 1)
				r := d.Reasons[0]
				require.Equal(t, "freezes", r.Rule)
				require.Equal(t, d.Message, r.Message)
				require.Empty(t, r.BlockedBy)
				require.NotNil(t, r.Until)
				require.Equal(t, "2026-07-16T00:00:00Z", r.Until.UTC().Format(time.RFC3339))
			},
		},
		{
			name:  "scheduled reason carries until",
			input: scheduledInput("2026-07-15T18:00:00Z"),
			data:  emptyData,
			assert: func(t *testing.T, d *Decision) {
				r, ok := reasonByRule(d.Reasons, "scheduled")
				require.True(t, ok)
				require.NotNil(t, r.Until)
				require.Equal(t, "2026-07-15T18:00:00Z", r.Until.UTC().Format(time.RFC3339))
				require.Empty(t, r.BlockedBy)
			},
		},
		{
			name:  "yield-to-rollback reason carries blocked_by and no until",
			input: testInput(ClassAutoForward),
			data: func() map[string]any {
				data := emptyData()
				data["queue"] = []any{
					map[string]any{"name": "test-promo", "class": "auto-forward", "createdAt": testNow},
					map[string]any{"name": "rb.01", "class": "rollback", "createdAt": testNow},
				}
				return data
			},
			assert: func(t *testing.T, d *Decision) {
				r, ok := reasonByRule(d.Reasons, "yield-to-rollback")
				require.True(t, ok)
				require.Equal(t, "rb.01", r.BlockedBy)
				require.Nil(t, r.Until)
			},
		},
		{
			name:  "would-regress reason has neither blocked_by nor until",
			input: orderingInput("freight-old", "2026-07-15T10:00:00Z"),
			data:  orderingData,
			assert: func(t *testing.T, d *Decision) {
				r, ok := reasonByRule(d.Reasons, "would-regress")
				require.True(t, ok)
				require.Empty(t, r.BlockedBy)
				require.Nil(t, r.Until)
			},
		},
		{
			name:  "co-firing rules each contribute a reason",
			input: orderingInput("freight-old", "2026-07-15T10:00:00Z"),
			data: func() map[string]any {
				data := orderingData()
				data["freezes"] = []any{map[string]any{
					"name":          "freeze",
					"start":         "2026-07-15T00:00:00Z",
					"end":           "2026-07-16T00:00:00Z",
					"scope":         "no-forward",
					"argocdServers": []any{},
				}}
				return data
			},
			assert: func(t *testing.T, d *Decision) {
				require.Len(t, d.Reasons, 2)
				freeze, ok := reasonByRule(d.Reasons, "freezes")
				require.True(t, ok)
				require.NotNil(t, freeze.Until)
				regress, ok := reasonByRule(d.Reasons, "would-regress")
				require.True(t, ok)
				require.Nil(t, regress.Until)
			},
		},
		{
			name:  "allow produces no reasons",
			input: testInput(ClassAutoForward),
			data:  emptyData,
			assert: func(t *testing.T, d *Decision) {
				require.True(t, d.Allow)
				require.Empty(t, d.Reasons)
			},
		},
	}

	engine := NewEngine()
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			decision, err := engine.Evaluate(
				context.Background(),
				"",
				"",
				testCase.input,
				testCase.data(),
			)
			require.NoError(t, err)
			testCase.assert(t, decision)
		})
	}
}

func TestEngineCachesCompileErrors(t *testing.T) {
	t.Parallel()
	engine := NewEngine()
	for range 2 {
		_, err := engine.Evaluate(
			context.Background(),
			"not rego at all",
			"",
			testInput(ClassAutoForward),
			emptyData(),
		)
		require.Error(t, err)
	}
	// Both evaluations must have hit the same cached preparedPolicy.
	require.Len(t, engine.prepared, 1)
}

func mustParseNS(s string) int64 {
	parsed, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(err)
	}
	return parsed.UnixNano()
}

// TestPolicySchemaEnforcement proves annotated modules are type-checked
// against the embedded JSON Schemas: a module annotated with schema.input
// that references an undeclared field must fail to compile.
func TestPolicySchemaEnforcement(t *testing.T) {
	t.Parallel()
	mods, err := policyModules("", "")
	require.NoError(t, err)
	schemas, err := policySchemas()
	require.NoError(t, err)
	mods["kargo/lib/bogus/bogus.rego"] = `# METADATA
# scope: package
# schemas:
#   - input: schema.input
package kargo.lib.bogus

import rego.v1

violation contains {"rule": "bogus", "msg": "x"} if input.promotion.clazz == "typo"
`
	modOpts, err := moduleOptions(mods)
	require.NoError(t, err)
	opts := []func(*rego.Rego){
		rego.Query(decisionQuery),
		rego.StrictBuiltinErrors(true),
		rego.Schemas(schemas),
	}
	opts = append(opts, modOpts...)
	opts = append(opts, builtins()...)
	_, err = rego.New(opts...).PrepareForEval(context.Background())
	require.Error(t, err)
	require.Contains(t, err.Error(), "clazz")
}
