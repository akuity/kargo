package promotionrequest

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/indexer"
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

func testOrigin() kargoapi.FreightOrigin {
	return kargoapi.FreightOrigin{
		Kind: kargoapi.FreightOriginKindWarehouse,
		Name: "fake-warehouse",
	}
}

// targetAwareStageRequestingFreight returns a target-aware Stage that requests
// Freight directly from testOrigin's Warehouse, so that origin resolution has
// candidates to select from.
func targetAwareStageRequestingFreight() *kargoapi.Stage {
	stage := targetAwareStage()
	stage.Spec.RequestedFreight = []kargoapi.FreightRequest{{
		Origin:  testOrigin(),
		Sources: kargoapi.FreightSources{Direct: true},
	}}
	return stage
}

func warehouse() *kargoapi.Warehouse {
	return &kargoapi.Warehouse{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testProject,
			Name:      testOrigin().Name,
		},
	}
}

func freight(name string, discoveredAt time.Time) *kargoapi.Freight {
	return &kargoapi.Freight{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testProject,
			Name:      name,
		},
		Origin:       testOrigin(),
		DiscoveredAt: &metav1.Time{Time: discoveredAt},
	}
}

// promotionRequestForOrigin returns a PromotionRequest as
// api.NewPromotionRequestForOrigin would construct it: an origin in place of
// Freight, and a placeholder generateName in place of a name.
func promotionRequestForOrigin(targets ...string) *kargoapi.PromotionRequest {
	promotionRequest := promotionRequest(targets...)
	promotionRequest.Name = ""
	promotionRequest.GenerateName = "promoreq-"
	promotionRequest.Spec.Freight = ""
	origin := testOrigin()
	promotionRequest.Spec.Origin = &origin
	return promotionRequest
}

