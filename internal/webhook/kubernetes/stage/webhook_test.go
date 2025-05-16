package stage

import (
	"context"
	"encoding/json"
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
	"github.com/akuity/kargo/internal/api"
	libWebhook "github.com/akuity/kargo/internal/webhook/kubernetes"
)

func TestNewWebhook(t *testing.T) {
	kubeClient := fake.NewClientBuilder().Build()
	w := newWebhook(
		libWebhook.Config{},
		kubeClient,
		admission.NewDecoder(kubeClient.Scheme()),
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
	scheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(scheme))

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
			stage: &kargoapi.Stage{},
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
				Spec: kargoapi.StageSpec{
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
			},
			assertions: func(t *testing.T, stage *kargoapi.Stage, err error) {
				require.NoError(t, err)
				require.Empty(t, stage.Spec.Shard)
				_, ok := stage.Labels[kargoapi.ShardLabelKey]
				require.False(t, ok)
			},
		},
		{
			name: "set reverify actor when request doesn't come from kargo control plane",
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
			},
			assertions: func(t *testing.T, stage *kargoapi.Stage, err error) {
				require.NoError(t, err)
				require.Contains(t, stage.Annotations, kargoapi.AnnotationKeyReverify)
				rr, ok := api.ReverifyAnnotationValue(stage.Annotations)
				require.True(t, ok)
				require.Equal(t, &kargoapi.VerificationRequest{
					ID: "fake-id",
					Actor: api.FormatEventKubernetesUserActor(authnv1.UserInfo{
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
						kargoapi.AnnotationKeyReverify: (&kargoapi.VerificationRequest{
							ID:    "fake-id",
							Actor: "fake-user",
						}).String(),
					},
				},
			},
			assertions: func(t *testing.T, stage *kargoapi.Stage, err error) {
				require.NoError(t, err)
				require.Contains(t, stage.Annotations, kargoapi.AnnotationKeyReverify)
				rr, ok := api.ReverifyAnnotationValue(stage.Annotations)
				require.True(t, ok)
				require.Equal(t, &kargoapi.VerificationRequest{
					ID: "fake-id",
					Actor: api.FormatEventKubernetesUserActor(authnv1.UserInfo{
						Username: "real-user",
					}),
					ControlPlane: false,
				}, rr)
			},
		},
		{
			name: "do not overwrite reverify actor when request comes from control plane",
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
						kargoapi.AnnotationKeyReverify: (&kargoapi.VerificationRequest{
							ID:    "fake-id",
							Actor: kargoapi.EventActorAdmin,
						}).String(),
					},
				},
			},
			assertions: func(t *testing.T, stage *kargoapi.Stage, err error) {
				require.NoError(t, err)
				require.Contains(t, stage.Annotations, kargoapi.AnnotationKeyReverify)
				rr, ok := api.ReverifyAnnotationValue(stage.Annotations)
				require.True(t, ok)
				require.Equal(t, &kargoapi.VerificationRequest{
					ID:           "fake-id",
					Actor:        kargoapi.EventActorAdmin,
					ControlPlane: true,
				}, rr)
			},
		},
		{
			name: "overwrite reverify actor when it has changed for the same ID",
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
						Object: &kargoapi.Stage{
							ObjectMeta: metav1.ObjectMeta{
								Annotations: map[string]string{
									kargoapi.AnnotationKeyReverify: (&kargoapi.VerificationRequest{
										ID:    "fake-id",
										Actor: "fake-user",
									}).String(),
								},
							},
						},
					},
				},
			},
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						kargoapi.AnnotationKeyReverify: (&kargoapi.VerificationRequest{
							ID:    "fake-id",
							Actor: "illegitimate-user",
						}).String(),
					},
				},
			},
			assertions: func(t *testing.T, stage *kargoapi.Stage, err error) {
				require.NoError(t, err)
				require.Contains(t, stage.Annotations, kargoapi.AnnotationKeyReverify)
				rr, ok := api.ReverifyAnnotationValue(stage.Annotations)
				require.True(t, ok)
				require.Equal(t, &kargoapi.VerificationRequest{
					ID:    "fake-id",
					Actor: api.FormatEventKubernetesUserActor(authnv1.UserInfo{Username: "real-user"}),
				}, rr)
			},
		},
		{
			name: "overwrite reverify control plane flag when it has changed for the same ID",
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
						Object: &kargoapi.Stage{
							ObjectMeta: metav1.ObjectMeta{
								Annotations: map[string]string{
									kargoapi.AnnotationKeyReverify: (&kargoapi.VerificationRequest{
										ID:    "fake-id",
										Actor: "fake-user",
									}).String(),
								},
							},
						},
					},
				},
			},
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						kargoapi.AnnotationKeyReverify: (&kargoapi.VerificationRequest{
							ID:           "fake-id",
							Actor:        "fake-user",
							ControlPlane: true,
						}).String(),
					},
				},
			},
			assertions: func(t *testing.T, stage *kargoapi.Stage, err error) {
				require.NoError(t, err)
				require.Contains(t, stage.Annotations, kargoapi.AnnotationKeyReverify)
				rr, ok := api.ReverifyAnnotationValue(stage.Annotations)
				require.True(t, ok)
				require.Equal(t, &kargoapi.VerificationRequest{
					ID:           "fake-id",
					Actor:        api.FormatEventKubernetesUserActor(authnv1.UserInfo{Username: "real-user"}),
					ControlPlane: false,
				}, rr)
			},
		},
		{
			name: "ignore empty reverify annotation",
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
						kargoapi.AnnotationKeyReverify: "",
					},
				},
			},
			assertions: func(t *testing.T, stage *kargoapi.Stage, err error) {
				require.NoError(t, err)
				v, ok := stage.Annotations[kargoapi.AnnotationKeyReverify]
				require.True(t, ok)
				require.Empty(t, v)
			},
		},
		{
			name: "ignore reverify annotation with empty ID",
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
						kargoapi.AnnotationKeyReverify: (&kargoapi.VerificationRequest{
							ID: "",
						}).String(),
					},
				},
			},
			assertions: func(t *testing.T, stage *kargoapi.Stage, err error) {
				require.NoError(t, err)
				v, ok := stage.Annotations[kargoapi.AnnotationKeyReverify]
				require.True(t, ok)
				require.Equal(t, (&kargoapi.VerificationRequest{
					ID: "",
				}).String(), v)
			},
		},
		{
			name: "ignore unchanged reverify annotation",
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
						Object: &kargoapi.Stage{
							ObjectMeta: metav1.ObjectMeta{
								Annotations: map[string]string{
									kargoapi.AnnotationKeyReverify: (&kargoapi.VerificationRequest{
										ID:    "fake-id",
										Actor: "fake-user",
									}).String(),
								},
							},
						},
					},
				},
			},
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						kargoapi.AnnotationKeyReverify: (&kargoapi.VerificationRequest{
							ID:    "fake-id",
							Actor: "fake-user",
						}).String(),
					},
				},
			},
			assertions: func(t *testing.T, stage *kargoapi.Stage, err error) {
				require.NoError(t, err)
				v, ok := stage.Annotations[kargoapi.AnnotationKeyReverify]
				require.True(t, ok)
				require.Equal(t, (&kargoapi.VerificationRequest{
					ID:    "fake-id",
					Actor: "fake-user",
				}).String(), v)
			},
		},
		{
			name: "set abort actor when request doesn't come from kargo control plane",
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
						kargoapi.AnnotationKeyAbort: "fake-id",
					},
				},
			},
			assertions: func(t *testing.T, stage *kargoapi.Stage, err error) {
				require.NoError(t, err)
				require.Contains(t, stage.Annotations, kargoapi.AnnotationKeyAbort)
				rr, ok := api.AbortVerificationAnnotationValue(stage.Annotations)
				require.True(t, ok)
				require.Equal(t, &kargoapi.VerificationRequest{
					ID: "fake-id",
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
						kargoapi.AnnotationKeyAbort: (&kargoapi.VerificationRequest{
							ID:    "fake-id",
							Actor: "fake-user",
						}).String(),
					},
				},
			},
			assertions: func(t *testing.T, stage *kargoapi.Stage, err error) {
				require.NoError(t, err)
				require.Contains(t, stage.Annotations, kargoapi.AnnotationKeyAbort)
				rr, ok := api.AbortVerificationAnnotationValue(stage.Annotations)
				require.True(t, ok)
				require.Equal(t, &kargoapi.VerificationRequest{
					ID: "fake-id",
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
						kargoapi.AnnotationKeyAbort: (&kargoapi.VerificationRequest{
							ID:    "fake-id",
							Actor: kargoapi.EventActorAdmin,
						}).String(),
					},
				},
			},
			assertions: func(t *testing.T, stage *kargoapi.Stage, err error) {
				require.NoError(t, err)
				require.Contains(t, stage.Annotations, kargoapi.AnnotationKeyAbort)
				rr, ok := api.AbortVerificationAnnotationValue(stage.Annotations)
				require.True(t, ok)
				require.Equal(t, &kargoapi.VerificationRequest{
					ID:           "fake-id",
					Actor:        kargoapi.EventActorAdmin,
					ControlPlane: true,
				}, rr)
			},
		},
		{
			name: "overwrite abort actor when it has changed for the same ID",
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
						Object: &kargoapi.Stage{
							ObjectMeta: metav1.ObjectMeta{
								Annotations: map[string]string{
									kargoapi.AnnotationKeyAbort: (&kargoapi.VerificationRequest{
										ID:    "fake-id",
										Actor: "fake-user",
									}).String(),
								},
							},
						},
					},
				},
			},
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						kargoapi.AnnotationKeyAbort: (&kargoapi.VerificationRequest{
							ID:    "fake-id",
							Actor: "illegitimate-user",
						}).String(),
					},
				},
			},
			assertions: func(t *testing.T, stage *kargoapi.Stage, err error) {
				require.NoError(t, err)
				require.Contains(t, stage.Annotations, kargoapi.AnnotationKeyAbort)
				rr, ok := api.AbortVerificationAnnotationValue(stage.Annotations)
				require.True(t, ok)
				require.Equal(t, &kargoapi.VerificationRequest{
					ID:    "fake-id",
					Actor: api.FormatEventKubernetesUserActor(authnv1.UserInfo{Username: "real-user"}),
				}, rr)
			},
		},
		{
			name: "overwrite abort control plane flag when it has changed for the same ID",
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
						Object: &kargoapi.Stage{
							ObjectMeta: metav1.ObjectMeta{
								Annotations: map[string]string{
									kargoapi.AnnotationKeyAbort: (&kargoapi.VerificationRequest{
										ID:    "fake-id",
										Actor: "fake-user",
									}).String(),
								},
							},
						},
					},
				},
			},
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						kargoapi.AnnotationKeyAbort: (&kargoapi.VerificationRequest{
							ID:           "fake-id",
							Actor:        "fake-user",
							ControlPlane: true,
						}).String(),
					},
				},
			},
			assertions: func(t *testing.T, stage *kargoapi.Stage, err error) {
				require.NoError(t, err)
				require.Contains(t, stage.Annotations, kargoapi.AnnotationKeyAbort)
				rr, ok := api.AbortVerificationAnnotationValue(stage.Annotations)
				require.True(t, ok)
				require.Equal(t, &kargoapi.VerificationRequest{
					ID:           "fake-id",
					Actor:        api.FormatEventKubernetesUserActor(authnv1.UserInfo{Username: "real-user"}),
					ControlPlane: false,
				}, rr)
			},
		},
		{
			name: "ignore empty abort annotation",
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
						kargoapi.AnnotationKeyAbort: "",
					},
				},
			},
			assertions: func(t *testing.T, stage *kargoapi.Stage, err error) {
				require.NoError(t, err)
				v, ok := stage.Annotations[kargoapi.AnnotationKeyAbort]
				require.True(t, ok)
				require.Empty(t, v)
			},
		},
		{
			name: "ignore abort annotation with empty ID",
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
						kargoapi.AnnotationKeyAbort: (&kargoapi.VerificationRequest{
							ID: "",
						}).String(),
					},
				},
			},
			assertions: func(t *testing.T, stage *kargoapi.Stage, err error) {
				require.NoError(t, err)
				v, ok := stage.Annotations[kargoapi.AnnotationKeyAbort]
				require.True(t, ok)
				require.Equal(t, (&kargoapi.VerificationRequest{
					ID: "",
				}).String(), v)
			},
		},
		{
			name: "ignore unchanged abort annotation",
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
						Object: &kargoapi.Stage{
							ObjectMeta: metav1.ObjectMeta{
								Annotations: map[string]string{
									kargoapi.AnnotationKeyAbort: (&kargoapi.VerificationRequest{
										ID:    "fake-id",
										Actor: "fake-user",
									}).String(),
								},
							},
						},
					},
				},
			},
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						kargoapi.AnnotationKeyAbort: (&kargoapi.VerificationRequest{
							ID:    "fake-id",
							Actor: "fake-user",
						}).String(),
					},
				},
			},
			assertions: func(t *testing.T, stage *kargoapi.Stage, err error) {
				require.NoError(t, err)
				v, ok := stage.Annotations[kargoapi.AnnotationKeyAbort]
				require.True(t, ok)
				require.Equal(t, (&kargoapi.VerificationRequest{
					ID:    "fake-id",
					Actor: "fake-user",
				}).String(), v)
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Apply default decoder to all test cases
			tc.webhook.decoder = admission.NewDecoder(scheme)

			// Make sure old object has corresponding Raw data instead of Object
			// since controller-runtime doesn't decode the old object.
			if tc.req.OldObject.Object != nil {
				data, err := json.Marshal(tc.req.OldObject.Object)
				require.NoError(t, err)
				tc.req.OldObject.Raw = data
				tc.req.OldObject.Object = nil
			}

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
	testFreightRequest := kargoapi.FreightRequest{
		Origin: kargoapi.FreightOrigin{
			Kind: kargoapi.FreightOriginKindWarehouse,
			Name: "test-warehouse",
		},
	}
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
				// Has multiple sources for one Freight origin...
				RequestedFreight: []kargoapi.FreightRequest{
					testFreightRequest,
					testFreightRequest,
				},
				PromotionTemplate: &kargoapi.PromotionTemplate{
					Spec: kargoapi.PromotionTemplateSpec{
						// This step alias matches a reserved pattern
						Steps: []kargoapi.PromotionStep{{As: "step-42"}},
					},
				},
			},
			assertions: func(t *testing.T, spec *kargoapi.StageSpec, errs field.ErrorList) {
				// We really want to see that all underlying errors have been bubbled up
				// to this level and been aggregated.
				require.Equal(
					t,
					field.ErrorList{
						{
							Type:     field.ErrorTypeInvalid,
							Field:    "spec.requestedFreight",
							BadValue: spec.RequestedFreight,
							Detail: `freight with origin Warehouse/test-warehouse requested multiple ` +
								"times in spec.requestedFreight",
						},
						{
							Type:     field.ErrorTypeInvalid,
							Field:    "spec.promotionTemplate.spec.steps[0].as",
							BadValue: "step-42",
							Detail:   "step alias is reserved",
						},
					},
					errs,
				)
			},
		},

		{
			name: "valid",
			spec: &kargoapi.StageSpec{
				RequestedFreight: []kargoapi.FreightRequest{
					testFreightRequest,
				},
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

func TestValidateRequestedFreight(t *testing.T) {
	testFreightRequest := kargoapi.FreightRequest{
		Origin: kargoapi.FreightOrigin{
			Kind: kargoapi.FreightOriginKindWarehouse,
			Name: "test-warehouse",
		},
	}
	testCases := []struct {
		name       string
		reqs       []kargoapi.FreightRequest
		assertions func(*testing.T, []kargoapi.FreightRequest, field.ErrorList)
	}{
		{
			name: "Freight origin found multiple times",
			reqs: []kargoapi.FreightRequest{
				testFreightRequest,
				testFreightRequest,
			},
			assertions: func(t *testing.T, reqs []kargoapi.FreightRequest, errs field.ErrorList) {
				require.Equal(
					t,
					field.ErrorList{
						{
							Type:     field.ErrorTypeInvalid,
							Field:    "requestedFreight",
							BadValue: reqs,
							Detail: `freight with origin Warehouse/test-warehouse requested ` +
								"multiple times in requestedFreight",
						},
					},
					errs,
				)
			},
		},

		{
			name: "success",
			reqs: []kargoapi.FreightRequest{
				testFreightRequest,
			},
			assertions: func(t *testing.T, _ []kargoapi.FreightRequest, errs field.ErrorList) {
				require.Nil(t, errs)
			},
		},
	}
	w := &webhook{}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				t,
				testCase.reqs,
				w.validateRequestedFreight(
					field.NewPath("requestedFreight"),
					testCase.reqs,
				),
			)
		})
	}
}

