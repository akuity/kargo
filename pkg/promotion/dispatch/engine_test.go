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
// cluster (operator) policy: hotfixes bypass every exclusion. Hotfix
// semantics are defined in the custom policy itself; the stdlib supplies
// only the semver building block. Rules only -- the engine prepends the
// package and imports.
const hotfixBypassRules = `exclusions_bypass(e) if is_hotfix

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
				require.False(t, d.Allow)
				require.Contains(t, d.Message, "change-ticket")
				require.Contains(t, d.Message, `frozen by exclusion "freeze"`)
			},
		},
		{
			name:          "cluster exclusions_bypass admits a hotfix through an active exclusion",
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
			name:          "cluster exclusions_bypass does not admit a minor version bump",
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
			name:          "project exclusions_bypass works the same way",
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