func Test_webhook_Default(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	require.NoError(t, kargoapi.AddToScheme(scheme))

	contextWithOperation := func(
		t *testing.T,
		operation admissionv1.Operation,
	) context.Context {
		return admission.NewContextWithRequest(t.Context(), admission.Request{
			AdmissionRequest: admissionv1.AdmissionRequest{Operation: operation},
		})
	}

	testCases := []struct {
		name             string
		ctx              func(*testing.T) context.Context
		objects          []client.Object
		interceptors     interceptor.Funcs
		promotionRequest *kargoapi.PromotionRequest
		assertions       func(*testing.T, *kargoapi.PromotionRequest, error)
	}{
		{
			name:             "no admission request in context",
			ctx:              func(t *testing.T) context.Context { return t.Context() },
			promotionRequest: promotionRequestForOrigin(),
			assertions: func(t *testing.T, _ *kargoapi.PromotionRequest, err error) {
				require.True(t, apierrors.IsInternalError(err), "got %T: %v", err, err)
			},
		},
		{
			name: "non-create operations are ignored",
			ctx: func(t *testing.T) context.Context {
				return contextWithOperation(t, admissionv1.Update)
			},
			promotionRequest: promotionRequestForOrigin(),
			assertions: func(t *testing.T, pr *kargoapi.PromotionRequest, err error) {
				require.NoError(t, err)
				assert.Empty(t, pr.Spec.Freight)
				assert.NotNil(t, pr.Spec.Origin)
			},
		},
		{
			name: "no origin is a no-op",
			ctx: func(t *testing.T) context.Context {
				return contextWithOperation(t, admissionv1.Create)
			},
			promotionRequest: promotionRequest("us-east"),
			assertions: func(t *testing.T, pr *kargoapi.PromotionRequest, err error) {
				require.NoError(t, err)
				assert.Equal(t, "fake-freight", pr.Spec.Freight)
			},
		},
		{
			// Validation makes freight and origin mutually exclusive, but it has
			// not fired yet. Resolution is skipped and the conflict left for it.
			name: "origin alongside freight is left for validation",
			ctx: func(t *testing.T) context.Context {
				return contextWithOperation(t, admissionv1.Create)
			},
			promotionRequest: func() *kargoapi.PromotionRequest {
				pr := promotionRequest()
				origin := testOrigin()
				pr.Spec.Origin = &origin
				return pr
			}(),
			assertions: func(t *testing.T, pr *kargoapi.PromotionRequest, err error) {
				require.NoError(t, err)
				assert.Equal(t, "fake-freight", pr.Spec.Freight)
				assert.NotNil(t, pr.Spec.Origin)
			},
		},
		{
			name: "Stage does not exist",
			ctx: func(t *testing.T) context.Context {
				return contextWithOperation(t, admissionv1.Create)
			},
			promotionRequest: promotionRequestForOrigin(),
			assertions: func(t *testing.T, _ *kargoapi.PromotionRequest, err error) {
				require.True(t, apierrors.IsInvalid(err), "got %T: %v", err, err)
				assert.ErrorContains(t, err, `Stage "fake-stage" not found`)
			},
		},
		{
			name: "Stage lookup fails",
			ctx: func(t *testing.T) context.Context {
				return contextWithOperation(t, admissionv1.Create)
			},
			objects: []client.Object{targetAwareStageRequestingFreight()},
			interceptors: interceptor.Funcs{
				Get: func(
					ctx context.Context,
					c client.WithWatch,
					key client.ObjectKey,
					obj client.Object,
					opts ...client.GetOption,
				) error {
					if _, ok := obj.(*kargoapi.Stage); ok {
						return errors.New("something went wrong")
					}
					return c.Get(ctx, key, obj, opts...)
				},
			},
			promotionRequest: promotionRequestForOrigin(),
			assertions: func(t *testing.T, _ *kargoapi.PromotionRequest, err error) {
				require.True(t, apierrors.IsInternalError(err), "got %T: %v", err, err)
				assert.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "Freight listing fails",
			ctx: func(t *testing.T) context.Context {
				return contextWithOperation(t, admissionv1.Create)
			},
			objects: []client.Object{
				targetAwareStageRequestingFreight(),
				warehouse(),
			},
			interceptors: interceptor.Funcs{
				List: func(
					ctx context.Context,
					c client.WithWatch,
					list client.ObjectList,
					opts ...client.ListOption,
				) error {
					if _, ok := list.(*kargoapi.FreightList); ok {
						return errors.New("something went wrong")
					}
					return c.List(ctx, list, opts...)
				},
			},
			promotionRequest: promotionRequestForOrigin(),
			assertions: func(t *testing.T, _ *kargoapi.PromotionRequest, err error) {
				require.True(t, apierrors.IsInternalError(err), "got %T: %v", err, err)
				assert.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "no candidate for the origin",
			ctx: func(t *testing.T) context.Context {
				return contextWithOperation(t, admissionv1.Create)
			},
			objects: []client.Object{
				targetAwareStageRequestingFreight(),
				warehouse(),
			},
			promotionRequest: promotionRequestForOrigin(),
			assertions: func(t *testing.T, _ *kargoapi.PromotionRequest, err error) {
				require.True(t, apierrors.IsInvalid(err), "got %T: %v", err, err)
				assert.ErrorContains(t, err, "no auto-promotion candidate found")
				assert.ErrorContains(t, err, `"Warehouse/fake-warehouse"`)
			},
		},
		{
			name: "origin resolves to the candidate Freight",
			ctx: func(t *testing.T) context.Context {
				return contextWithOperation(t, admissionv1.Create)
			},
			objects: []client.Object{
				targetAwareStageRequestingFreight(),
				warehouse(),
				freight("older-freight", time.Now().Add(-time.Hour)),
				freight("newer-freight", time.Now()),
			},
			promotionRequest: promotionRequestForOrigin("us-east"),
			assertions: func(t *testing.T, pr *kargoapi.PromotionRequest, err error) {
				require.NoError(t, err)
				assert.Equal(t, "newer-freight", pr.Spec.Freight)
				assert.Nil(t, pr.Spec.Origin)
				// The name embeds a short hash of the resolved Freight, which the
				// caller could not have known.
				assert.True(t, strings.HasPrefix(pr.Name, "fake-stage."), pr.Name)
				assert.True(t, strings.HasSuffix(pr.Name, ".newer-f"), pr.Name)
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			w := &webhook{
				client: fake.NewClientBuilder().
					WithScheme(scheme).
					WithObjects(testCase.objects...).
					WithIndex(
						&kargoapi.Freight{},
						indexer.FreightByWarehouseField,
						indexer.FreightByWarehouse,
					).
					WithInterceptorFuncs(testCase.interceptors).
					Build(),
			}
			err := w.Default(testCase.ctx(t), testCase.promotionRequest)
			testCase.assertions(t, testCase.promotionRequest, err)
		})
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
			// Defaulting resolves a lone origin before validation fires, so an
			// origin surviving to this point means the caller set both.
			name:    "both freight and origin set",
			objects: append(projectObjects(), targetAwareStage()),
			promotionRequest: func() *kargoapi.PromotionRequest {
				pr := promotionRequest()
				origin := testOrigin()
				pr.Spec.Origin = &origin
				return pr
			}(),
			assertions: func(t *testing.T, warnings admission.Warnings, err error) {
				assert.Empty(t, warnings)
				assert.ErrorContains(
					t, err, "exactly one of spec.freight or spec.origin must be set",
				)
			},
		},
		{
			name:    "neither freight nor origin set",
			objects: append(projectObjects(), targetAwareStage()),
			promotionRequest: func() *kargoapi.PromotionRequest {
				pr := promotionRequest()
				pr.Spec.Freight = ""
				return pr
			}(),
			assertions: func(t *testing.T, warnings admission.Warnings, err error) {
				assert.Empty(t, warnings)
				assert.ErrorContains(
					t, err, "exactly one of spec.freight or spec.origin must be set",
				)
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

	t.Run("introducing origin on an update is rejected", func(t *testing.T) {
		t.Parallel()

		w := newWebhook(append(
			projectObjects(),
			targetAwareStage(),
			target("us-east"),
		)...)

		updated := promotionRequest("us-east")
		origin := testOrigin()
		updated.Spec.Origin = &origin

		warnings, err := w.ValidateUpdate(
			t.Context(),
			promotionRequest("us-east"),
			updated,
		)
		assert.Empty(t, warnings)
		assert.ErrorContains(
			t, err, "origin is resolved to freight at creation",
		)
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

// Test_webhook_failsClosed covers the paths where a lookup the webhook depends
// on fails. A transient API error must fail the admission request rather than
// silently admit a PromotionRequest nothing has actually checked.
func Test_webhook_failsClosed(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	require.NoError(t, kargoapi.AddToScheme(scheme))

	// Fresh objects per client: the builder takes ownership of what it is given
	// and stamps a resourceVersion onto it, so subtests that run in parallel
	// cannot share fixtures.
	newWebhook := func(funcs interceptor.Funcs) *webhook {
		return &webhook{
			client: fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(append(
					projectObjects(),
					targetAwareStage(),
					target("us-east"),
				)...).
				WithInterceptorFuncs(funcs).
				Build(),
		}
	}

	t.Run("Stage lookup fails", func(t *testing.T) {
		t.Parallel()

		w := newWebhook(interceptor.Funcs{
			Get: func(
				ctx context.Context,
				c client.WithWatch,
				key client.ObjectKey,
				obj client.Object,
				opts ...client.GetOption,
			) error {
				if _, ok := obj.(*kargoapi.Stage); ok {
					return errors.New("something went wrong")
				}
				return c.Get(ctx, key, obj, opts...)
			},
		})

		_, err := w.ValidateCreate(t.Context(), promotionRequest("us-east"))
		require.True(t, apierrors.IsInternalError(err), "got %T: %v", err, err)
		assert.ErrorContains(t, err, "something went wrong")
	})

	t.Run("Target lookup fails", func(t *testing.T) {
		t.Parallel()

		w := newWebhook(interceptor.Funcs{
			List: func(
				ctx context.Context,
				c client.WithWatch,
				list client.ObjectList,
				opts ...client.ListOption,
			) error {
				if _, ok := list.(*kargoapi.TargetList); ok {
					return errors.New("something went wrong")
				}
				return c.List(ctx, list, opts...)
			},
		})

		_, err := w.ValidateCreate(t.Context(), promotionRequest("us-east"))
		require.True(t, apierrors.IsInternalError(err), "got %T: %v", err, err)
		assert.ErrorContains(t, err, "something went wrong")
	})

	t.Run("Target lookup fails on update", func(t *testing.T) {
		t.Parallel()

		w := newWebhook(interceptor.Funcs{
			List: func(
				ctx context.Context,
				c client.WithWatch,
				list client.ObjectList,
				opts ...client.ListOption,
			) error {
				if _, ok := list.(*kargoapi.TargetList); ok {
					return errors.New("something went wrong")
				}
				return c.List(ctx, list, opts...)
			},
		})

		_, err := w.ValidateUpdate(
			t.Context(),
			promotionRequest("us-east"),
			promotionRequest("us-east", "us-west"),
		)
		require.True(t, apierrors.IsInternalError(err), "got %T: %v", err, err)
	})
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
