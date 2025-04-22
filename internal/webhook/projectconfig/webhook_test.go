package projectconfig

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func Test_webhook_ValidateCreate(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	require.NoError(t, kargoapi.AddToScheme(scheme))

	testProjectName := "test-project"
	testNs := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: testProjectName,
			Labels: map[string]string{
				kargoapi.ProjectLabelKey: kargoapi.LabelTrueValue,
			},
		},
	}
	testNsNoLabel := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: testProjectName,
		},
	}
	testNsWrongLabel := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: testProjectName,
			Labels: map[string]string{
				kargoapi.ProjectLabelKey: "false",
			},
		},
	}

	tests := []struct {
		name          string
		projectConfig *kargoapi.ProjectConfig
		objects       []client.Object
		isDryRun      bool
		assertions    func(*testing.T, admission.Warnings, error)
	}{
		{
			name: "valid project config",
			projectConfig: &kargoapi.ProjectConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testProjectName,
					Namespace: testProjectName,
				},
				Spec: kargoapi.ProjectConfigSpec{
					PromotionPolicies: []kargoapi.PromotionPolicy{
						{Stage: "stage-1"},
						{Stage: "stage-2"},
					},
				},
			},
			objects: []client.Object{testNs},
			assertions: func(t *testing.T, warnings admission.Warnings, err error) {
				assert.Empty(t, warnings)
				assert.NoError(t, err)
			},
		},
		{
			name: "invalid metadata: name does not match namespace",
			projectConfig: &kargoapi.ProjectConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "another-name",
					Namespace: testProjectName,
				},
			},
			objects: []client.Object{testNs}, // Namespace needs to exist for later checks
			assertions: func(t *testing.T, warnings admission.Warnings, err error) {
				assert.Empty(t, warnings)
				require.Error(t, err)

				var statusErr *apierrors.StatusError
				require.True(t, errors.As(err, &statusErr))

				assert.Contains(
					t,
					statusErr.ErrStatus.Details.Causes[0].Message,
					`name "another-name" must match project name "test-project"`,
				)
				assert.Equal(t, "metadata.name", statusErr.ErrStatus.Details.Causes[0].Field)
			},
		},
		{
			name: "invalid spec: duplicate promotion policy stage",
			projectConfig: &kargoapi.ProjectConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testProjectName,
					Namespace: testProjectName,
				},
				Spec: kargoapi.ProjectConfigSpec{
					PromotionPolicies: []kargoapi.PromotionPolicy{
						{Stage: "stage-1"},
						{Stage: "stage-1"}, // Duplicate
					},
				},
			},
			objects: []client.Object{testNs}, // Namespace needs to exist for later checks
			assertions: func(t *testing.T, warnings admission.Warnings, err error) {
				assert.Empty(t, warnings)
				require.Error(t, err)

				var statusErr *apierrors.StatusError
				require.True(t, errors.As(err, &statusErr))

				assert.Equal(t, metav1.StatusReasonInvalid, statusErr.ErrStatus.Reason)
				assert.Contains(t,
					statusErr.ErrStatus.Details.Causes[0].Message,
					`multiple spec.promotionPolicies reference stage "stage-1"`,
				)
				assert.Equal(t, "spec.promotionPolicies", statusErr.ErrStatus.Details.Causes[0].Field)
			},
		},
		{
			name: "namespace does not exist",
			projectConfig: &kargoapi.ProjectConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testProjectName,
					Namespace: testProjectName,
				},
			},
			// No existing namespace object
			assertions: func(t *testing.T, warnings admission.Warnings, err error) {
				assert.Empty(t, warnings)
				require.Error(t, err)

				var statusErr *apierrors.StatusError
				require.True(t, errors.As(err, &statusErr))

				assert.Equal(t, metav1.StatusReasonInternalError, statusErr.ErrStatus.Reason)
				assert.Contains(t, statusErr.ErrStatus.Message, `error getting namespace "test-project"`)
			},
		},
		{
			name: "namespace exists but missing project label",
			projectConfig: &kargoapi.ProjectConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testProjectName,
					Namespace: testProjectName,
				},
			},
			objects: []client.Object{testNsNoLabel},
			assertions: func(t *testing.T, warnings admission.Warnings, err error) {
				assert.Empty(t, warnings)
				require.Error(t, err)

				var statusErr *apierrors.StatusError
				require.True(t, errors.As(err, &statusErr))

				assert.Equal(t, metav1.StatusReasonForbidden, statusErr.ErrStatus.Reason)
				assert.Contains(
					t,
					statusErr.ErrStatus.Message,
					fmt.Sprintf(
						`namespace %q does not belong to Kargo project (missing %q label)`,
						testProjectName, kargoapi.ProjectLabelKey,
					),
				)
			},
		},
		{
			name: "namespace exists but wrong project label value",
			projectConfig: &kargoapi.ProjectConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testProjectName,
					Namespace: testProjectName,
				},
			},
			objects: []client.Object{testNsWrongLabel},
			assertions: func(t *testing.T, warnings admission.Warnings, err error) {
				assert.Empty(t, warnings)
				require.Error(t, err)

				var statusErr *apierrors.StatusError
				require.True(t, errors.As(err, &statusErr))

				assert.Equal(t, metav1.StatusReasonForbidden, statusErr.ErrStatus.Reason)
				assert.Contains(
					t,
					statusErr.ErrStatus.Message,
					fmt.Sprintf(
						`namespace %q does not belong to Kargo project (missing %q label)`,
						testProjectName, kargoapi.ProjectLabelKey,
					),
				)
			},
		},
		{
			name: "valid project config during dry run (skips namespace check)",
			projectConfig: &kargoapi.ProjectConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testProjectName,
					Namespace: testProjectName,
				},
				Spec: kargoapi.ProjectConfigSpec{
					PromotionPolicies: []kargoapi.PromotionPolicy{
						{Stage: "stage-1"},
						{Stage: "stage-2"},
					},
				},
			},
			// No namespace object provided, but should pass due to dry run
			isDryRun: true,
			assertions: func(t *testing.T, warnings admission.Warnings, err error) {
				assert.Empty(t, warnings)
				assert.NoError(t, err)
			},
		},
		{
			name: "invalid metadata during dry run (still checked)",
			projectConfig: &kargoapi.ProjectConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "another-name",
					Namespace: testProjectName,
				},
			},
			isDryRun: true,
			assertions: func(t *testing.T, warnings admission.Warnings, err error) {
				assert.Empty(t, warnings)
				require.Error(t, err)

				var statusErr *apierrors.StatusError
				require.True(t, errors.As(err, &statusErr))

				assert.Equal(t, metav1.StatusReasonInvalid, statusErr.ErrStatus.Reason)
				assert.Contains(
					t,
					statusErr.ErrStatus.Details.Causes[0].Message,
					`name "another-name" must match project name "test-project"`,
				)
			},
		},
		{
			name: "invalid spec during dry run (still checked)",
			projectConfig: &kargoapi.ProjectConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testProjectName,
					Namespace: testProjectName,
				},
				Spec: kargoapi.ProjectConfigSpec{
					PromotionPolicies: []kargoapi.PromotionPolicy{
						{Stage: "stage-1"},
						{Stage: "stage-1"}, // Duplicate
					},
				},
			},
			isDryRun: true,
			assertions: func(t *testing.T, warnings admission.Warnings, err error) {
				assert.Empty(t, warnings)
				require.Error(t, err)

				var statusErr *apierrors.StatusError
				require.True(t, errors.As(err, &statusErr))

				assert.Equal(t, metav1.StatusReasonInvalid, statusErr.ErrStatus.Reason)
				assert.Contains(
					t,
					statusErr.ErrStatus.Details.Causes[0].Message,
					`multiple spec.promotionPolicies reference stage "stage-1"`,
				)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(tt.objects...).
				Build()

			w := &webhook{
				client: c,
			}

			ctx := admission.NewContextWithRequest(context.Background(), admission.Request{
				AdmissionRequest: admissionv1.AdmissionRequest{
					DryRun: &tt.isDryRun,
				},
			})

			warnings, err := w.ValidateCreate(ctx, tt.projectConfig)
			tt.assertions(t, warnings, err)
		})
	}
}

