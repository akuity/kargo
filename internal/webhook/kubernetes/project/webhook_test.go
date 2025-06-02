package project

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/api"
)

func TestWebhookConfigFromEnv(t *testing.T) {
	const kargoNamespace = "test-kargo-namespace"
	t.Setenv("KARGO_NAMESPACE", kargoNamespace)
	cfg := WebhookConfigFromEnv()
	assert.Equal(t, kargoNamespace, cfg.KargoNamespace)
}

func Test_webhook_ValidateCreate(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	require.NoError(t, rbacv1.AddToScheme(scheme))
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

	tests := []struct {
		name       string
		project    *kargoapi.Project
		objects    []client.Object
		isDryRun   bool
		assertions func(*testing.T, admission.Warnings, error)
	}{
		{
			name: "valid project",
			project: &kargoapi.Project{
				ObjectMeta: metav1.ObjectMeta{
					Name: testProjectName,
				},
			},
			assertions: func(t *testing.T, warnings admission.Warnings, err error) {
				assert.Empty(t, warnings)
				assert.NoError(t, err)
			},
		},
		{
			name: "project with deprecated spec",
			project: &kargoapi.Project{
				ObjectMeta: metav1.ObjectMeta{
					Name: testProjectName,
				},
				Spec: &kargoapi.ProjectSpec{}, // nolint: staticcheck
			},
			assertions: func(t *testing.T, warnings admission.Warnings, err error) {
				assert.Len(t, warnings, 1)
				assert.Contains(t, warnings[0], "ProjectSpec is deprecated")
				assert.Contains(t, warnings[0], testProjectName)
				assert.NoError(t, err) // Creation should succeed with warnings
			},
		},
		{
			name: "namespace exists with project label",
			project: &kargoapi.Project{
				ObjectMeta: metav1.ObjectMeta{
					Name: testProjectName,
				},
			},
			objects: []client.Object{testNs},
			assertions: func(t *testing.T, warnings admission.Warnings, err error) {
				assert.Empty(t, warnings)
				assert.NoError(t, err)
			},
		},
		{
			name: "namespace exists but without project label",
			project: &kargoapi.Project{
				ObjectMeta: metav1.ObjectMeta{
					Name: testProjectName,
				},
			},
			objects: []client.Object{testNsNoLabel},
			assertions: func(t *testing.T, warnings admission.Warnings, err error) {
				assert.Empty(t, warnings)
				require.Error(t, err)

				var statusErr *apierrors.StatusError
				require.True(t, errors.As(err, &statusErr))

				assert.Equal(t, metav1.StatusReasonConflict, statusErr.ErrStatus.Reason)
				assert.Contains(t, statusErr.ErrStatus.Message, "not labeled as a Project namespace")
			},
		},
		{
			name: "dry run request",
			project: &kargoapi.Project{
				ObjectMeta: metav1.ObjectMeta{
					Name: testProjectName,
				},
			},
			isDryRun: true,
			assertions: func(t *testing.T, warnings admission.Warnings, err error) {
				assert.Empty(t, warnings)
				assert.NoError(t, err)
			},
		},
		{
			name: "dry run request with deprecated spec",
			project: &kargoapi.Project{
				ObjectMeta: metav1.ObjectMeta{
					Name: testProjectName,
				},
				Spec: &kargoapi.ProjectSpec{}, // nolint: staticcheck
			},
			isDryRun: true,
			assertions: func(t *testing.T, warnings admission.Warnings, err error) {
				assert.Len(t, warnings, 1)
				assert.Contains(t, warnings[0], "ProjectSpec is deprecated")
				assert.Contains(t, warnings[0], testProjectName)
				assert.NoError(t, err)
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
				cfg: WebhookConfig{
					KargoNamespace: "kargo-system",
				},
			}

			ctx := admission.NewContextWithRequest(context.Background(), admission.Request{
				AdmissionRequest: admissionv1.AdmissionRequest{
					DryRun: &tt.isDryRun,
				},
			})

			warnings, err := w.ValidateCreate(ctx, tt.project)
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
		oldProject *kargoapi.Project
		newProject *kargoapi.Project
		assertions func(*testing.T, admission.Warnings, error)
	}{
		{
			name: "no spec changes",
			oldProject: &kargoapi.Project{
				ObjectMeta: metav1.ObjectMeta{
					Name: testProjectName,
				},
			},
			newProject: &kargoapi.Project{
				ObjectMeta: metav1.ObjectMeta{
					Name: testProjectName,
				},
			},
			assertions: func(t *testing.T, warnings admission.Warnings, err error) {
				assert.Empty(t, warnings)
				assert.NoError(t, err)
			},
		},
		{
			name: "no change to deprecated spec",
			oldProject: &kargoapi.Project{
				ObjectMeta: metav1.ObjectMeta{
					Name: testProjectName,
				},
				Spec: &kargoapi.ProjectSpec{}, // nolint: staticcheck
			},
			newProject: &kargoapi.Project{
				ObjectMeta: metav1.ObjectMeta{
					Name: testProjectName,
				},
				Spec: &kargoapi.ProjectSpec{}, // nolint: staticcheck
			},
			assertions: func(t *testing.T, warnings admission.Warnings, err error) {
				assert.Len(t, warnings, 1)
				assert.Contains(t, warnings[0], "ProjectSpec is deprecated")
				assert.Contains(t, warnings[0], testProjectName)
				assert.NoError(t, err) // Should succeed with warnings
			},
		},
		{
			name: "changes to deprecated spec without migration annotation",
			oldProject: &kargoapi.Project{
				ObjectMeta: metav1.ObjectMeta{
					Name: testProjectName,
				},
				Spec: &kargoapi.ProjectSpec{ // nolint: staticcheck
					PromotionPolicies: []kargoapi.PromotionPolicy{
						{
							Stage:                "test-stage",
							AutoPromotionEnabled: false,
						},
					},
				},
			},
			newProject: &kargoapi.Project{
				ObjectMeta: metav1.ObjectMeta{
					Name: testProjectName,
				},
				Spec: &kargoapi.ProjectSpec{ // nolint: staticcheck
					PromotionPolicies: []kargoapi.PromotionPolicy{
						{
							Stage:                "test-stage",
							AutoPromotionEnabled: true,
						},
					},
				},
			},
			assertions: func(t *testing.T, warnings admission.Warnings, err error) {
				assert.Len(t, warnings, 1)
				assert.Contains(t, warnings[0], "ProjectSpec is deprecated")
				assert.Contains(t, warnings[0], testProjectName)
				assert.NoError(t, err) // Should succeed with warnings when no migration annotation
			},
		},
		{
			name: "changes to deprecated spec with migration annotation",
			oldProject: &kargoapi.Project{
				ObjectMeta: metav1.ObjectMeta{
					Name: testProjectName,
				},
				Spec: &kargoapi.ProjectSpec{ // nolint: staticcheck
					PromotionPolicies: []kargoapi.PromotionPolicy{
						{
							Stage:                "test-stage",
							AutoPromotionEnabled: false,
						},
					},
				},
			},
			newProject: func() *kargoapi.Project {
				p := &kargoapi.Project{
					ObjectMeta: metav1.ObjectMeta{
						Name: testProjectName,
					},
					Spec: &kargoapi.ProjectSpec{ // nolint: staticcheck
						PromotionPolicies: []kargoapi.PromotionPolicy{
							{
								Stage:                "test-stage",
								AutoPromotionEnabled: true,
							},
						},
					},
				}
				api.AddMigrationAnnotationValue(p, api.MigratedProjectSpecToProjectConfig)
				return p
			}(),
			assertions: func(t *testing.T, warnings admission.Warnings, err error) {
				assert.Empty(t, warnings) // No warnings when there's an error
				require.Error(t, err)

				var statusErr *apierrors.StatusError
				require.True(t, errors.As(err, &statusErr))

				assert.Equal(t, metav1.StatusReasonInvalid, statusErr.ErrStatus.Reason)
				assert.Contains(t, statusErr.ErrStatus.Details.Causes[0].Message, "deprecated field")
				assert.Equal(t, "spec", statusErr.ErrStatus.Details.Causes[0].Field)
			},
		},
		{
			name: "no changes to deprecated spec with migration annotation",
			oldProject: &kargoapi.Project{
				ObjectMeta: metav1.ObjectMeta{
					Name:        testProjectName,
					Annotations: make(map[string]string),
				},
				Spec: &kargoapi.ProjectSpec{ // nolint: staticcheck
					PromotionPolicies: []kargoapi.PromotionPolicy{
						{
							Stage:                "test-stage",
							AutoPromotionEnabled: false,
						},
					},
				},
			},
			newProject: func() *kargoapi.Project {
				p := &kargoapi.Project{
					ObjectMeta: metav1.ObjectMeta{
						Name: testProjectName,
					},
					Spec: &kargoapi.ProjectSpec{ // nolint: staticcheck
						PromotionPolicies: []kargoapi.PromotionPolicy{
							{
								Stage:                "test-stage",
								AutoPromotionEnabled: false,
							},
						},
					},
				}
				api.AddMigrationAnnotationValue(p, api.MigratedProjectSpecToProjectConfig)
				return p
			}(),
			assertions: func(t *testing.T, warnings admission.Warnings, err error) {
				assert.Len(t, warnings, 1)
				assert.Contains(t, warnings[0], "ProjectSpec is deprecated")
				assert.Contains(t, warnings[0], testProjectName)
				assert.NoError(t, err) // Should succeed when no changes to spec
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &webhook{}
			warnings, err := w.ValidateUpdate(context.Background(), tt.oldProject, tt.newProject)
			tt.assertions(t, warnings, err)
		})
	}
}

func Test_webhook_ValidateDelete(t *testing.T) {
	w := &webhook{}

	project := &kargoapi.Project{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-project",
		},
	}

	warnings, err := w.ValidateDelete(context.Background(), project)
	assert.Empty(t, warnings)
	assert.NoError(t, err)
}

