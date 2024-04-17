package stage

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	admissionv1 "k8s.io/api/admission/v1"
	authnv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	libWebhook "github.com/akuity/kargo/internal/webhook"
)

func TestNewWebhook(t *testing.T) {
	kubeClient := fake.NewClientBuilder().Build()
	w := newWebhook(
		libWebhook.Config{},
		kubeClient,
	)
	// Assert that all overridable behaviors were initialized to a default:
	require.NotNil(t, w.admissionRequestFromContextFn)
	require.NotNil(t, w.validateProjectFn)
	require.NotNil(t, w.validateCreateOrUpdateFn)
	require.NotNil(t, w.validateSpecFn)
	require.NotNil(t, w.isRequestFromKargoControlplaneFn)
}

func TestDefault(t *testing.T) {
	const testShardName = "fake-shard"
	testCases := []struct {
		name       string
		webhook    *webhook
		req        admission.Request
		stage      *kargoapi.Stage
		assertions func(*testing.T, *kargoapi.Stage, error)
	}{
		{
			name: "shard stays default when not specified at all",
			webhook: &webhook{
				admissionRequestFromContextFn: admission.RequestFromContext,
				isRequestFromKargoControlplaneFn: func(admission.Request) bool {
					return true
				},
			},
			req: admission.Request{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Operation: admissionv1.Create,
				},
			},
			stage: &kargoapi.Stage{
				Spec: &kargoapi.StageSpec{},
			},
			assertions: func(t *testing.T, stage *kargoapi.Stage, err error) {
				require.NoError(t, err)
				require.Empty(t, stage.Labels)
				require.Empty(t, stage.Spec.Shard)
			},
		},
		{
			name: "sync shard label to non-empty shard field",
			webhook: &webhook{
				admissionRequestFromContextFn: admission.RequestFromContext,
				isRequestFromKargoControlplaneFn: func(admission.Request) bool {
					return true
				},
			},
			req: admission.Request{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Operation: admissionv1.Create,
				},
			},
			stage: &kargoapi.Stage{
				Spec: &kargoapi.StageSpec{
					Shard: testShardName,
				},
			},
			assertions: func(t *testing.T, stage *kargoapi.Stage, err error) {
				require.NoError(t, err)
				require.Equal(t, testShardName, stage.Spec.Shard)
				require.Equal(t, testShardName, stage.Labels[kargoapi.ShardLabelKey])
			},
		},
		{
			name: "sync shard label to empty shard field",
			webhook: &webhook{
				admissionRequestFromContextFn: admission.RequestFromContext,
				isRequestFromKargoControlplaneFn: func(admission.Request) bool {
					return true
				},
			},
			req: admission.Request{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Operation: admissionv1.Create,
				},
			},
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						kargoapi.ShardLabelKey: testShardName,
					},
				},
				Spec: &kargoapi.StageSpec{},
			},
			assertions: func(t *testing.T, stage *kargoapi.Stage, err error) {
				require.NoError(t, err)
				require.Empty(t, stage.Spec.Shard)
				_, ok := stage.Labels[kargoapi.ShardLabelKey]
				require.False(t, ok)
			},
		},
		{
			name: "set reverify actor when request doesn't come from kargo controlplane",
			webhook: &webhook{
				admissionRequestFromContextFn: admission.RequestFromContext,
				isRequestFromKargoControlplaneFn: func(admission.Request) bool {
					return false
				},
			},
			req: admission.Request{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Operation: admissionv1.Update,
					UserInfo: authnv1.UserInfo{
						Username: "real-user",
					},
					OldObject: runtime.RawExtension{
						Object: &kargoapi.Stage{},
					},
				},
			},
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						kargoapi.AnnotationKeyReverify: "fake-id",
					},
				},
				Spec: &kargoapi.StageSpec{},
			},
			assertions: func(t *testing.T, stage *kargoapi.Stage, err error) {
				require.NoError(t, err)
				require.Contains(t, stage.Annotations, kargoapi.AnnotationKeyReverify)
				rr, ok := kargoapi.ReverifyAnnotationValue(stage.Annotations)
				require.True(t, ok)
				require.Equal(t, &kargoapi.ReverificationRequest{
					ID: "fake-id",
					Actor: kargoapi.FormatEventKubernetesUserActor(authnv1.UserInfo{
						Username: "real-user",
					}),
					ControlPlane: false,
				}, rr)
			},
		},
		{
			name: "overwrite with admission request user info if reverify actor annotation exists",
			webhook: &webhook{
				admissionRequestFromContextFn: admission.RequestFromContext,
				isRequestFromKargoControlplaneFn: func(admission.Request) bool {
					return false
				},
			},
			req: admission.Request{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Operation: admissionv1.Update,
					UserInfo: authnv1.UserInfo{
						Username: "real-user",
					},
					OldObject: runtime.RawExtension{
						Object: &kargoapi.Stage{},
					},
				},
			},
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						kargoapi.AnnotationKeyReverify: (&kargoapi.ReverificationRequest{
							ID:    "fake-id",
							Actor: "fake-user",
						}).String(),
					},
				},
				Spec: &kargoapi.StageSpec{},
			},
			assertions: func(t *testing.T, stage *kargoapi.Stage, err error) {
				require.NoError(t, err)
				require.Contains(t, stage.Annotations, kargoapi.AnnotationKeyReverify)
				rr, ok := kargoapi.ReverifyAnnotationValue(stage.Annotations)
				require.True(t, ok)
				require.Equal(t, &kargoapi.ReverificationRequest{
					ID: "fake-id",
					Actor: kargoapi.FormatEventKubernetesUserActor(authnv1.UserInfo{
						Username: "real-user",
					}),
					ControlPlane: false,
				}, rr)
			},
		},
		{
			name: "do not overwrite reverify actor when request comes from controlplane",
			webhook: &webhook{
				admissionRequestFromContextFn: admission.RequestFromContext,
				isRequestFromKargoControlplaneFn: func(admission.Request) bool {
					return true
				},
			},
			req: admission.Request{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Operation: admissionv1.Update,
					UserInfo: authnv1.UserInfo{
						Username: "control-plane-user",
					},
					OldObject: runtime.RawExtension{
						Object: &kargoapi.Stage{},
					},
				},
			},
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						kargoapi.AnnotationKeyReverify: (&kargoapi.ReverificationRequest{
							ID:    "fake-id",
							Actor: kargoapi.EventActorAdmin,
						}).String(),
					},
				},
				Spec: &kargoapi.StageSpec{},
			},
			assertions: func(t *testing.T, stage *kargoapi.Stage, err error) {
				require.NoError(t, err)
				require.Contains(t, stage.Annotations, kargoapi.AnnotationKeyReverify)
				rr, ok := kargoapi.ReverifyAnnotationValue(stage.Annotations)
				require.True(t, ok)
				require.Equal(t, &kargoapi.ReverificationRequest{
					ID:           "fake-id",
					Actor:        kargoapi.EventActorAdmin,
					ControlPlane: true,
				}, rr)
			},
		},
	}
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ctx := admission.NewContextWithRequest(
				context.Background(),
				tc.req,
			)
			tc.assertions(
				t,
				tc.stage,
				tc.webhook.Default(ctx, tc.stage),
			)
		})
	}
}

