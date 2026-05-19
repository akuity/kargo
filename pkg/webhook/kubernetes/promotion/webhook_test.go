package promotion

import (
	"context"
	"encoding/json"
	"errors"
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"
	admissionv1 "k8s.io/api/admission/v1"
	authnv1 "k8s.io/api/authentication/v1"
	authzv1 "k8s.io/api/authorization/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apiserver/pkg/authentication/serviceaccount"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/api"
	k8sevent "github.com/akuity/kargo/pkg/event/kubernetes"
	fakeevent "github.com/akuity/kargo/pkg/kubernetes/event/fake"
	libWebhook "github.com/akuity/kargo/pkg/webhook/kubernetes"
)

func TestNewWebhook(t *testing.T) {
	kubeClient := fake.NewClientBuilder().Build()
	w := newWebhook(
		libWebhook.Config{},
		kubeClient,
		admission.NewDecoder(kubeClient.Scheme()),
		k8sevent.NewEventSender(&fakeevent.EventRecorder{}),
	)
	// Assert that all overridable behaviors were initialized to a default:
	require.NotNil(t, w.getFreightFn)
	require.NotNil(t, w.getStageFn)
	require.NotNil(t, w.validateProjectFn)
	require.NotNil(t, w.authorizeFn)
	require.NotNil(t, w.admissionRequestFromContextFn)
	require.NotNil(t, w.createSubjectAccessReviewFn)
	require.NotNil(t, w.isRequestFromKargoControlplaneFn)
}

