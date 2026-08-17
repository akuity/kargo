package promotionrequest

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

const testProject = "fake-project"

func projectObjects() []client.Object {
	return []client.Object{
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: testProject,
				Labels: map[string]string{
					kargoapi.LabelKeyProject: kargoapi.LabelValueTrue,
				},
			},
		},
		&kargoapi.Project{ObjectMeta: metav1.ObjectMeta{Name: testProject}},
	}
}

func targetAwareStage() *kargoapi.Stage {
	return &kargoapi.Stage{
		ObjectMeta: metav1.ObjectMeta{Namespace: testProject, Name: "fake-stage"},
		Spec: kargoapi.StageSpec{
			Targets: &kargoapi.StageTargets{
				Selectors: []metav1.LabelSelector{{
					MatchLabels: map[string]string{"region": "us"},
				}},
			},
		},
	}
}

func target(name string) *kargoapi.Target {
	return &kargoapi.Target{
		ObjectMeta: metav1.ObjectMeta{Namespace: testProject, Name: name},
	}
}

func promotionRequest(targets ...string) *kargoapi.PromotionRequest {
	specTargets := make([]kargoapi.PromotionRequestTarget, len(targets))
	for i, name := range targets {
		specTargets[i] = kargoapi.PromotionRequestTarget{Name: name}
	}
	return &kargoapi.PromotionRequest{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testProject,
			Name:      "fake-stage.01jexample.abcdef1",
		},
		Spec: kargoapi.PromotionRequestSpec{
			Stage:   "fake-stage",
			Freight: "fake-freight",
			Targets: specTargets,
		},
	}
}

func Test_webhook_ValidateCreate(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	require.NoError(t, kargoapi.AddToScheme(scheme))

	tests := []struct {
		name             string
		objects          []client.Object
		promotionRequest *kargoapi.PromotionRequest
		assertions       func(*testing.T, admission.Warnings, error)
	}{
		{
			name: "project does not exist",
			objects: []client.Object{
				&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: testProject}},
			},
			promotionRequest: promotionRequest(),
			assertions: func(t *testing.T, warnings admission.Warnings, err error) {
				assert.Empty(t, warnings)
				assert.ErrorContains(t, err, `namespace "fake-project" is not a project`)
			},
		},
		{
			name:    "Stage does not exist",
			objects: append(projectObjects(), target("us-east")),
			// No Stage, so the request names one that cannot govern anything.
			promotionRequest: promotionRequest("us-east"),
			assertions: func(t *testing.T, warnings admission.Warnings, err error) {
				assert.Empty(t, warnings)
				assert.ErrorContains(t, err, `Stage "fake-stage" not found`)
			},
		},
		{
			name: "Stage is not target-aware",
			objects: append(
				projectObjects(),
				&kargoapi.Stage{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProject,
						Name:      "fake-stage",
					},
				},
				target("us-east"),
			),
			promotionRequest: promotionRequest("us-east"),
			assertions: func(t *testing.T, warnings admission.Warnings, err error) {
				assert.Empty(t, warnings)
				assert.ErrorContains(t, err, "does not select Targets")
			},
		},
		{
			// The guarantee spec.targets' doc comment promises, which the schema
			// deliberately does not enforce.
			name: "duplicate Target names",
			objects: append(
				projectObjects(),
				targetAwareStage(),
				target("us-east"),
			),
			promotionRequest: promotionRequest("us-east", "us-east"),
			assertions: func(t *testing.T, warnings admission.Warnings, err error) {
				assert.Empty(t, warnings)
				assert.ErrorContains(t, err, "Duplicate value")
				assert.ErrorContains(t, err, "us-east")
			},
		},
		{
			name: "Target does not exist",
			objects: append(
				projectObjects(),
				targetAwareStage(),
				target("us-east"),
			),
			promotionRequest: promotionRequest("us-east", "nonexistent"),
			assertions: func(t *testing.T, warnings admission.Warnings, err error) {
				assert.Empty(t, warnings)
				assert.ErrorContains(t, err, `Target "nonexistent" not found`)
			},
		},
		{
			name: "Target in another Project does not count",
			objects: append(
				projectObjects(),
				targetAwareStage(),
				&kargoapi.Target{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "other-project",
						Name:      "us-east",
					},
				},
			),
			promotionRequest: promotionRequest("us-east"),
			assertions: func(t *testing.T, warnings admission.Warnings, err error) {
				assert.Empty(t, warnings)
				assert.ErrorContains(t, err, `Target "us-east" not found`)
			},
		},
		{
			name: "valid",
			objects: append(
				projectObjects(),
				targetAwareStage(),
				target("us-east"),
				target("us-west"),
			),
			promotionRequest: promotionRequest("us-east", "us-west"),
			assertions: func(t *testing.T, warnings admission.Warnings, err error) {
				assert.Empty(t, warnings)
				assert.NoError(t, err)
			},
		},
		{
			// A Stage may govern no Targets at the moment the request is
			// created. That is recorded, not rejected.
			name:             "empty Targets list is valid",
			objects:          append(projectObjects(), targetAwareStage()),
			promotionRequest: promotionRequest(),
			assertions: func(t *testing.T, warnings admission.Warnings, err error) {
				assert.Empty(t, warnings)
				assert.NoError(t, err)
			},
		},
		{
			// spec.targets is a snapshot of what the Stage governed at creation.
			// A Target whose labels have since stopped matching must not
			// retroactively invalidate a request already in flight.
			name: "Target no longer matching the Stage's selectors is still valid",
			objects: append(
				projectObjects(),
				targetAwareStage(),
				&kargoapi.Target{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProject,
						Name:      "us-east",
						Labels:    map[string]string{"region": "eu"},
					},
				},
			),
			promotionRequest: promotionRequest("us-east"),
			assertions: func(t *testing.T, warnings admission.Warnings, err error) {
				assert.Empty(t, warnings)
				assert.NoError(t, err)
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			w := &webhook{
				client: fake.NewClientBuilder().
					WithScheme(scheme).
					WithObjects(testCase.objects...).
					Build(),
			}
			warnings, err := w.ValidateCreate(t.Context(), testCase.promotionRequest)
			testCase.assertions(t, warnings, err)
		})
	}
}

