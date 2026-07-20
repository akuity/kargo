package dispatch

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func TestClassOf(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name        string
		annotations map[string]string
		class       string
	}{
		{
			name:  "no annotations is auto-forward",
			class: ClassAutoForward,
		},
		{
			name: "rollback annotation wins",
			annotations: map[string]string{
				kargoapi.AnnotationKeyRollback:    "true",
				kargoapi.AnnotationKeyCreateActor: "admin",
			},
			class: ClassRollback,
		},
		{
			name: "controller actor is auto-forward",
			annotations: map[string]string{
				kargoapi.AnnotationKeyCreateActor: kargoapi.EventActorControllerPrefix + "stage-controller",
			},
			class: ClassAutoForward,
		},
		{
			name: "user actor is manual-forward",
			annotations: map[string]string{
				kargoapi.AnnotationKeyCreateActor: "admin",
			},
			class: ClassManualForward,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			promo := &kargoapi.Promotion{
				ObjectMeta: metav1.ObjectMeta{Annotations: testCase.annotations},
			}
			require.Equal(t, testCase.class, ClassOf(promo))
		})
	}
}

func TestSelectorMatches(t *testing.T) {
	t.Parallel()
	stage := &kargoapi.Stage{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "prod-east",
			Labels: map[string]string{"env": "prod"},
		},
	}
	testCases := []struct {
		name     string
		selector *kargoapi.PromotionPolicySelector
		matches  bool
		wantErr  bool
	}{
		{
			name:    "nil selector matches everything",
			matches: true,
		},
		{
			name:     "exact name match",
			selector: &kargoapi.PromotionPolicySelector{Name: "prod-east"},
			matches:  true,
		},
		{
			name:     "exact name mismatch",
			selector: &kargoapi.PromotionPolicySelector{Name: "uat"},
			matches:  false,
		},
		{
			name:     "glob pattern match",
			selector: &kargoapi.PromotionPolicySelector{Name: "glob:prod-*"},
			matches:  true,
		},
		{
			name: "label selector match",
			selector: &kargoapi.PromotionPolicySelector{
				LabelSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"env": "prod"},
				},
			},
			matches: true,
		},
		{
			name: "label selector mismatch",
			selector: &kargoapi.PromotionPolicySelector{
				LabelSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"env": "dev"},
				},
			},
			matches: false,
		},
		{
			name: "name and label are ANDed",
			selector: &kargoapi.PromotionPolicySelector{
				Name: "glob:prod-*",
				LabelSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"env": "dev"},
				},
			},
			matches: false,
		},
		{
			name:     "invalid pattern errors",
			selector: &kargoapi.PromotionPolicySelector{Name: "regex:["},
			wantErr:  true,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			matches, err := selectorMatches(testCase.selector, stage)
			if testCase.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, testCase.matches, matches)
		})
	}
}

func TestBuildData(t *testing.T) {
	t.Parallel()
	stage := &kargoapi.Stage{
		ObjectMeta: metav1.ObjectMeta{Name: "prod", Namespace: "demo"},
	}
	projectSpec := &kargoapi.ProjectConfigSpec{
		PromotionWindows: []kargoapi.PromotionWindow{
			{
				Name:          "prod-window",
				StageSelector: &kargoapi.PromotionPolicySelector{Name: "prod"},
				Recurrence:    "FREQ=DAILY",
				Start:         "09:00",
				End:           "17:00",
				Location:      "UTC",
			},
			{
				Name:          "uat-window",
				StageSelector: &kargoapi.PromotionPolicySelector{Name: "uat"},
				Recurrence:    "FREQ=DAILY",
				Start:         "00:00",
				End:           "23:59",
			},
		},
		RateLimits: []kargoapi.PromotionRateLimit{
			{
				Name:          "uat-throttle",
				StageSelector: &kargoapi.PromotionPolicySelector{Name: "uat"},
				MaxPromotions: 5,
				Window:        metav1.Duration{Duration: time.Hour},
			},
			{
				Name:          "default-throttle",
				MaxPromotions: 2,
				Window:        metav1.Duration{Duration: 30 * time.Minute},
			},
		},
	}
	freezes := []kargoapi.PromotionFreeze{{
		Name:          "holiday",
		Start:         metav1.Date(2026, 12, 20, 0, 0, 0, 0, time.UTC),
		End:           metav1.Date(2027, 1, 2, 0, 0, 0, 0, time.UTC),
		Scope:         "no-forward",
		ArgoCDServers: []string{"https://prod.example.com"},
	}}
	dispatched := time.Date(2026, 7, 15, 14, 40, 0, 0, time.UTC)
	created := time.Date(2026, 7, 15, 14, 30, 0, 0, time.UTC)
	queue := []kargoapi.Promotion{{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "prod.01",
			CreationTimestamp: metav1.NewTime(created),
			Annotations:       map[string]string{kargoapi.AnnotationKeyRollback: kargoapi.AnnotationValueTrue},
		},
	}}

	data, err := BuildData(projectSpec, freezes, stage, nil, []time.Time{dispatched}, queue)
	require.NoError(t, err)

	// Only the window whose selector matches this Stage is projected.
	require.Equal(t, []any{map[string]any{
		"name":       "prod-window",
		"recurrence": "FREQ=DAILY",
		"start":      "09:00",
		"end":        "17:00",
		"location":   "UTC",
	}}, data["windows"])

	// The first rate limit does not match; the second (selector-less) does.
	require.Equal(t, map[string]any{"prod": map[string]any{
		"max":        int64(2),
		"window":     (30 * time.Minute).Nanoseconds(),
		"dispatches": []any{dispatched.UnixNano()},
	}}, data["rateLimit"])

	require.Equal(t, []any{map[string]any{
		"name":          "holiday",
		"start":         "2026-12-20T00:00:00Z",
		"end":           "2027-01-02T00:00:00Z",
		"scope":         "no-forward",
		"argocdServers": []any{"https://prod.example.com"},
	}}, data["freezes"])

	require.Equal(t, defaultScopes, data["scopes"])

	// The queue projects each awaiting Promotion's identity, class, and
	// creation time, preserving the given order.
	require.Equal(t, []any{map[string]any{
		"name":      "prod.01",
		"class":     ClassRollback,
		"createdAt": "2026-07-15T14:30:00Z",
	}}, data["queue"])
}