func Test_webhook_ValidateUpdate(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(scheme))

	testProjectName := "test-project"

	tests := []struct {
		name       string
		projectCfg *kargoapi.ProjectConfig
		assertions func(*testing.T, admission.Warnings, error)
	}{
		{
			name: "valid update",
			projectCfg: &kargoapi.ProjectConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testProjectName,
					Namespace: testProjectName,
				},
				Spec: kargoapi.ProjectConfigSpec{
					PromotionPolicies: []kargoapi.PromotionPolicy{
						{Stage: "stage-1"},
						{Stage: "stage-2"}, // Added a stage
					},
				},
			},
			assertions: func(t *testing.T, warnings admission.Warnings, err error) {
				assert.Empty(t, warnings)
				assert.NoError(t, err)
			},
		},
		{
			name: "invalid spec: duplicate promotion policy stage",
			projectCfg: &kargoapi.ProjectConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testProjectName,
					Namespace: testProjectName,
				},
				Spec: kargoapi.ProjectConfigSpec{
					PromotionPolicies: []kargoapi.PromotionPolicy{
						{Stage: "stage-duplicate"},
						{Stage: "stage-duplicate"}, // Duplicate
					},
				},
			},
			assertions: func(t *testing.T, warnings admission.Warnings, err error) {
				assert.Empty(t, warnings)

				require.Error(t, err)
				var statusErr *apierrors.StatusError
				require.True(t, errors.As(err, &statusErr))

				assert.Equal(t, metav1.StatusReasonInvalid, statusErr.ErrStatus.Reason)
				assert.Contains(
					t,
					statusErr.ErrStatus.Details.Causes[0].Message,
					`multiple spec.promotionPolicies reference stage "stage-duplicate"`,
				)
				assert.Equal(t, "spec.promotionPolicies", statusErr.ErrStatus.Details.Causes[0].Field)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &webhook{}
			warnings, err := w.ValidateUpdate(context.Background(), nil, tt.projectCfg)
			tt.assertions(t, warnings, err)
		})
	}
}
