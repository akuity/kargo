package builtin

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/promotion"
	"github.com/akuity/kargo/pkg/x/promotion/runner/builtin"
)

func Test_authorizeArgoCDAppAccess(t *testing.T) {
	testCases := []struct {
		name      string
		stepCtx   *promotion.StepContext
		appMeta   metav1.ObjectMeta
		expectErr bool
		errSubstr string
	}{
		{
			name: "no annotation",
			stepCtx: &promotion.StepContext{
				Project: "my-project",
				Stage:   "my-stage",
			},
			appMeta: metav1.ObjectMeta{
				Name:      "my-app",
				Namespace: "argocd",
			},
			expectErr: true,
			errSubstr: "does not permit access",
		},
		{
			name: "unparseable annotation",
			stepCtx: &promotion.StepContext{
				Project: "my-project",
				Stage:   "my-stage",
			},
			appMeta: metav1.ObjectMeta{
				Name:      "my-app",
				Namespace: "argocd",
				Annotations: map[string]string{
					kargoapi.AnnotationKeyAuthorizedStage: "bad-value",
				},
			},
			expectErr: true,
			errSubstr: "unable to parse",
		},
		{
			name: "glob pattern in annotation",
			stepCtx: &promotion.StepContext{
				Project: "my-project",
				Stage:   "my-stage",
			},
			appMeta: metav1.ObjectMeta{
				Name:      "my-app",
				Namespace: "argocd",
				Annotations: map[string]string{
					kargoapi.AnnotationKeyAuthorizedStage: "my-project:*",
				},
			},
			expectErr: true,
			errSubstr: "deprecated glob",
		},
		{
			name: "wrong project",
			stepCtx: &promotion.StepContext{
				Project: "my-project",
				Stage:   "my-stage",
			},
			appMeta: metav1.ObjectMeta{
				Name:      "my-app",
				Namespace: "argocd",
				Annotations: map[string]string{
					kargoapi.AnnotationKeyAuthorizedStage: "other-project:my-stage",
				},
			},
			expectErr: true,
			errSubstr: "does not permit access",
		},
		{
			name: "wrong stage",
			stepCtx: &promotion.StepContext{
				Project: "my-project",
				Stage:   "my-stage",
			},
			appMeta: metav1.ObjectMeta{
				Name:      "my-app",
				Namespace: "argocd",
				Annotations: map[string]string{
					kargoapi.AnnotationKeyAuthorizedStage: "my-project:other-stage",
				},
			},
			expectErr: true,
			errSubstr: "does not permit access",
		},
		{
			name: "authorized",
			stepCtx: &promotion.StepContext{
				Project: "my-project",
				Stage:   "my-stage",
			},
			appMeta: metav1.ObjectMeta{
				Name:      "my-app",
				Namespace: "argocd",
				Annotations: map[string]string{
					kargoapi.AnnotationKeyAuthorizedStage: "my-project:my-stage",
				},
			},
			expectErr: false,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := authorizeArgoCDAppAccess(tc.stepCtx, tc.appMeta)
			if tc.expectErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errSubstr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func Test_buildArgoCDAppLabelSelector(t *testing.T) {
	testCases := []struct {
		name      string
		selector  *builtin.ArgoCDAppSelector
		expectErr bool
	}{
		{
			name:      "empty selector",
			selector:  &builtin.ArgoCDAppSelector{},
			expectErr: true,
		},
		{
			name: "valid matchLabels",
			selector: &builtin.ArgoCDAppSelector{
				MatchLabels: map[string]string{"app": "test"},
			},
			expectErr: false,
		},
		{
			name: "valid matchExpressions",
			selector: &builtin.ArgoCDAppSelector{
				MatchExpressions: []builtin.MatchExpression{
					{Key: "app", Operator: builtin.In, Values: []string{"a", "b"}},
				},
			},
			expectErr: false,
		},
		{
			name: "invalid operator",
			selector: &builtin.ArgoCDAppSelector{
				MatchExpressions: []builtin.MatchExpression{
					{Key: "app", Operator: "Invalid"},
				},
			},
			expectErr: true,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			sel, err := buildArgoCDAppLabelSelector(tc.selector)
			if tc.expectErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.NotNil(t, sel)
			}
		})
	}
}