func Test_webhook_Default(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(scheme))

	testCases := []struct {
		name       string
		promotion  *kargoapi.Promotion
		req        admission.Request
		webhook    *webhook
		assertions func(*testing.T, *kargoapi.Promotion, error)
	}{
		{
			name: "error getting stage",
			webhook: &webhook{
				admissionRequestFromContextFn: admission.RequestFromContext,
				getStageFn: func(
					context.Context,
					client.Client,
					types.NamespacedName,
				) (*kargoapi.Stage, error) {
					return nil, errors.New("something went wrong")
				},
				isRequestFromKargoControlplaneFn: func(admission.Request) bool {
					return false
				},
			},
			req: admission.Request{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Operation: admissionv1.Create,
				},
			},
			promotion: &kargoapi.Promotion{},
			assertions: func(t *testing.T, _ *kargoapi.Promotion, err error) {
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "stage not found",
			webhook: &webhook{
				admissionRequestFromContextFn: admission.RequestFromContext,
				getStageFn: func(
					context.Context,
					client.Client,
					types.NamespacedName,
				) (*kargoapi.Stage, error) {
					return nil, nil
				},
				isRequestFromKargoControlplaneFn: func(admission.Request) bool {
					return false
				},
			},
			req: admission.Request{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Operation: admissionv1.Create,
				},
			},
			promotion: &kargoapi.Promotion{},
			assertions: func(t *testing.T, _ *kargoapi.Promotion, err error) {
				require.ErrorContains(t, err, "could not find Stage")
			},
		},
		{
			name: "Stage with no PromotionTemplate is rejected",
			webhook: &webhook{
				admissionRequestFromContextFn: admission.RequestFromContext,
				getStageFn: func(
					context.Context,
					client.Client,
					types.NamespacedName,
				) (*kargoapi.Stage, error) {
					return &kargoapi.Stage{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "fake-stage",
							Namespace: "fake-project",
						},
						// No PromotionTemplate.
					}, nil
				},
				isRequestFromKargoControlplaneFn: func(admission.Request) bool {
					return false
				},
			},
			req: admission.Request{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Operation: admissionv1.Create,
				},
			},
			promotion: &kargoapi.Promotion{
				Spec: kargoapi.PromotionSpec{
					Stage:   "fake-stage",
					Freight: "abc1234567",
				},
			},
			assertions: func(t *testing.T, _ *kargoapi.Promotion, err error) {
				require.ErrorContains(t, err, "defines no promotion steps")
			},
		},
		{
			name: "success with PromotionTemplate",
			webhook: &webhook{
				admissionRequestFromContextFn: admission.RequestFromContext,
				getStageFn: func(
					context.Context,
					client.Client,
					types.NamespacedName,
				) (*kargoapi.Stage, error) {
					return &kargoapi.Stage{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "fake-stage",
							Namespace: "fake-project",
						},
						Spec: kargoapi.StageSpec{
							Shard: "fake-shard",
							Vars: []kargoapi.ExpressionVariable{
								{Name: "stage-var", Value: "stage-val"},
							},
							PromotionTemplate: &kargoapi.PromotionTemplate{
								Spec: kargoapi.PromotionTemplateSpec{
									Vars: []kargoapi.ExpressionVariable{
										{Name: "tmpl-var", Value: "tmpl-val"},
									},
									Steps: []kargoapi.PromotionStep{
										{As: "from-template", Uses: "tmpl-step"},
									},
								},
							},
						},
					}, nil
				},
				isRequestFromKargoControlplaneFn: func(admission.Request) bool {
					return false
				},
			},
			req: admission.Request{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Operation: admissionv1.Create,
				},
			},
			// Caller-supplied name, steps, and vars are intentionally set to
			// values that don't appear in the Stage's spec, so that the
			// assertions below verify they were unconditionally discarded.
			promotion: &kargoapi.Promotion{
				ObjectMeta: metav1.ObjectMeta{
					Name: "user-supplied-name",
				},
				Spec: kargoapi.PromotionSpec{
					Stage:   "fake-stage",
					Freight: "abc1234567",
					Steps: []kargoapi.PromotionStep{
						{As: "from-user", Uses: "user-step"},
					},
					Vars: []kargoapi.ExpressionVariable{
						{Name: "user-var", Value: "user-val"},
					},
				},
			},
			assertions: func(t *testing.T, promo *kargoapi.Promotion, err error) {
				require.NoError(t, err)
				// Steps come from the template; the caller's were discarded.
				require.Len(t, promo.Spec.Steps, 1)
				require.Equal(t, "from-template", promo.Spec.Steps[0].As)
				require.Equal(t, "tmpl-step", promo.Spec.Steps[0].Uses)
				// Vars come from the Stage and template only; the caller's
				// were discarded.
				require.Equal(t, []kargoapi.ExpressionVariable{
					{Name: "stage-var", Value: "stage-val"},
					{Name: "tmpl-var", Value: "tmpl-val"},
				}, promo.Spec.Vars)
				// Caller-supplied name was overwritten with one Kargo
				// generated.
				require.NotEqual(t, "user-supplied-name", promo.Name)
				require.Contains(t, promo.Name, "fake-stage")
				require.Contains(t, promo.Name, "abc1234")
				// Shard label and owner reference are set.
				require.Equal(t, "fake-shard", promo.Labels[kargoapi.LabelKeyShard])
				require.NotEmpty(t, promo.OwnerReferences)
			},
		},
		{
			name: "set abort actor when request doesn't come from kargo control plane",
			webhook: &webhook{
				admissionRequestFromContextFn: admission.RequestFromContext,
				getStageFn: func(
					context.Context,
					client.Client,
					types.NamespacedName,
				) (*kargoapi.Stage, error) {
					return &kargoapi.Stage{}, nil
				},
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
						Object: &kargoapi.Promotion{},
					},
				},
			},
			promotion: &kargoapi.Promotion{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						kargoapi.AnnotationKeyAbort: "fake-action",
					},
				},
				Spec: kargoapi.PromotionSpec{
					Steps: []kargoapi.PromotionStep{
						{},
					},
				},
			},
			assertions: func(t *testing.T, promotion *kargoapi.Promotion, err error) {
				require.NoError(t, err)
				require.Contains(t, promotion.Annotations, kargoapi.AnnotationKeyAbort)
				rr, ok := api.AbortPromotionAnnotationValue(promotion.Annotations)
				require.True(t, ok)
				require.Equal(t, &kargoapi.AbortPromotionRequest{
					Action: "fake-action",
					Actor: api.FormatEventKubernetesUserActor(authnv1.UserInfo{
						Username: "real-user",
					}),
					ControlPlane: false,
				}, rr)
			},
		},
		{
			name: "overwrite with admission request user info if abort actor annotation exists",
			webhook: &webhook{
				admissionRequestFromContextFn: admission.RequestFromContext,
				getStageFn: func(
					context.Context,
					client.Client,
					types.NamespacedName,
				) (*kargoapi.Stage, error) {
					return &kargoapi.Stage{}, nil
				},
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
						Object: &kargoapi.Promotion{},
					},
				},
			},
			promotion: &kargoapi.Promotion{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						kargoapi.AnnotationKeyAbort: (&kargoapi.AbortPromotionRequest{
							Action: "fake-action",
							Actor:  "fake-user",
						}).String(),
					},
				},
				Spec: kargoapi.PromotionSpec{
					Steps: []kargoapi.PromotionStep{
						{},
					},
				},
			},
			assertions: func(t *testing.T, promotion *kargoapi.Promotion, err error) {
				require.NoError(t, err)
				require.Contains(t, promotion.Annotations, kargoapi.AnnotationKeyAbort)
				rr, ok := api.AbortPromotionAnnotationValue(promotion.Annotations)
				require.True(t, ok)
				require.Equal(t, &kargoapi.AbortPromotionRequest{
					Action: "fake-action",
					Actor: api.FormatEventKubernetesUserActor(authnv1.UserInfo{
						Username: "real-user",
					}),
					ControlPlane: false,
				}, rr)
			},
		},
		{
			name: "do not overwrite abort actor when request comes from control plane",
			webhook: &webhook{
				admissionRequestFromContextFn: admission.RequestFromContext,
				getStageFn: func(
					context.Context,
					client.Client,
					types.NamespacedName,
				) (*kargoapi.Stage, error) {
					return &kargoapi.Stage{}, nil
				},
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
						Object: &kargoapi.Promotion{},
					},
				},
			},
			promotion: &kargoapi.Promotion{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						kargoapi.AnnotationKeyAbort: (&kargoapi.AbortPromotionRequest{
							Action: "fake-action",
							Actor:  kargoapi.EventActorAdmin,
						}).String(),
					},
				},
				Spec: kargoapi.PromotionSpec{
					Steps: []kargoapi.PromotionStep{
						{},
					},
				},
			},
			assertions: func(t *testing.T, promotion *kargoapi.Promotion, err error) {
				require.NoError(t, err)
				require.Contains(t, promotion.Annotations, kargoapi.AnnotationKeyAbort)
				rr, ok := api.AbortPromotionAnnotationValue(promotion.Annotations)
				require.True(t, ok)
				require.Equal(t, &kargoapi.AbortPromotionRequest{
					Action:       "fake-action",
					Actor:        kargoapi.EventActorAdmin,
					ControlPlane: true,
				}, rr)
			},
		},
		{
			name: "overwrite abort actor when it has changed for the same ID",
			webhook: &webhook{
				admissionRequestFromContextFn: admission.RequestFromContext,
				getStageFn: func(
					context.Context,
					client.Client,
					types.NamespacedName,
				) (*kargoapi.Stage, error) {
					return &kargoapi.Stage{}, nil
				},
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
						Object: &kargoapi.Promotion{
							ObjectMeta: metav1.ObjectMeta{
								Annotations: map[string]string{
									kargoapi.AnnotationKeyAbort: (&kargoapi.AbortPromotionRequest{
										Action: "fake-action",
										Actor:  "fake-user",
									}).String(),
								},
							},
						},
					},
				},
			},
			promotion: &kargoapi.Promotion{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						kargoapi.AnnotationKeyAbort: (&kargoapi.AbortPromotionRequest{
							Action: "fake-action",
							Actor:  "illegitimate-user",
						}).String(),
					},
				},
				Spec: kargoapi.PromotionSpec{
					Steps: []kargoapi.PromotionStep{
						{},
					},
				},
			},
			assertions: func(t *testing.T, promotion *kargoapi.Promotion, err error) {
				require.NoError(t, err)
				require.Contains(t, promotion.Annotations, kargoapi.AnnotationKeyAbort)
				rr, ok := api.AbortPromotionAnnotationValue(promotion.Annotations)
				require.True(t, ok)
				require.Equal(t, &kargoapi.AbortPromotionRequest{
					Action: "fake-action",
					Actor:  api.FormatEventKubernetesUserActor(authnv1.UserInfo{Username: "real-user"}),
				}, rr)
			},
		},
		{
			name: "overwrite abort control plane flag when it has changed for the same ID",
			webhook: &webhook{
				admissionRequestFromContextFn: admission.RequestFromContext,
				getStageFn: func(
					context.Context,
					client.Client,
					types.NamespacedName,
				) (*kargoapi.Stage, error) {
					return &kargoapi.Stage{}, nil
				},
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
						Object: &kargoapi.Promotion{
							ObjectMeta: metav1.ObjectMeta{
								Annotations: map[string]string{
									kargoapi.AnnotationKeyAbort: (&kargoapi.AbortPromotionRequest{
										Action: "fake-action",
										Actor:  "fake-user",
									}).String(),
								},
							},
						},
					},
				},
			},
			promotion: &kargoapi.Promotion{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						kargoapi.AnnotationKeyAbort: (&kargoapi.AbortPromotionRequest{
							Action:       "fake-action",
							Actor:        "fake-user",
							ControlPlane: true,
						}).String(),
					},
				},
				Spec: kargoapi.PromotionSpec{
					Steps: []kargoapi.PromotionStep{
						{},
					},
				},
			},
			assertions: func(t *testing.T, promotion *kargoapi.Promotion, err error) {
				require.NoError(t, err)
				require.Contains(t, promotion.Annotations, kargoapi.AnnotationKeyAbort)
				rr, ok := api.AbortPromotionAnnotationValue(promotion.Annotations)
				require.True(t, ok)
				require.Equal(t, &kargoapi.AbortPromotionRequest{
					Action:       "fake-action",
					Actor:        api.FormatEventKubernetesUserActor(authnv1.UserInfo{Username: "real-user"}),
					ControlPlane: false,
				}, rr)
			},
		},
		{
			name: "ignore empty abort annotation",
			webhook: &webhook{
				admissionRequestFromContextFn: admission.RequestFromContext,
				getStageFn: func(
					context.Context,
					client.Client,
					types.NamespacedName,
				) (*kargoapi.Stage, error) {
					return &kargoapi.Stage{}, nil
				},
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
						Object: &kargoapi.Promotion{},
					},
				},
			},
			promotion: &kargoapi.Promotion{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						kargoapi.AnnotationKeyAbort: "",
					},
				},
				Spec: kargoapi.PromotionSpec{
					Steps: []kargoapi.PromotionStep{
						{},
					},
				},
			},
			assertions: func(t *testing.T, promotion *kargoapi.Promotion, err error) {
				require.NoError(t, err)
				v, ok := promotion.Annotations[kargoapi.AnnotationKeyAbort]
				require.True(t, ok)
				require.Empty(t, v)
			},
		},
		{
			name: "ignore abort annotation with empty ID",
			webhook: &webhook{
				admissionRequestFromContextFn: admission.RequestFromContext,
				getStageFn: func(
					context.Context,
					client.Client,
					types.NamespacedName,
				) (*kargoapi.Stage, error) {
					return &kargoapi.Stage{}, nil
				},
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
						Object: &kargoapi.Promotion{},
					},
				},
			},
			promotion: &kargoapi.Promotion{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						kargoapi.AnnotationKeyAbort: (&kargoapi.AbortPromotionRequest{
							Action: "",
						}).String(),
					},
				},
				Spec: kargoapi.PromotionSpec{
					Steps: []kargoapi.PromotionStep{
						{},
					},
				},
			},
			assertions: func(t *testing.T, promotion *kargoapi.Promotion, err error) {
				require.NoError(t, err)
				v, ok := promotion.Annotations[kargoapi.AnnotationKeyAbort]
				require.True(t, ok)
				require.Equal(t, (&kargoapi.AbortPromotionRequest{
					Action: "",
				}).String(), v)
			},
		},
		{
			name: "ignore unchanged abort annotation",
			webhook: &webhook{
				admissionRequestFromContextFn: admission.RequestFromContext,
				getStageFn: func(
					context.Context,
					client.Client,
					types.NamespacedName,
				) (*kargoapi.Stage, error) {
					return &kargoapi.Stage{}, nil
				},
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
						Object: &kargoapi.Promotion{
							ObjectMeta: metav1.ObjectMeta{
								Annotations: map[string]string{
									kargoapi.AnnotationKeyAbort: (&kargoapi.AbortPromotionRequest{
										Action: "fake-action",
										Actor:  "fake-user",
									}).String(),
								},
							},
						},
					},
				},
			},
			promotion: &kargoapi.Promotion{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						kargoapi.AnnotationKeyAbort: (&kargoapi.AbortPromotionRequest{
							Action: "fake-action",
							Actor:  "fake-user",
						}).String(),
					},
				},
				Spec: kargoapi.PromotionSpec{
					Steps: []kargoapi.PromotionStep{
						{},
					},
				},
			},
			assertions: func(t *testing.T, promotion *kargoapi.Promotion, err error) {
				require.NoError(t, err)
				v, ok := promotion.Annotations[kargoapi.AnnotationKeyAbort]
				require.True(t, ok)
				require.Equal(t, (&kargoapi.AbortPromotionRequest{
					Action: "fake-action",
					Actor:  "fake-user",
				}).String(), v)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			// Apply default decoder to all test cases
			testCase.webhook.decoder = admission.NewDecoder(scheme)

			// Make sure old object has corresponding Raw data instead of Object
			// since controller-runtime doesn't decode the old object.
			if testCase.req.OldObject.Object != nil {
				data, err := json.Marshal(testCase.req.OldObject.Object)
				require.NoError(t, err)
				testCase.req.OldObject.Raw = data
				testCase.req.OldObject.Object = nil
			}

			ctx := admission.NewContextWithRequest(
				t.Context(),
				testCase.req,
			)
			testCase.assertions(
				t,
				testCase.promotion,
				testCase.webhook.Default(ctx, testCase.promotion),
			)
		})
	}
}

