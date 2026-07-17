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

// extractCustomPolicy returns the commented customPolicy block from a demo
// manifest: the lines following "  # customPolicy: |", with the comment
// prefix stripped.
func extractCustomPolicy(t *testing.T, path string) string {
	t.Helper()
	f, err := os.Open(path)
	require.NoError(t, err)
	defer f.Close()
	var (
		b       strings.Builder
		inBlock bool
	)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case strings.TrimRight(line, " ") == "  # customPolicy: |":
			inBlock = true
		case !inBlock:
		case line == "  #":
			b.WriteString("\n")
		case strings.HasPrefix(line, "  #   "):
			b.WriteString(strings.TrimPrefix(line, "  #   "))
			b.WriteString("\n")
		default:
			inBlock = false
		}
	}
	require.NoError(t, scanner.Err())
	require.NotEmpty(t, b.String(), "no commented customPolicy block in %s", path)
	return b.String()
}

func TestDemoCustomPolicies(t *testing.T) {
	t.Parallel()

	// The operator's cluster policy in two forms: the always-on PCI rule
	// (active) and the Scenario-6 expansion that adds the hotfix bypass
	// alongside it (commented). The project's prod-approval rule (commented).
	pciActive := activeCustomPolicy(t, "40-clusterconfig.yaml")
	pciPlusHotfix := extractCustomPolicy(t, "40-clusterconfig.yaml")
	prodApproval := extractCustomPolicy(t, "10-projectconfig.yaml")

	now := time.Date(2026, 7, 15, 15, 0, 0, 0, time.UTC)

	freeze := func(name string) []kargoapi.PromotionExclusion {
		return []kargoapi.PromotionExclusion{{
			Name:  name,
			Start: metav1.Time{Time: now.Add(-time.Hour)},
			End:   metav1.Time{Time: now.Add(9 * time.Hour)},
			Scope: "no-forward",
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
		exclusions    []kargoapi.PromotionExclusion
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
			exclusions:    freeze("holiday-freeze"),
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
			exclusions:    freeze("incident-freeze"),
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
			exclusions:    freeze("holiday-freeze"),
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
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			stage := &kargoapi.Stage{ObjectMeta: metav1.ObjectMeta{
				Name:      "prod",
				Namespace: "policy-demo",
			}}
			data, err := dispatch.BuildData(nil, testCase.exclusions, stage, nil, nil)
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