func TestValidateCreate(t *testing.T) {
	testCases := []struct {
		name       string
		webhook    *webhook
		assertions func(*testing.T, error)
	}{
		{
			name: "error validating project",
			webhook: &webhook{
				validateProjectFn: func(
					context.Context,
					client.Client,
					schema.GroupKind,
					client.Object,
				) error {
					return errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, err error) {
				require.Error(t, err)
				require.Equal(t, "something went wrong", err.Error())
			},
		},
		{
			name: "error validating stage",
			webhook: &webhook{
				validateProjectFn: func(
					context.Context,
					client.Client,
					schema.GroupKind,
					client.Object,
				) error {
					return nil
				},
				validateCreateOrUpdateFn: func(
					*kargoapi.Stage,
				) (admission.Warnings, error) {
					return nil, errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, err error) {
				require.Error(t, err)
				require.Equal(t, "something went wrong", err.Error())
			},
		},
		{
			name: "success",
			webhook: &webhook{
				validateProjectFn: func(
					context.Context,
					client.Client,
					schema.GroupKind,
					client.Object,
				) error {
					return nil
				},
				validateCreateOrUpdateFn: func(
					*kargoapi.Stage,
				) (admission.Warnings, error) {
					return nil, nil
				},
			},
			assertions: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := testCase.webhook.ValidateCreate(
				context.Background(),
				&kargoapi.Stage{},
			)
			testCase.assertions(t, err)
		})
	}
}

func TestValidateUpdate(t *testing.T) {
	testCases := []struct {
		name       string
		webhook    *webhook
		assertions func(*testing.T, error)
	}{
		{
			name: "error validating stage",
			webhook: &webhook{
				validateCreateOrUpdateFn: func(
					*kargoapi.Stage,
				) (admission.Warnings, error) {
					return nil, errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, err error) {
				require.Error(t, err)
				require.Equal(t, "something went wrong", err.Error())
			},
		},
		{
			name: "success",
			webhook: &webhook{
				validateCreateOrUpdateFn: func(
					*kargoapi.Stage,
				) (admission.Warnings, error) {
					return nil, nil
				},
			},
			assertions: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := testCase.webhook.ValidateUpdate(
				context.Background(),
				nil,
				&kargoapi.Stage{},
			)
			testCase.assertions(t, err)
		})
	}
}

func TestValidateDelete(t *testing.T) {
	w := &webhook{}
	_, err := w.ValidateDelete(context.Background(), nil)
	require.NoError(t, err)
}

func TestValidateCreateOrUpdate(t *testing.T) {
	testCases := []struct {
		name       string
		webhook    *webhook
		assertions func(*testing.T, error)
	}{
		{
			name: "error validating spec",
			webhook: &webhook{
				validateSpecFn: func(
					*field.Path,
					*kargoapi.StageSpec,
				) field.ErrorList {
					return field.ErrorList{{}}
				},
			},
			assertions: func(t *testing.T, err error) {
				require.Error(t, err)
			},
		},
		{
			name: "success",
			webhook: &webhook{
				validateSpecFn: func(
					*field.Path,
					*kargoapi.StageSpec,
				) field.ErrorList {
					return nil
				},
			},
			assertions: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := testCase.webhook.validateCreateOrUpdate(&kargoapi.Stage{})
			testCase.assertions(t, err)
		})
	}
}