func Test_webhook_ValidateCreate(t *testing.T) {
	const testWarehouse = "fake-warehouse"

	testCases := []struct {
		name       string
		webhook    *webhook
		userInfo   *authnv1.UserInfo
		promotion  *kargoapi.Promotion
		assertions func(*testing.T, *fakeevent.EventRecorder, error)
	}{
		{
			name: "error validating project",
			webhook: &webhook{
				validateProjectFn: func(
					context.Context,
					client.Client,
					client.Object,
				) error {
					return errors.New("something went wrong")
				},
			},
			promotion: &kargoapi.Promotion{
				Spec: kargoapi.PromotionSpec{Freight: "fake-freight"},
			},
			assertions: func(t *testing.T, _ *fakeevent.EventRecorder, err error) {
				var statusErr *apierrors.StatusError
				require.True(t, errors.As(err, &statusErr))
				require.Equal(
					t,
					metav1.StatusReasonInternalError,
					statusErr.ErrStatus.Reason,
				)
				require.Contains(t, statusErr.ErrStatus.Message, "something went wrong")
			},
		},
		{
			name: "authorization error",
			webhook: &webhook{
				validateProjectFn: func(
					context.Context,
					client.Client,
					client.Object,
				) error {
					return nil
				},
				authorizeFn: func(context.Context, *kargoapi.Promotion, string) error {
					return errors.New("something went wrong")
				},
			},
			promotion: &kargoapi.Promotion{
				Spec: kargoapi.PromotionSpec{Freight: "fake-freight"},
			},
			assertions: func(t *testing.T, _ *fakeevent.EventRecorder, err error) {
				var statusErr *apierrors.StatusError
				require.True(t, errors.As(err, &statusErr))
				require.Equal(
					t,
					metav1.StatusReasonInternalError,
					statusErr.ErrStatus.Reason,
				)
				require.Contains(t, statusErr.ErrStatus.Message, "something went wrong")
			},
		},
		{
			name: "error getting Stage",
			webhook: &webhook{
				validateProjectFn: func(
					context.Context,
					client.Client,
					client.Object,
				) error {
					return nil
				},
				authorizeFn: func(context.Context, *kargoapi.Promotion, string) error {
					return nil
				},
				admissionRequestFromContextFn: admission.RequestFromContext,
				getStageFn: func(
					context.Context,
					client.Client,
					types.NamespacedName,
				) (*kargoapi.Stage, error) {
					return nil, errors.New("something went wrong")
				},
			},
			promotion: &kargoapi.Promotion{
				Spec: kargoapi.PromotionSpec{Freight: "fake-freight"},
			},
			assertions: func(t *testing.T, _ *fakeevent.EventRecorder, err error) {
				var statusErr *apierrors.StatusError
				require.True(t, errors.As(err, &statusErr))
				require.Equal(
					t,
					metav1.StatusReasonInternalError,
					statusErr.ErrStatus.Reason,
				)
				require.Contains(t, statusErr.ErrStatus.Message, "something went wrong")
			},
		},
		{
			name: "error getting Freight",
			webhook: &webhook{
				validateProjectFn: func(
					context.Context,
					client.Client,
					client.Object,
				) error {
					return nil
				},
				authorizeFn: func(context.Context, *kargoapi.Promotion, string) error {
					return nil
				},
				admissionRequestFromContextFn: admission.RequestFromContext,
				getStageFn: func(
					context.Context,
					client.Client,
					types.NamespacedName,
				) (*kargoapi.Stage, error) {
					return &kargoapi.Stage{}, nil
				},
				getFreightFn: func(
					context.Context,
					client.Client,
					types.NamespacedName,
				) (*kargoapi.Freight, error) {
					return nil, errors.New("something went wrong")
				},
			},
			promotion: &kargoapi.Promotion{
				Spec: kargoapi.PromotionSpec{Freight: "fake-freight"},
			},
			assertions: func(t *testing.T, _ *fakeevent.EventRecorder, err error) {
				var statusErr *apierrors.StatusError
				require.True(t, errors.As(err, &statusErr))
				require.Equal(
					t,
					metav1.StatusReasonInternalError,
					statusErr.ErrStatus.Reason,
				)
				require.Contains(t, statusErr.ErrStatus.Message, "something went wrong")
			},
		},
		{
			name: "Freight is not available to Stage",
			webhook: &webhook{
				validateProjectFn: func(
					context.Context,
					client.Client,
					client.Object,
				) error {
					return nil
				},
				authorizeFn: func(context.Context, *kargoapi.Promotion, string) error {
					return nil
				},
				admissionRequestFromContextFn: admission.RequestFromContext,
				getStageFn: func(
					context.Context,
					client.Client,
					types.NamespacedName,
				) (*kargoapi.Stage, error) {
					return &kargoapi.Stage{}, nil
				},
				getFreightFn: func(
					context.Context,
					client.Client,
					types.NamespacedName,
				) (*kargoapi.Freight, error) {
					return &kargoapi.Freight{}, nil
				},
			},
			promotion: &kargoapi.Promotion{
				Spec: kargoapi.PromotionSpec{Freight: "fake-freight"},
			},
			assertions: func(t *testing.T, _ *fakeevent.EventRecorder, err error) {
				var statusErr *apierrors.StatusError
				require.True(t, errors.As(err, &statusErr))
				require.Equal(t, metav1.StatusReasonInvalid, statusErr.ErrStatus.Reason)
				require.Contains(
					t,
					statusErr.ErrStatus.Message,
					"Freight is not available to this Stage",
				)
			},
		},
		{
			name: "record promotion created event on non-controlplane request",
			webhook: &webhook{
				validateProjectFn: func(
					context.Context,
					client.Client,
					client.Object,
				) error {
					return nil
				},
				authorizeFn: func(context.Context, *kargoapi.Promotion, string) error {
					return nil
				},
				admissionRequestFromContextFn: admission.RequestFromContext,
				getStageFn: func(
					context.Context,
					client.Client,
					types.NamespacedName,
				) (*kargoapi.Stage, error) {
					return &kargoapi.Stage{
						Spec: kargoapi.StageSpec{
							RequestedFreight: []kargoapi.FreightRequest{{
								Origin: kargoapi.FreightOrigin{
									Kind: kargoapi.FreightOriginKindWarehouse,
									Name: testWarehouse,
								},
								Sources: kargoapi.FreightSources{Direct: true},
							}},
						},
					}, nil
				},
				getFreightFn: func(
					context.Context,
					client.Client,
					types.NamespacedName,
				) (*kargoapi.Freight, error) {
					return &kargoapi.Freight{
						Origin: kargoapi.FreightOrigin{
							Kind: kargoapi.FreightOriginKindWarehouse,
							Name: testWarehouse,
						},
					}, nil
				},
				isRequestFromKargoControlplaneFn: libWebhook.IsRequestFromKargoControlplane(
					regexp.MustCompile("^system:serviceaccount:kargo:(kargo-api|kargo-controller)$"),
				),
			},
			userInfo: &authnv1.UserInfo{
				Username: "fake-user",
			},
			promotion: &kargoapi.Promotion{
				Spec: kargoapi.PromotionSpec{
					Freight: "fake-freight",
					Steps:   []kargoapi.PromotionStep{{Uses: "fake-step"}},
				},
			},
			assertions: func(t *testing.T, r *fakeevent.EventRecorder, err error) {
				require.NoError(t, err)
				require.Len(t, r.Events, 1)
				event := <-r.Events
				require.Equal(t, string(kargoapi.EventTypePromotionCreated), event.Reason)
			},
		},
		{
			name: "skip recording promotion created event on controlplane request",
			webhook: &webhook{
				validateProjectFn: func(
					context.Context,
					client.Client,
					client.Object,
				) error {
					return nil
				},
				authorizeFn: func(context.Context, *kargoapi.Promotion, string) error {
					return nil
				},
				getStageFn: func(
					context.Context,
					client.Client,
					types.NamespacedName,
				) (*kargoapi.Stage, error) {
					return &kargoapi.Stage{
						Spec: kargoapi.StageSpec{
							RequestedFreight: []kargoapi.FreightRequest{{
								Origin: kargoapi.FreightOrigin{
									Kind: kargoapi.FreightOriginKindWarehouse,
									Name: "fake-warehouse",
								},
								Sources: kargoapi.FreightSources{Direct: true},
							}},
						},
					}, nil
				},
				getFreightFn: func(
					context.Context,
					client.Client,
					types.NamespacedName,
				) (*kargoapi.Freight, error) {
					return &kargoapi.Freight{
						Origin: kargoapi.FreightOrigin{
							Kind: kargoapi.FreightOriginKindWarehouse,
							Name: "fake-warehouse",
						},
					}, nil
				},
				admissionRequestFromContextFn: admission.RequestFromContext,
				isRequestFromKargoControlplaneFn: libWebhook.IsRequestFromKargoControlplane(
					regexp.MustCompile("^system:serviceaccount:kargo:(kargo-api|kargo-controller)$"),
				),
			},
			userInfo: &authnv1.UserInfo{
				Username: serviceaccount.ServiceAccountUsernamePrefix + "kargo:kargo-api",
			},
			promotion: &kargoapi.Promotion{
				Spec: kargoapi.PromotionSpec{
					Freight: "fake-freight",
					Steps:   []kargoapi.PromotionStep{{Uses: "fake-step"}},
				},
			},
			assertions: func(t *testing.T, r *fakeevent.EventRecorder, err error) {
				require.NoError(t, err)
				require.Empty(t, r.Events)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			recorder := fakeevent.NewEventRecorder(1)
			testCase.webhook.sender = k8sevent.NewEventSender(recorder)

			var req admission.Request
			if testCase.userInfo != nil {
				req.UserInfo = *testCase.userInfo
			}
			ctx := admission.NewContextWithRequest(t.Context(), req)

			_, err := testCase.webhook.ValidateCreate(ctx, testCase.promotion)
			testCase.assertions(t, recorder, err)
		})
	}
}