func TestBuildDataNilPolicy(t *testing.T) {
	t.Parallel()
	stage := &kargoapi.Stage{ObjectMeta: metav1.ObjectMeta{Name: "prod"}}
	data, err := BuildData(nil, nil, stage, nil, nil, nil)
	require.NoError(t, err)
	require.Empty(t, data["windows"])
	require.Empty(t, data["freezes"])
	require.Empty(t, data["rateLimit"])
	require.Empty(t, data["queue"])
}

func TestBuildDataProjectSelector(t *testing.T) {
	t.Parallel()
	stage := &kargoapi.Stage{
		ObjectMeta: metav1.ObjectMeta{Name: "prod", Namespace: "demo"},
	}
	pciProject := &kargoapi.Project{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "demo",
			Labels: map[string]string{"compliance": "pci", "env": "prod"},
		},
	}
	freeze := func(selector *metav1.LabelSelector) []kargoapi.PromotionFreeze {
		return []kargoapi.PromotionFreeze{{
			Name:            "freeze",
			Start:           metav1.Date(2026, 12, 20, 0, 0, 0, 0, time.UTC),
			End:             metav1.Date(2027, 1, 2, 0, 0, 0, 0, time.UTC),
			Scope:           "no-forward",
			ProjectSelector: selector,
		}}
	}
	// names returns the "name" of each projected freeze.
	names := func(t *testing.T, data map[string]any) []string {
		t.Helper()
		docs, ok := data["freezes"].([]any)
		require.True(t, ok)
		out := make([]string, len(docs))
		for i, d := range docs {
			m, ok := d.(map[string]any)
			require.True(t, ok)
			out[i], _ = m["name"].(string)
		}
		return out
	}

	testCases := []struct {
		name     string
		selector *metav1.LabelSelector
		project  *kargoapi.Project
		assert   func(*testing.T, map[string]any, error)
	}{
		{
			name:     "nil selector applies to every Project",
			selector: nil,
			project:  pciProject,
			assert: func(t *testing.T, data map[string]any, err error) {
				require.NoError(t, err)
				require.Equal(t, []string{"freeze"}, names(t, data))
			},
		},
		{
			name:     "matchLabels matching the Project is projected",
			selector: &metav1.LabelSelector{MatchLabels: map[string]string{"compliance": "pci"}},
			project:  pciProject,
			assert: func(t *testing.T, data map[string]any, err error) {
				require.NoError(t, err)
				require.Equal(t, []string{"freeze"}, names(t, data))
			},
		},
		{
			name:     "matchLabels not matching the Project is filtered out",
			selector: &metav1.LabelSelector{MatchLabels: map[string]string{"compliance": "pci"}},
			project: &kargoapi.Project{ObjectMeta: metav1.ObjectMeta{
				Name:   "demo",
				Labels: map[string]string{"env": "prod"},
			}},
			assert: func(t *testing.T, data map[string]any, err error) {
				require.NoError(t, err)
				require.Empty(t, names(t, data))
			},
		},
		{
			name: "matchExpressions In matching the Project is projected",
			selector: &metav1.LabelSelector{MatchExpressions: []metav1.LabelSelectorRequirement{{
				Key:      "env",
				Operator: metav1.LabelSelectorOpIn,
				Values:   []string{"prod", "staging"},
			}}},
			project: pciProject,
			assert: func(t *testing.T, data map[string]any, err error) {
				require.NoError(t, err)
				require.Equal(t, []string{"freeze"}, names(t, data))
			},
		},
		{
			name:     "nil Project with a matchLabels selector is filtered out",
			selector: &metav1.LabelSelector{MatchLabels: map[string]string{"compliance": "pci"}},
			project:  nil,
			assert: func(t *testing.T, data map[string]any, err error) {
				require.NoError(t, err)
				require.Empty(t, names(t, data))
			},
		},
		{
			name: "invalid selector returns an error",
			selector: &metav1.LabelSelector{MatchExpressions: []metav1.LabelSelectorRequirement{{
				Key:      "env",
				Operator: metav1.LabelSelectorOpIn, // In requires values; none given
			}}},
			project: pciProject,
			assert: func(t *testing.T, _ map[string]any, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "freeze")
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			data, err := BuildData(nil, freeze(testCase.selector), stage, testCase.project, nil, nil)
			testCase.assert(t, data, err)
		})
	}
}