func TestValidateSpec(t *testing.T) {
	testCases := []struct {
		name       string
		spec       *kargoapi.StageSpec
		assertions func(*testing.T, *kargoapi.StageSpec, field.ErrorList)
	}{
		{
			name: "nil",
			assertions: func(t *testing.T, _ *kargoapi.StageSpec, errs field.ErrorList) {
				require.Nil(t, errs)
			},
		},

		{
			name: "invalid",
			spec: &kargoapi.StageSpec{
				// Has two conflicting types of subs...
				Subscriptions: &kargoapi.Subscriptions{
					Warehouse: "test-warehouse",
					UpstreamStages: []kargoapi.StageSubscription{
						{},
					},
				},
				// Doesn't actually define any mechanisms...
				PromotionMechanisms: &kargoapi.PromotionMechanisms{},
			},
			assertions: func(t *testing.T, spec *kargoapi.StageSpec, errs field.ErrorList) {
				// We really want to see that all underlying errors have been bubbled up
				// to this level and been aggregated.
				require.Equal(
					t,
					field.ErrorList{
						{
							Type:     field.ErrorTypeInvalid,
							Field:    "spec.subscriptions",
							BadValue: spec.Subscriptions,
							Detail: "exactly one of spec.subscriptions.warehouse or " +
								"spec.subscriptions.upstreamStages must be defined",
						},
						{
							Type:     field.ErrorTypeInvalid,
							Field:    "spec.promotionMechanisms",
							BadValue: spec.PromotionMechanisms,
							Detail: "at least one of " +
								"spec.promotionMechanisms.gitRepoUpdates or " +
								"spec.promotionMechanisms.argoCDAppUpdates must be non-empty",
						},
					},
					errs,
				)
			},
		},

		{
			name: "valid",
			spec: &kargoapi.StageSpec{
				// Nil subs and promo mechanisms are caught by declarative validation,
				// so for the purposes of this test, leaving those completely undefined
				// should surface no errors.
			},
			assertions: func(t *testing.T, _ *kargoapi.StageSpec, errs field.ErrorList) {
				require.Nil(t, errs)
			},
		},
	}
	w := &webhook{}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				t,
				testCase.spec,
				w.validateSpec(
					field.NewPath("spec"),
					testCase.spec,
				),
			)
		})
	}
}