func Tes_webhook_tValidateUpdate(t *testing.T) {
	testCases := []struct {
		name        string
		setup       func() (*kargoapi.Promotion, *kargoapi.Promotion)
		authorizeFn func(
			ctx context.Context,
			promo *kargoapi.Promotion,
			action string,
		) error
		assertions func(*testing.T, error)
	}{
		{
			name: "authorization error",
			setup: func() (*kargoapi.Promotion, *kargoapi.Promotion) {
				return &kargoapi.Promotion{}, &kargoapi.Promotion{}
			},
			authorizeFn: func(context.Context, *kargoapi.Promotion, string) error {
				return errors.New("something went wrong")
			},
			assertions: func(t *testing.T, err error) {
				var statusErr *apierrors.StatusError
				require.True(t, errors.As(err, &statusErr))
				require.Equal(
					t,
					metav1.StatusReasonInternalError,
					statusErr.ErrStatus.Reason,
				)
				require.Contains(t, statusErr.ErrStatus.Message, "something went wrong")
			},
		},

		{
			name: "attempt to mutate",
			setup: func() (*kargoapi.Promotion, *kargoapi.Promotion) {
				oldPromo := &kargoapi.Promotion{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "fake-name",
						Namespace: "fake-namespace",
					},
					Spec: kargoapi.PromotionSpec{
						Stage:   "fake-stage",
						Freight: "fake-freight",
					},
				}
				newPromo := oldPromo.DeepCopy()
				newPromo.Spec.Freight = "another-fake-freight"
				return oldPromo, newPromo
			},
			authorizeFn: func(context.Context, *kargoapi.Promotion, string) error {
				return nil
			},
			assertions: func(t *testing.T, err error) {
				var statusErr *apierrors.StatusError
				require.True(t, errors.As(err, &statusErr))
				require.Equal(t, metav1.StatusReasonInvalid, statusErr.ErrStatus.Reason)
				require.Contains(t, statusErr.ErrStatus.Message, "spec is immutable")
			},
		},

		{
			name: "update without mutation",
			setup: func() (*kargoapi.Promotion, *kargoapi.Promotion) {
				oldPromo := &kargoapi.Promotion{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "fake-name",
						Namespace: "fake-namespace",
					},
					Spec: kargoapi.PromotionSpec{
						Stage:   "fake-stage",
						Freight: "fake-freight",
					},
				}
				newPromo := oldPromo.DeepCopy()
				return oldPromo, newPromo
			},
			authorizeFn: func(context.Context, *kargoapi.Promotion, string) error {
				return nil
			},
			assertions: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			w := &webhook{
				authorizeFn: testCase.authorizeFn,
			}
			oldPromo, newPromo := testCase.setup()
			_, err := w.ValidateUpdate(t.Context(), oldPromo, newPromo)
			testCase.assertions(t, err)
		})
	}
}

