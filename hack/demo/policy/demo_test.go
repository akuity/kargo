// Package policy_test verifies the customPolicy example blocks in the demo
// manifests -- both the always-on active rule and the commented blocks the
// scenarios toggle on -- against the real dispatch policy engine, end to end
// through the exported input/data projections.
package policy_test

import (
	"bufio"
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/promotion/dispatch"
)

// activeCustomPolicy returns the live spec.customPolicy from a demo manifest
// (the rule that ships active, e.g. the operator's PCI mandate). YAML comments
// -- including the commented-out example blocks -- are ignored by the parser.
func activeCustomPolicy(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	var doc struct {
		Spec struct {
			CustomPolicy string `json:"customPolicy"`
		} `json:"spec"`
	}
	require.NoError(t, yaml.Unmarshal(data, &doc))
	require.NotEmpty(t, doc.Spec.CustomPolicy, "no active customPolicy in %s", path)
	return doc.Spec.CustomPolicy
}

// extractCustomPolicy returns a commented customPolicy block from a demo
// manifest: the lines following "  # customPolicy: |", with the comment
// prefix stripped. A file may carry several such blocks (alternative
// policies a scenario swaps in); anchor selects one by a substring that
// appears in a comment at or before its header (e.g. "Scenario 6"). An
// empty anchor takes the first block.
func extractCustomPolicy(t *testing.T, path, anchor string) string {
	t.Helper()
	f, err := os.Open(path)
	require.NoError(t, err)
	defer f.Close()
	var b strings.Builder
	armed := anchor == ""
	inBlock := false
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if !armed {
			if strings.Contains(line, anchor) {
				armed = true
			}
			continue
		}
		switch {
		case strings.TrimRight(line, " ") == "  # customPolicy: |":
			inBlock = true
		case !inBlock:
			// Waiting for the block to start.
		case line == "  #":
			b.WriteString("\n")
		case strings.HasPrefix(line, "  #   "):
			b.WriteString(strings.TrimPrefix(line, "  #   "))
			b.WriteString("\n")
		default:
			// A non-comment line ends the block.
			if b.Len() > 0 {
				require.NoError(t, scanner.Err())
				return b.String()
			}
			inBlock = false
		}
	}
	require.NoError(t, scanner.Err())
	require.NotEmpty(t, b.String(), "no commented customPolicy block after %q in %s", anchor, path)
	return b.String()
}