func TestBuildInput(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 7, 15, 15, 0, 0, 0, time.UTC)
	promo := &kargoapi.Promotion{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "promo-1",
			CreationTimestamp: metav1.NewTime(now.Add(-time.Hour)),
			Annotations: map[string]string{
				kargoapi.AnnotationKeyCreateActor: "admin",
			},
		},
	}
	freight := &kargoapi.Freight{
		ObjectMeta: metav1.ObjectMeta{Name: "freight-1"},
		Alias:      "salty-pike",
		Images:     []kargoapi.Image{{RepoURL: "example/nginx", Tag: "1.2.4"}},
	}
	stage := &kargoapi.Stage{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "prod",
			Namespace: "demo",
			Labels:    map[string]string{"env": "prod"},
		},
		Status: kargoapi.StageStatus{
			LastPromotion: &kargoapi.PromotionReference{
				Name: "promo-0",
				Freight: &kargoapi.FreightReference{
					Name:   "freight-0",
					Images: []kargoapi.Image{{RepoURL: "example/nginx", Tag: "1.2.3"}},
				},
			},
		},
	}
	project := &kargoapi.Project{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "demo",
			Labels: map[string]string{"team": "payments"},
		},
	}

	input := BuildInput(promo, freight, stage, project, nil, now)

	promoDoc, ok := input["promotion"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "promo-1", promoDoc["name"])
	require.Equal(t, ClassManualForward, promoDoc["class"])
	require.Equal(t, "admin", promoDoc["actor"])

	freightDoc, ok := input["freight"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "freight-1", freightDoc["name"])
	require.Equal(t, "salty-pike", freightDoc["alias"])
	require.Equal(t,
		[]any{map[string]any{"repoURL": "example/nginx", "tag": "1.2.4", "digest": ""}},
		freightDoc["images"],
	)

	stageDoc, ok := input["stage"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "prod", stageDoc["name"])
	require.Equal(t, "demo", stageDoc["project"])
	lastPromo, ok := stageDoc["lastPromotion"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "promo-0", lastPromo["name"])
	lastFreight, ok := lastPromo["freight"].(map[string]any)
	require.True(t, ok)
	require.Equal(t,
		[]any{map[string]any{"repoURL": "example/nginx", "tag": "1.2.3", "digest": ""}},
		lastFreight["images"],
	)

	projectDoc, ok := input["project"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, map[string]any{"team": "payments"}, projectDoc["labels"])

	require.Equal(t, "2026-07-15T15:00:00Z", input["now"])
	require.Equal(t, []any{}, input["applications"])
}

func TestBuildInputNilSafety(t *testing.T) {
	t.Parallel()
	promo := &kargoapi.Promotion{ObjectMeta: metav1.ObjectMeta{Name: "promo-1"}}
	stage := &kargoapi.Stage{ObjectMeta: metav1.ObjectMeta{Name: "prod", Namespace: "demo"}}
	input := BuildInput(promo, nil, stage, nil, nil, time.Now())
	require.Equal(t, map[string]any{}, input["freight"])
	stageDoc, ok := input["stage"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, map[string]any{}, stageDoc["lastPromotion"])
}
