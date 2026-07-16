// Package policy_test verifies the commented customPolicy example blocks
// in the demo manifests against the real dispatch policy engine, end to
// end through the exported input/data projections.
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

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/promotion/dispatch"
)

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

	projectCustom := extractCustomPolicy(t, "10-projectconfig.yaml")
	clusterCustom := extractCustomPolicy(t, "40-clusterconfig.yaml")

	now := time.Date(2026, 7, 15, 15, 0, 0, 0, time.UTC)

	freeze := func(name string) []kargoapi.PromotionExclusion {
		return []kargoapi.PromotionExclusion{{
			Name:  name,
			Start: metav1.Time{Time: now.Add(-time.Hour)},
			End:   metav1.Time{Time: now.Add(9 * time.Hour)},
			Scope: "no-forward",
		}}
	}
	bothAnnotations := map[string]string{
		"change-ticket": "CHG-1234",
		"approved-by":   "eron",
	}
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
		name        string
		class       string
		annotations map[string]string
		lastTag     string
		exclusions  []kargoapi.PromotionExclusion
		assert      func(*testing.T, *dispatch.Decision)
	}{
		{
			name:  "ticketless unapproved manual promotion trips both custom rules",
			class: dispatch.ClassManualForward,
			assert: func(t *testing.T, d *dispatch.Decision) {
				require.False(t, d.Allow)
				require.Contains(t, d.Message, "change-ticket")
				require.Contains(t, d.Message, "approved-by")
			},
		},
		{
			name:  "unapproved auto promotion is held by the approval rule only",
			class: dispatch.ClassAutoForward,
			assert: func(t *testing.T, d *dispatch.Decision) {
				require.False(t, d.Allow)
				require.Contains(t, d.Message, "approved-by")
				require.NotContains(t, d.Message, "change-ticket")
			},
		},
		{
			name:  "rollback needs no approval",
			class: dispatch.ClassRollback,
			assert: func(t *testing.T, d *dispatch.Decision) {
				require.True(t, d.Allow)
			},
		},
		{
			name:        "annotated patch-bump passes the holiday freeze via the hotfix lane",
			class:       dispatch.ClassManualForward,
			annotations: bothAnnotations,
			lastTag:     "1.2.2",
			exclusions:  freeze("holiday-freeze"),
			assert: func(t *testing.T, d *dispatch.Decision) {
				require.True(t, d.Allow)
			},
		},
		{
			name:        "annotated patch-bump stays held by an incident freeze (bypass is name-scoped)",
			class:       dispatch.ClassManualForward,
			annotations: bothAnnotations,
			lastTag:     "1.2.2",
			exclusions:  freeze("incident-freeze"),
			assert: func(t *testing.T, d *dispatch.Decision) {
				require.False(t, d.Allow)
				require.Contains(t, d.Message, "incident-freeze")
			},
		},
		{
			name:        "annotated minor bump stays held by the holiday freeze",
			class:       dispatch.ClassManualForward,
			annotations: bothAnnotations,
			lastTag:     "1.1.9",
			exclusions:  freeze("holiday-freeze"),
			assert: func(t *testing.T, d *dispatch.Decision) {
				require.False(t, d.Allow)
				require.Contains(t, d.Message, "holiday-freeze")
			},
		},
		{
			name:        "annotated manual promotion is allowed with no freeze",
			class:       dispatch.ClassManualForward,
			annotations: bothAnnotations,
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
			data, err := dispatch.BuildData(nil, testCase.exclusions, stage, nil)
			require.NoError(t, err)
			decision, err := engine.Evaluate(
				context.Background(),
				projectCustom,
				clusterCustom,
				makeInput(testCase.class, testCase.annotations, testCase.lastTag),
				data,
			)
			require.NoError(t, err)
			testCase.assert(t, decision)
		})
	}
}
