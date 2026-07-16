package dispatch

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// testNow is a Wednesday, 15:00 UTC (07:00 in America/Los_Angeles).
const testNow = "2026-07-15T15:00:00Z"

func emptyData() map[string]any {
	return map[string]any{
		"windows":    []any{},
		"exclusions": []any{},
		"scopes":     defaultScopes,
		"rateLimit":  map[string]any{},
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

func TestEngineEvaluate(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		custom string
		input  map[string]any
		data   func() map[string]any
		assert func(*testing.T, *Decision, error)
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
			name:  "active exclusion denies with requeue until its end",
			input: testInput(ClassAutoForward),
			data: func() map[string]any {
				data := emptyData()
				data["exclusions"] = []any{map[string]any{
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
				require.Contains(t, d.Message, `frozen by exclusion "holiday"`)
				// 9 hours from testNow to the end of the exclusion.
				require.Equal(t, 9*time.Hour, d.RequeueAfter)
			},
		},
		{
			name:  "expired exclusion does not deny",
			input: testInput(ClassAutoForward),
			data: func() map[string]any {
				data := emptyData()
				data["exclusions"] = []any{map[string]any{
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
			name:  "no-forward exclusion permits rollbacks",
			input: testInput(ClassRollback),
			data: func() map[string]any {
				data := emptyData()
				data["exclusions"] = []any{map[string]any{
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
			name:  "no-auto exclusion permits manual promotions",
			input: testInput(ClassManualForward),
			data: func() map[string]any {
				data := emptyData()
				data["exclusions"] = []any{map[string]any{
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
			name: "server-scoped exclusion applies only to matching destinations",
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
				data["exclusions"] = []any{map[string]any{
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
			name: "server-scoped exclusion denies matching destinations",
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
				data["exclusions"] = []any{map[string]any{
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
				data["exclusions"] = []any{map[string]any{
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
				require.Contains(t, d.Message, "frozen by exclusion")
				require.Contains(t, d.Message, "outside all promotion windows")
				// Window opens at 18:00 (3h), exclusion ends at 00:00 (9h);
				// the soonest boundary wins.
				require.Equal(t, 3*time.Hour, d.RequeueAfter)
			},
		},
		{
			name: "custom policy replaces the default",
			custom: `package kargo.dispatch

import rego.v1

decision := {"allow": false, "message": "computer says no", "requeue_after": 60}
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
			name: "custom policy composes lib blocks and adds its own rule",
			custom: `package kargo.dispatch

import rego.v1

import data.kargo.lib.exclusions
import data.kargo.lib.windows

violation contains v if some v in windows.violation

violation contains v if some v in exclusions.violation

violation contains v if {
	input.stage.name == "prod"
	input.promotion.class == "auto-forward"
	some img in input.freight.images
	contains(img.tag, "-")
	v := {
		"rule": "no-prerelease-prod",
		"msg": sprintf("prerelease %q is not auto-promoted to prod", [img.tag]),
	}
}

decision := {"allow": count(violation) == 0, "message": concat("; ", [v.msg | some v in violation])}
`,
			input: func() map[string]any {
				input := testInput(ClassAutoForward)
				input["freight"] = map[string]any{
					"name": "test-freight",
					"images": []any{
						map[string]any{"repoURL": "example/nginx", "tag": "1.4.0-rc.1", "digest": ""},
					},
				}
				return input
			}(),
			data: emptyData,
			assert: func(t *testing.T, d *Decision, err error) {
				require.NoError(t, err)
				require.False(t, d.Allow)
				require.Contains(t, d.Message, `prerelease "1.4.0-rc.1" is not auto-promoted to prod`)
			},
		},
		{
			name: "custom policy bypasses exclusions for hotfixes via helpers",
			custom: `package kargo.dispatch

import rego.v1

import data.kargo.lib.exclusions
import data.kargo.lib.helpers

violation contains v if {
	some v in exclusions.violation
	not helpers.is_hotfix
}

decision := {"allow": count(violation) == 0, "message": concat("; ", [v.msg | some v in violation])}
`,
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
				data["exclusions"] = []any{map[string]any{
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
			name: "hotfix bypass does not admit a minor version bump",
			custom: `package kargo.dispatch

import rego.v1

import data.kargo.lib.exclusions
import data.kargo.lib.helpers

violation contains v if {
	some v in exclusions.violation
	not helpers.is_hotfix
}

decision := {"allow": count(violation) == 0, "message": concat("; ", [v.msg | some v in violation])}
`,
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
				data["exclusions"] = []any{map[string]any{
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
			name:   "invalid custom policy surfaces a compile error",
			custom: "this is not rego",
			input:  testInput(ClassAutoForward),
			data:   emptyData,
			assert: func(t *testing.T, d *Decision, err error) {
				require.Error(t, err)
				require.Nil(t, d)
				require.Contains(t, err.Error(), "preparing dispatch policy")
			},
		},
		{
			name: "custom policy without a decision surfaces an error",
			custom: `package kargo.dispatch

import rego.v1

some_other_rule := true
`,
			input: testInput(ClassAutoForward),
			data:  emptyData,
			assert: func(t *testing.T, d *Decision, err error) {
				require.Error(t, err)
				require.Nil(t, d)
				require.Contains(t, err.Error(), "did not produce a decision")
			},
		},
	}

	engine := NewEngine()
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			decision, err := engine.Evaluate(
				context.Background(),
				testCase.custom,
				testCase.input,
				testCase.data(),
			)
			testCase.assert(t, decision, err)
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