func TestValidateSubs(t *testing.T) {
	testCases := []struct {
		name       string
		subs       *kargoapi.Subscriptions
		assertions func(*testing.T, *kargoapi.Subscriptions, field.ErrorList)
	}{
		{
			name: "nil",
			assertions: func(t *testing.T, _ *kargoapi.Subscriptions, errs field.ErrorList) {
				require.Nil(t, errs)
			},
		},

		{
			name: "no subscriptions",
			subs: &kargoapi.Subscriptions{},
			assertions: func(t *testing.T, subs *kargoapi.Subscriptions, errs field.ErrorList) {
				require.Equal(
					t,
					field.ErrorList{
						{
							Type:     field.ErrorTypeInvalid,
							Field:    "subscriptions",
							BadValue: subs,
							Detail: "exactly one of subscriptions.warehouse or " +
								"subscriptions.upstreamStages must be defined",
						},
					},
					errs,
				)
			},
		},

		{
			name: "has warehouse sub and Stage subs", // Should be "one of"
			subs: &kargoapi.Subscriptions{
				Warehouse: "test-warehouse",
				UpstreamStages: []kargoapi.StageSubscription{
					{},
				},
			},
			assertions: func(t *testing.T, subs *kargoapi.Subscriptions, errs field.ErrorList) {
				require.Equal(
					t,
					field.ErrorList{
						{
							Type:     field.ErrorTypeInvalid,
							Field:    "subscriptions",
							BadValue: subs,
							Detail: "exactly one of subscriptions.warehouse or " +
								"subscriptions.upstreamStages must be defined",
						},
					},
					errs,
				)
			},
		},

		{
			name: "success",
			subs: &kargoapi.Subscriptions{
				Warehouse: "test-warehouse",
			},
			assertions: func(t *testing.T, _ *kargoapi.Subscriptions, errs field.ErrorList) {
				require.Nil(t, errs)
			},
		},
	}
	w := &webhook{}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				t,
				testCase.subs,
				w.validateSubs(
					field.NewPath("subscriptions"),
					testCase.subs,
				),
			)
		})
	}
}

func TestValidatePromotionMechanisms(t *testing.T) {
	testCases := []struct {
		name       string
		promoMechs *kargoapi.PromotionMechanisms
		assertions func(*testing.T, *kargoapi.PromotionMechanisms, field.ErrorList)
	}{
		{
			name: "nil",
			assertions: func(t *testing.T, _ *kargoapi.PromotionMechanisms, errs field.ErrorList) {
				require.Nil(t, errs)
			},
		},

		{
			name: "invalid",
			// Does not define any mechanisms
			promoMechs: &kargoapi.PromotionMechanisms{},
			assertions: func(
				t *testing.T,
				promoMechs *kargoapi.PromotionMechanisms,
				errs field.ErrorList,
			) {
				require.NotNil(t, errs)
				require.Equal(
					t,
					field.ErrorList{
						{
							Type:     field.ErrorTypeInvalid,
							Field:    "promotionMechanisms",
							BadValue: promoMechs,
							Detail: "at least one of promotionMechanisms.gitRepoUpdates or " +
								"promotionMechanisms.argoCDAppUpdates must be non-empty",
						},
					},
					errs,
				)
			},
		},

		{
			name: "valid",
			promoMechs: &kargoapi.PromotionMechanisms{
				GitRepoUpdates: []kargoapi.GitRepoUpdate{
					{
						Kustomize: &kargoapi.KustomizePromotionMechanism{},
					},
				},
			},
			assertions: func(t *testing.T, _ *kargoapi.PromotionMechanisms, errs field.ErrorList) {
				require.Nil(t, errs)
			},
		},
	}
	w := &webhook{}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				t,
				testCase.promoMechs,
				w.validatePromotionMechanisms(
					field.NewPath("promotionMechanisms"),
					testCase.promoMechs,
				),
			)
		})
	}
}