func TestValidatePromotionTemplate(t *testing.T) {
	testCases := []struct {
		name          string
		promoTemplate *kargoapi.PromotionTemplate
		assertions    func(*testing.T, field.ErrorList)
	}{
		{
			name: "promotionTemplate is nil",
			assertions: func(t *testing.T, errs field.ErrorList) {
				require.Empty(t, errs)
			},
		},
		{
			name: "promotionTemplate is valid",
			promoTemplate: &kargoapi.PromotionTemplate{
				Spec: kargoapi.PromotionTemplateSpec{
					Steps: []kargoapi.PromotionStep{
						{},
						{As: "fake-step"},
					},
				},
			},
			assertions: func(t *testing.T, errs field.ErrorList) {
				require.Empty(t, errs)
			},
		},
		{
			name: "promotionTemplate is invalid",
			promoTemplate: &kargoapi.PromotionTemplate{
				Spec: kargoapi.PromotionTemplateSpec{
					Steps: []kargoapi.PromotionStep{
						{},
						{As: "step-42"}, // This step alias matches a reserved pattern
					},
				},
			},
			assertions: func(t *testing.T, errs field.ErrorList) {
				require.Equal(
					t,
					field.ErrorList{
						{
							Type:     field.ErrorTypeInvalid,
							Field:    "promotionTemplate.spec.steps[1].as",
							BadValue: "step-42",
							Detail:   "step alias is reserved",
						},
					},
					errs,
				)
			},
		},
	}
	w := &webhook{}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				t,
				w.ValidatePromotionTemplate(
					field.NewPath("promotionTemplate"),
					testCase.promoTemplate,
				),
			)
		})
	}
}