func Test_webhook_ensureNamespace(t *testing.T) {
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

	tests := []struct {
		name       string
		project    *kargoapi.Project
		objects    []client.Object
		assertions func(*testing.T, error)
	}{
		{
			name: "namespace does not exist, should create",
			project: &kargoapi.Project{
				ObjectMeta: metav1.ObjectMeta{
					Name: testProjectName,
				},
			},
			objects: []client.Object{}, // No namespace exists yet
			assertions: func(t *testing.T, err error) {
				assert.NoError(t, err)
			},
		},
		{
			name: "namespace exists with project label, no conflict",
			project: &kargoapi.Project{
				ObjectMeta: metav1.ObjectMeta{
					Name: testProjectName,
				},
			},
			objects: []client.Object{testNs},
			assertions: func(t *testing.T, err error) {
				assert.NoError(t, err)
			},
		},
		{
			name: "namespace exists without project label, conflict",
			project: &kargoapi.Project{
				ObjectMeta: metav1.ObjectMeta{
					Name: testProjectName,
				},
			},
			objects: []client.Object{testNsNoLabel},
			assertions: func(t *testing.T, err error) {
				require.Error(t, err)
				var statusErr *apierrors.StatusError
				require.True(t, errors.As(err, &statusErr))
				assert.Equal(t, metav1.StatusReasonConflict, statusErr.ErrStatus.Reason)
				assert.Contains(t, statusErr.ErrStatus.Message, "not labeled as a Project namespace")
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

			err := w.ensureNamespace(context.Background(), tt.project)
			tt.assertions(t, err)
		})
	}
}