func TestValidateGitRepoUpdates(t *testing.T) {
	testCases := []struct {
		name       string
		update     kargoapi.GitRepoUpdate
		assertions func(*testing.T, kargoapi.GitRepoUpdate, field.ErrorList)
	}{
		{
			name: "more than one config management tool specified",
			update: kargoapi.GitRepoUpdate{
				Render:    &kargoapi.KargoRenderPromotionMechanism{},
				Kustomize: &kargoapi.KustomizePromotionMechanism{},
				Helm:      &kargoapi.HelmPromotionMechanism{},
			},
			assertions: func(t *testing.T, update kargoapi.GitRepoUpdate, errs field.ErrorList) {
				require.Equal(
					t,
					field.ErrorList{
						{
							Type:     field.ErrorTypeInvalid,
							Field:    "gitRepoUpdates[0]",
							BadValue: update,
							Detail: "no more than one of gitRepoUpdates[0].render, or " +
								"gitRepoUpdates[0].kustomize, or gitRepoUpdates[0].helm " +
								"may be defined",
						},
					},
					errs,
				)
			},
		},

		{
			name: "valid",
			update: kargoapi.GitRepoUpdate{
				Kustomize: &kargoapi.KustomizePromotionMechanism{},
			},
			assertions: func(t *testing.T, _ kargoapi.GitRepoUpdate, errs field.ErrorList) {
				require.Nil(t, errs)
			},
		},
	}
	w := &webhook{}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				t,
				testCase.update,
				w.validateGitRepoUpdates(
					field.NewPath("gitRepoUpdates"),
					[]kargoapi.GitRepoUpdate{
						testCase.update,
					},
				),
			)
		})
	}
}

func TestValidateGitRepoUpdate(t *testing.T) {
	testCases := []struct {
		name       string
		update     kargoapi.GitRepoUpdate
		assertions func(*testing.T, kargoapi.GitRepoUpdate, field.ErrorList)
	}{
		{
			name: "more than one config management tool specified",
			update: kargoapi.GitRepoUpdate{
				Render:    &kargoapi.KargoRenderPromotionMechanism{},
				Kustomize: &kargoapi.KustomizePromotionMechanism{},
				Helm:      &kargoapi.HelmPromotionMechanism{},
			},
			assertions: func(t *testing.T, update kargoapi.GitRepoUpdate, errs field.ErrorList) {
				require.Equal(
					t,
					field.ErrorList{
						{
							Type:     field.ErrorTypeInvalid,
							Field:    "gitRepoUpdate",
							BadValue: update,
							Detail: "no more than one of gitRepoUpdate.render, or " +
								"gitRepoUpdate.kustomize, or gitRepoUpdate.helm may be " +
								"defined",
						},
					},
					errs,
				)
			},
		},

		{
			name: "valid",
			update: kargoapi.GitRepoUpdate{
				Kustomize: &kargoapi.KustomizePromotionMechanism{},
			},
			assertions: func(t *testing.T, _ kargoapi.GitRepoUpdate, errs field.ErrorList) {
				require.Nil(t, errs)
			},
		},
	}
	w := &webhook{}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				t,
				testCase.update,
				w.validateGitRepoUpdate(
					field.NewPath("gitRepoUpdate"),
					testCase.update,
				),
			)
		})
	}
}

func TestValidateHelmPromotionMechanism(t *testing.T) {
	testCases := []struct {
		name       string
		promoMech  *kargoapi.HelmPromotionMechanism
		assertions func(*testing.T, *kargoapi.HelmPromotionMechanism, field.ErrorList)
	}{
		{
			name: "nil",
			assertions: func(t *testing.T, _ *kargoapi.HelmPromotionMechanism, errs field.ErrorList) {
				require.Empty(t, errs)
			},
		},

		{
			name: "invalid",
			// Doesn't define any changes
			promoMech: &kargoapi.HelmPromotionMechanism{},
			assertions: func(
				t *testing.T,
				promoMech *kargoapi.HelmPromotionMechanism,
				errs field.ErrorList,
			) {
				require.Equal(
					t,
					field.ErrorList{
						{
							Type:     field.ErrorTypeInvalid,
							Field:    "helm",
							BadValue: promoMech,
							Detail: "at least one of helm.images or helm.charts must be " +
								"non-empty",
						},
					},
					errs,
				)
			},
		},

		{
			name: "valid",
			promoMech: &kargoapi.HelmPromotionMechanism{
				Images: []kargoapi.HelmImageUpdate{
					{},
				},
			},
			assertions: func(t *testing.T, _ *kargoapi.HelmPromotionMechanism, errs field.ErrorList) {
				require.Empty(t, errs)
			},
		},
	}
	w := &webhook{}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				t,
				testCase.promoMech,
				w.validateHelmPromotionMechanism(
					field.NewPath("helm"),
					testCase.promoMech,
				),
			)
		})
	}
}