func Test_webhook_ValidateDelete(t *testing.T) {
	testCases := []struct {
		name       string
		webhook    *webhook
		assertions func(*testing.T, error)
	}{
		{
			name: "authorization error",
			webhook: &webhook{
				authorizeFn: func(context.Context, *kargoapi.Promotion, string) error {
					return errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "success",
			webhook: &webhook{
				authorizeFn: func(context.Context, *kargoapi.Promotion, string) error {
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
			_, err := testCase.webhook.ValidateDelete(
				t.Context(),
				&kargoapi.Promotion{},
			)
			testCase.assertions(t, err)
		})
	}
}

func Test_webhook_Authorize(t *testing.T) {
	testCases := []struct {
		name                           string
		externalWebhooksServerUsername string
		admissionRequestFromContextFn  func(
			context.Context,
		) (admission.Request, error)
		createSubjectAccessReviewFn func(
			context.Context,
			client.Object,
			...client.CreateOption,
		) error
		assertions func(*testing.T, error)
	}{
		{
			name:                           "error getting admission request bound to context",
			externalWebhooksServerUsername: "system:serviceaccount:kargo:kargo-external-webhooks-server",
			admissionRequestFromContextFn: func(
				context.Context,
			) (admission.Request, error) {
				return admission.Request{}, errors.New("something went wrong")
			},
			assertions: func(t *testing.T, err error) {
				require.ErrorContains(
					t, err, "error retrieving admission request from context; refusing to",
				)
			},
		},
		{
			name:                           "error creating subject access review",
			externalWebhooksServerUsername: "system:serviceaccount:kargo:kargo-external-webhooks-server",
			admissionRequestFromContextFn: func(
				context.Context,
			) (admission.Request, error) {
				return admission.Request{}, nil
			},
			createSubjectAccessReviewFn: func(
				context.Context,
				client.Object,
				...client.CreateOption,
			) error {
				return errors.New("something went wrong")
			},
			assertions: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "error creating SubjectAccessReview")
			},
		},
		{
			name:                           "subject is not authorized",
			externalWebhooksServerUsername: "system:serviceaccount:kargo:kargo-external-webhooks-server",
			admissionRequestFromContextFn: func(
				context.Context,
			) (admission.Request, error) {
				return admission.Request{}, nil
			},
			createSubjectAccessReviewFn: func(
				_ context.Context,
				obj client.Object,
				_ ...client.CreateOption,
			) error {
				obj.(*authzv1.SubjectAccessReview).Status.Allowed = false // nolint: forcetypeassert
				return nil
			},
			assertions: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "is not permitted")
			},
		},
		{
			name:                           "subject is authorized",
			externalWebhooksServerUsername: "system:serviceaccount:kargo:kargo-external-webhooks-server",
			admissionRequestFromContextFn: func(
				context.Context,
			) (admission.Request, error) {
				return admission.Request{}, nil
			},
			createSubjectAccessReviewFn: func(
				_ context.Context,
				obj client.Object,
				_ ...client.CreateOption,
			) error {
				obj.(*authzv1.SubjectAccessReview).Status.Allowed = true // nolint: forcetypeassert
				return nil
			},
			assertions: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			name:                           "bypass for external webhooks server",
			externalWebhooksServerUsername: "system:serviceaccount:kargo:kargo-external-webhooks-server",
			admissionRequestFromContextFn: func(
				context.Context,
			) (admission.Request, error) {
				return admission.Request{
					AdmissionRequest: admissionv1.AdmissionRequest{
						UserInfo: authnv1.UserInfo{
							Username: "system:serviceaccount:kargo:kargo-external-webhooks-server",
						},
					},
				}, nil
			},
			// createSubjectAccessReviewFn is intentionally nil to confirm the
			// SAR check is never reached for the external webhooks server.
			assertions: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			name:                           "no bypass when username does not match external webhooks server",
			externalWebhooksServerUsername: "system:serviceaccount:kargo:kargo-external-webhooks-server",
			admissionRequestFromContextFn: func(
				context.Context,
			) (admission.Request, error) {
				return admission.Request{
					AdmissionRequest: admissionv1.AdmissionRequest{
						UserInfo: authnv1.UserInfo{
							Username: "some-other-user",
						},
					},
				}, nil
			},
			createSubjectAccessReviewFn: func(
				_ context.Context,
				obj client.Object,
				_ ...client.CreateOption,
			) error {
				obj.(*authzv1.SubjectAccessReview).Status.Allowed = true // nolint: forcetypeassert
				return nil
			},
			assertions: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			w := &webhook{
				externalWebhooksServerUsername: testCase.externalWebhooksServerUsername,
				admissionRequestFromContextFn:  testCase.admissionRequestFromContextFn,
				createSubjectAccessReviewFn:    testCase.createSubjectAccessReviewFn,
			}
			testCase.assertions(
				t,
				w.authorize(
					t.Context(),
					&kargoapi.Promotion{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "fake-promotion",
							Namespace: "fake-namespace",
						},
						Spec: kargoapi.PromotionSpec{
							Stage: "fake-stage",
						},
					},
					"create",
				),
			)
		})
	}
}