func Test_webhook_ensureProjectAdminPermissions(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, rbacv1.AddToScheme(scheme))
	require.NoError(t, kargoapi.AddToScheme(scheme))

	testProjectName := "test-project"
	kargoNamespace := "kargo-system"

	roleBindingName := "kargo-project-admin"
	existingRoleBinding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      roleBindingName,
			Namespace: testProjectName,
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "ClusterRole",
			Name:     "kargo-project-admin",
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      "kargo-api",
				Namespace: kargoNamespace,
			},
			{
				Kind:      "ServiceAccount",
				Name:      "kargo-admin",
				Namespace: kargoNamespace,
			},
		},
	}

	tests := []struct {
		name       string
		project    *kargoapi.Project
		objects    []client.Object
		assertions func(*testing.T, error, client.Client)
	}{
		{
			name: "role binding does not exist, should create",
			project: &kargoapi.Project{
				ObjectMeta: metav1.ObjectMeta{
					Name: testProjectName,
				},
			},
			objects: []client.Object{},
			assertions: func(t *testing.T, err error, c client.Client) {
				assert.NoError(t, err)

				// Check role binding creation
				rb := &rbacv1.RoleBinding{}
				err = c.Get(context.Background(), client.ObjectKey{
					Name:      roleBindingName,
					Namespace: testProjectName,
				}, rb)
				assert.NoError(t, err)
				assert.Len(t, rb.Subjects, 2)
				assert.Equal(t, "kargo-api", rb.Subjects[0].Name)
				assert.Equal(t, "kargo-admin", rb.Subjects[1].Name)
				assert.Equal(t, kargoNamespace, rb.Subjects[0].Namespace)
			},
		},
		{
			name: "role binding already exists",
			project: &kargoapi.Project{
				ObjectMeta: metav1.ObjectMeta{
					Name: testProjectName,
				},
			},
			objects: []client.Object{existingRoleBinding},
			assertions: func(t *testing.T, err error, _ client.Client) {
				assert.NoError(t, err)
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
				cfg: WebhookConfig{
					KargoNamespace: kargoNamespace,
				},
			}

			err := w.ensureProjectAdminPermissions(context.Background(), tt.project)
			tt.assertions(t, err, c)
		})
	}
}