func Test_webhook_ValidateUpdate(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	require.NoError(t, kargoapi.AddToScheme(scheme))

	newWebhook := func(objects ...client.Object) *webhook {
		return &webhook{
			client: fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(objects...).
				Build(),
		}
	}

	t.Run("adding a Target in flight is allowed", func(t *testing.T) {
		t.Parallel()

		w := newWebhook(append(
			projectObjects(),
			targetAwareStage(),
			target("us-east"),
			target("us-west"),
		)...)

		warnings, err := w.ValidateUpdate(
			t.Context(),
			promotionRequest("us-east"),
			promotionRequest("us-east", "us-west"),
		)
		assert.Empty(t, warnings)
		assert.NoError(t, err)
	})

	t.Run("adding a duplicate Target is rejected", func(t *testing.T) {
		t.Parallel()

		w := newWebhook(append(
			projectObjects(),
			targetAwareStage(),
			target("us-east"),
		)...)

		warnings, err := w.ValidateUpdate(
			t.Context(),
			promotionRequest("us-east"),
			promotionRequest("us-east", "us-east"),
		)
		assert.Empty(t, warnings)
		assert.ErrorContains(t, err, "Duplicate value")
	})

	t.Run("adding a nonexistent Target is rejected", func(t *testing.T) {
		t.Parallel()

		w := newWebhook(append(
			projectObjects(),
			targetAwareStage(),
			target("us-east"),
		)...)

		warnings, err := w.ValidateUpdate(
			t.Context(),
			promotionRequest("us-east"),
			promotionRequest("us-east", "nonexistent"),
		)
		assert.Empty(t, warnings)
		assert.ErrorContains(t, err, `Target "nonexistent" not found`)
	})
}

func Test_webhook_ValidateDelete(t *testing.T) {
	t.Parallel()

	w := &webhook{}
	warnings, err := w.ValidateDelete(t.Context(), promotionRequest())
	assert.Empty(t, warnings)
	assert.NoError(t, err)
}

func Test_validateTargetsUnique(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		targets    []string
		assertions func(*testing.T, field.ErrorList)
	}{
		{
			name:    "empty",
			targets: nil,
			assertions: func(t *testing.T, errs field.ErrorList) {
				assert.Empty(t, errs)
			},
		},
		{
			name:    "all unique",
			targets: []string{"us-east", "us-west", "eu-west"},
			assertions: func(t *testing.T, errs field.ErrorList) {
				assert.Empty(t, errs)
			},
		},
		{
			name:    "one duplicate reports once, at the later index",
			targets: []string{"us-east", "us-west", "us-east"},
			assertions: func(t *testing.T, errs field.ErrorList) {
				require.Len(t, errs, 1)
				assert.Equal(t, field.ErrorTypeDuplicate, errs[0].Type)
				assert.Equal(t, "targets[2].name", errs[0].Field)
			},
		},
		{
			name:    "a name repeated three times reports twice",
			targets: []string{"us-east", "us-east", "us-east"},
			assertions: func(t *testing.T, errs field.ErrorList) {
				require.Len(t, errs, 2)
				assert.Equal(t, "targets[1].name", errs[0].Field)
				assert.Equal(t, "targets[2].name", errs[1].Field)
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			targets := make([]kargoapi.PromotionRequestTarget, len(testCase.targets))
			for i, name := range testCase.targets {
				targets[i] = kargoapi.PromotionRequestTarget{Name: name}
			}
			testCase.assertions(
				t,
				validateTargetsUnique(field.NewPath("targets"), targets),
			)
		})
	}
}
