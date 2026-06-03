package promotions

import (
	"testing"

	"github.com/stretchr/testify/require"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func TestPromotionSelectorsMatchApp(t *testing.T) {
	const (
		project = "fake-project"
		stage   = "fake-stage"
	)

	matchExprConfig := `{"apps":[{"namespace":"argocd","selector":` +
		`{"matchExpressions":[{"key":"env","operator":"In","values":["prod"]}]}}]}`
	templatedConfig := `{"apps":[{"namespace":"argocd","selector":` +
		`{"matchLabels":{"stage":"${{ ctx.stage }}"}}}]}`

	promoWithStep := func(config string) *kargoapi.Promotion {
		return &kargoapi.Promotion{
			ObjectMeta: metav1.ObjectMeta{Name: "fake-promotion", Namespace: project},
			Spec: kargoapi.PromotionSpec{
				Stage: stage,
				Steps: []kargoapi.PromotionStep{{
					Uses:   "argocd-update",
					Config: &apiextensionsv1.JSON{Raw: []byte(config)},
				}},
			},
			Status: kargoapi.PromotionStatus{
				Phase:       kargoapi.PromotionPhaseRunning,
				CurrentStep: 0,
			},
		}
	}

	testCases := []struct {
		name         string
		promo        *kargoapi.Promotion
		appNamespace string
		appLabels    map[string]string
		expected     bool
	}{
		{
			name:         "matchLabels match",
			promo:        promoWithStep(`{"apps":[{"namespace":"argocd","selector":{"matchLabels":{"app":"foo"}}}]}`),
			appNamespace: "argocd",
			appLabels:    map[string]string{"app": "foo"},
			expected:     true,
		},
		{
			name:         "matchLabels do not match",
			promo:        promoWithStep(`{"apps":[{"namespace":"argocd","selector":{"matchLabels":{"app":"foo"}}}]}`),
			appNamespace: "argocd",
			appLabels:    map[string]string{"app": "bar"},
			expected:     false,
		},
		{
			name:         "namespace does not match",
			promo:        promoWithStep(`{"apps":[{"namespace":"argocd","selector":{"matchLabels":{"app":"foo"}}}]}`),
			appNamespace: "other",
			appLabels:    map[string]string{"app": "foo"},
			expected:     false,
		},
		{
			name:         "matchExpressions match",
			promo:        promoWithStep(matchExprConfig),
			appNamespace: "argocd",
			appLabels:    map[string]string{"env": "prod"},
			expected:     true,
		},
		{
			name:         "templated matchLabels value resolves against context",
			promo:        promoWithStep(templatedConfig),
			appNamespace: "argocd",
			appLabels:    map[string]string{"stage": stage},
			expected:     true,
		},
		{
			name:         "name-based entry is ignored",
			promo:        promoWithStep(`{"apps":[{"namespace":"argocd","name":"some-app"}]}`),
			appNamespace: "argocd",
			appLabels:    map[string]string{"app": "foo"},
			expected:     false,
		},
	}

	scheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(scheme))

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			c := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(&kargoapi.Stage{
					ObjectMeta: metav1.ObjectMeta{Name: stage, Namespace: project},
				}).
				Build()

			require.Equal(
				t,
				testCase.expected,
				promotionSelectorsMatchApp(
					t.Context(),
					c,
					testCase.promo,
					testCase.appNamespace,
					testCase.appLabels,
				),
			)
		})
	}

	t.Run("missing Stage returns false", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).Build()
		require.False(t, promotionSelectorsMatchApp(
			t.Context(),
			c,
			promoWithStep(`{"apps":[{"namespace":"argocd","selector":{"matchLabels":{"app":"foo"}}}]}`),
			"argocd",
			map[string]string{"app": "foo"},
		))
	})
}