func TestDemoCustomPolicies(t *testing.T) {
	t.Parallel()

	// The operator's cluster policy in two forms: the always-on PCI rule
	// (active) and the Scenario-6 expansion that adds the hotfix bypass
	// alongside it (commented). The project's prod-approval rule (commented).
	pciActive := activeCustomPolicy(t, "40-clusterconfig.yaml")
	pciPlusHotfix := extractCustomPolicy(t, "40-clusterconfig.yaml", "Scenario 6")
	queueOrdering := extractCustomPolicy(t, "40-clusterconfig.yaml", "Scenario 8")
	prodApproval := extractCustomPolicy(t, "10-projectconfig.yaml", "approved-by")

	now := time.Date(2026, 7, 15, 15, 0, 0, 0, time.UTC)

	freeze := func(name string) []kargoapi.PromotionFreeze {
		return []kargoapi.PromotionFreeze{{
			Name:  name,
			Start: metav1.Time{Time: now.Add(-time.Hour)},
			End:   metav1.Time{Time: now.Add(9 * time.Hour)},
			Scope: "no-forward",
		}}
	}
	// queuePromo builds a Pending Promotion of the given class for the
	// data.queue projection: the class is inferred from annotations by
	// ClassOf, exactly as the controller derives it.
	queuePromo := func(name, class string) kargoapi.Promotion {
		ann := map[string]string{}
		switch class {
		case dispatch.ClassManualForward:
			ann[kargoapi.AnnotationKeyCreateActor] = "admin"
		case dispatch.ClassRollback:
			ann[kargoapi.AnnotationKeyRollback] = kargoapi.AnnotationValueTrue
		}
		return kargoapi.Promotion{ObjectMeta: metav1.ObjectMeta{
			Name:              name,
			CreationTimestamp: metav1.Time{Time: now},
			Annotations:       ann,
		}}
	}
	ticketOnly := map[string]string{"change-ticket": "CHG-1234"}
	// makeInput assembles the input document for a candidate Promotion of
	// the given class to the prod Stage of a PCI-labeled Project. When
	// lastTag is set, the Stage's last promoted freight shares the image
	// repository, making a hotfix determination possible.
	makeInput := func(class string, annotations map[string]string, lastTag string) map[string]any {
		promoAnnotations := map[string]string{}
		for k, v := range annotations {
			promoAnnotations[k] = v
		}
		switch class {
		case dispatch.ClassManualForward:
			promoAnnotations[kargoapi.AnnotationKeyCreateActor] = "admin"
		case dispatch.ClassRollback:
			promoAnnotations[kargoapi.AnnotationKeyRollback] = kargoapi.AnnotationValueTrue
		}
		promo := &kargoapi.Promotion{ObjectMeta: metav1.ObjectMeta{
			Name:              "prod.01-fresh",
			Namespace:         "policy-demo",
			CreationTimestamp: metav1.Time{Time: now},
			Annotations:       promoAnnotations,
		}}
		freight := &kargoapi.Freight{
			ObjectMeta: metav1.ObjectMeta{Name: "fresh-freight"},
			Images:     []kargoapi.Image{{RepoURL: "example/nginx", Tag: "1.2.3"}},
		}
		stage := &kargoapi.Stage{ObjectMeta: metav1.ObjectMeta{
			Name:      "prod",
			Namespace: "policy-demo",
		}}
		if lastTag != "" {
			stage.Status.LastPromotion = &kargoapi.PromotionReference{
				Name: "prod.00-old",
				Freight: &kargoapi.FreightReference{
					Name:   "old-freight",
					Images: []kargoapi.Image{{RepoURL: "example/nginx", Tag: lastTag}},
				},
			}
		}
		project := &kargoapi.Project{ObjectMeta: metav1.ObjectMeta{
			Name:   "policy-demo",
			Labels: map[string]string{"compliance": "pci"},
		}}
		return dispatch.BuildInput(promo, freight, stage, project, nil, now)
	}

	engine := dispatch.NewEngine()

	testCases := []struct {
		name          string
		clusterCustom string
		projectCustom string
		class         string
		annotations   map[string]string
		lastTag       string
		freezes       []kargoapi.PromotionFreeze
		queue         []kargoapi.Promotion
		assert        func(*testing.T, *dispatch.Decision)
	}{
		// The always-on PCI rule (Scenario 5).
		{
			name:          "PCI: ticketless manual promotion is denied",
			clusterCustom: pciActive,
			class:         dispatch.ClassManualForward,
			assert: func(t *testing.T, d *dispatch.Decision) {
				require.False(t, d.Allow)
				require.Contains(t, d.Message, "change-ticket")
			},
		},
		{
			name:          "PCI: auto promotion is unaffected",
			clusterCustom: pciActive,
			class:         dispatch.ClassAutoForward,
			assert: func(t *testing.T, d *dispatch.Decision) {
				require.True(t, d.Allow)
			},
		},
		{
			name:          "PCI: manual promotion with a change-ticket is allowed",
			clusterCustom: pciActive,
			class:         dispatch.ClassManualForward,
			annotations:   ticketOnly,
			assert: func(t *testing.T, d *dispatch.Decision) {
				require.True(t, d.Allow)
			},
		},
		// The hotfix lane, added to the cluster policy (Scenario 6). The PCI
		// rule is retained in this combined form.
		{
			name:          "hotfix: annotated patch-bump passes the holiday freeze",
			clusterCustom: pciPlusHotfix,
			class:         dispatch.ClassManualForward,
			annotations:   ticketOnly,
			lastTag:       "1.2.2",
			freezes:       freeze("holiday-freeze"),
			assert: func(t *testing.T, d *dispatch.Decision) {
				require.True(t, d.Allow)
			},
		},
		{
			name:          "hotfix: annotated patch-bump stays held by an incident freeze (bypass is name-scoped)",
			clusterCustom: pciPlusHotfix,
			class:         dispatch.ClassManualForward,
			annotations:   ticketOnly,
			lastTag:       "1.2.2",
			freezes:       freeze("incident-freeze"),
			assert: func(t *testing.T, d *dispatch.Decision) {
				require.False(t, d.Allow)
				require.Contains(t, d.Message, "incident-freeze")
			},
		},
		{
			name:          "hotfix: annotated minor bump stays held by the holiday freeze",
			clusterCustom: pciPlusHotfix,
			class:         dispatch.ClassManualForward,
			annotations:   ticketOnly,
			lastTag:       "1.1.9",
			freezes:       freeze("holiday-freeze"),
			assert: func(t *testing.T, d *dispatch.Decision) {
				require.False(t, d.Allow)
				require.Contains(t, d.Message, "holiday-freeze")
			},
		},
		{
			name:          "hotfix: the retained PCI rule still denies a ticketless manual promotion",
			clusterCustom: pciPlusHotfix,
			class:         dispatch.ClassManualForward,
			assert: func(t *testing.T, d *dispatch.Decision) {
				require.False(t, d.Allow)
				require.Contains(t, d.Message, "change-ticket")
			},
		},
		// The project's prod-approval rule (Scenario 7).
		{
			name:          "prod-approval: forward promotion without approved-by is denied",
			projectCustom: prodApproval,
			class:         dispatch.ClassManualForward,
			assert: func(t *testing.T, d *dispatch.Decision) {
				require.False(t, d.Allow)
				require.Contains(t, d.Message, "approved-by")
				require.NotContains(t, d.Message, "change-ticket")
			},
		},
		{
			name:          "prod-approval: rollback is exempt",
			projectCustom: prodApproval,
			class:         dispatch.ClassRollback,
			assert: func(t *testing.T, d *dispatch.Decision) {
				require.True(t, d.Allow)
			},
		},
		// The queue-aware ordering policy (Scenario 8), reading data.queue.
		{
			name:          "queue: a forward yields to a queued rollback",
			clusterCustom: queueOrdering,
			class:         dispatch.ClassAutoForward,
			queue: []kargoapi.Promotion{
				queuePromo("prod.01-fresh", dispatch.ClassAutoForward),
				queuePromo("rb.02", dispatch.ClassRollback),
			},
			assert: func(t *testing.T, d *dispatch.Decision) {
				require.False(t, d.Allow)
				require.Contains(t, d.Message, `yielding to queued rollback "rb.02"`)
			},
		},
		{
			name:          "queue: a manual forward also yields to a queued rollback",
			clusterCustom: queueOrdering,
			class:         dispatch.ClassManualForward,
			queue: []kargoapi.Promotion{
				queuePromo("prod.01-fresh", dispatch.ClassManualForward),
				queuePromo("rb.02", dispatch.ClassRollback),
			},
			assert: func(t *testing.T, d *dispatch.Decision) {
				require.False(t, d.Allow)
				require.Contains(t, d.Message, "yielding to queued rollback")
			},
		},
		{
			name:          "queue: the rollback itself is not held",
			clusterCustom: queueOrdering,
			class:         dispatch.ClassRollback,
			queue: []kargoapi.Promotion{
				queuePromo("prod.01-fresh", dispatch.ClassRollback),
			},
			assert: func(t *testing.T, d *dispatch.Decision) {
				require.True(t, d.Allow)
			},
		},
		{
			name:          "queue: a forward dispatches when no rollback is queued",
			clusterCustom: queueOrdering,
			class:         dispatch.ClassAutoForward,
			queue: []kargoapi.Promotion{
				queuePromo("prod.01-fresh", dispatch.ClassAutoForward),
				queuePromo("fwd.02", dispatch.ClassAutoForward),
			},
			assert: func(t *testing.T, d *dispatch.Decision) {
				require.True(t, d.Allow)
			},
		},
		{
			name:          "queue: a deep backlog holds an auto promotion (backpressure)",
			clusterCustom: queueOrdering,
			class:         dispatch.ClassAutoForward,
			queue: []kargoapi.Promotion{
				queuePromo("prod.01-fresh", dispatch.ClassAutoForward),
				queuePromo("fwd.02", dispatch.ClassAutoForward),
				queuePromo("fwd.03", dispatch.ClassAutoForward),
				queuePromo("fwd.04", dispatch.ClassAutoForward),
			},
			assert: func(t *testing.T, d *dispatch.Decision) {
				require.False(t, d.Allow)
				require.Contains(t, d.Message, "deep backlog (4 queued)")
			},
		},
		{
			name:          "queue: backpressure does not hold a manual promotion",
			clusterCustom: queueOrdering,
			class:         dispatch.ClassManualForward,
			queue: []kargoapi.Promotion{
				queuePromo("prod.01-fresh", dispatch.ClassManualForward),
				queuePromo("fwd.02", dispatch.ClassAutoForward),
				queuePromo("fwd.03", dispatch.ClassAutoForward),
				queuePromo("fwd.04", dispatch.ClassAutoForward),
			},
			assert: func(t *testing.T, d *dispatch.Decision) {
				require.True(t, d.Allow)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			stage := &kargoapi.Stage{ObjectMeta: metav1.ObjectMeta{
				Name:      "prod",
				Namespace: "policy-demo",
			}}
			data, err := dispatch.BuildData(nil, testCase.freezes, stage, nil, nil, testCase.queue)
			require.NoError(t, err)
			decision, err := engine.Evaluate(
				context.Background(),
				testCase.projectCustom,
				testCase.clusterCustom,
				makeInput(testCase.class, testCase.annotations, testCase.lastTag),
				data,
			)
			require.NoError(t, err)
			testCase.assert(t, decision)
		})
	}
}
