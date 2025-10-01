package stage

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	admissionv1 "k8s.io/api/admission/v1"
	authnv1 "k8s.io/api/authentication/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/api"
	libWebhook "github.com/akuity/kargo/pkg/webhook/kubernetes"
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
	require.NotNil(t, w.validateSpecFn)
	require.NotNil(t, w.isRequestFromKargoControlplaneFn)
}

func Test_webhook_Default(t *testing.T) {
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
				require.Equal(t, testShardName, stage.Labels[kargoapi.LabelKeyShard])
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
						kargoapi.LabelKeyShard: testShardName,
					},
				},
			},
			assertions: func(t *testing.T, stage *kargoapi.Stage, err error) {
				require.NoError(t, err)
				require.Empty(t, stage.Spec.Shard)
				_, ok := stage.Labels[kargoapi.LabelKeyShard]
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

func Test_webhook_ValidateCreate(t *testing.T) {
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
					client.Object,
				) error {
					return errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, err error) {
				require.Error(t, err)
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
			name: "error validating spec",
			webhook: &webhook{
				validateProjectFn: func(
					context.Context,
					client.Client,
					client.Object,
				) error {
					return nil
				},
				validateSpecFn: func(*field.Path, kargoapi.StageSpec) field.ErrorList {
					return field.ErrorList{
						field.Invalid(field.NewPath(""), "", "something went wrong"),
					}
				},
			},
			assertions: func(t *testing.T, err error) {
				require.Error(t, err)
				var statusErr *apierrors.StatusError
				require.True(t, errors.As(err, &statusErr))
				require.Equal(t, metav1.StatusReasonInvalid, statusErr.ErrStatus.Reason)
				require.Contains(t, statusErr.ErrStatus.Message, "something went wrong")
			},
		},
		{
			name: "success",
			webhook: &webhook{
				validateProjectFn: func(
					context.Context,
					client.Client,
					client.Object,
				) error {
					return nil
				},
				validateSpecFn: func(*field.Path, kargoapi.StageSpec) field.ErrorList {
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
			_, err := testCase.webhook.ValidateCreate(
				context.Background(),
				&kargoapi.Stage{},
			)
			testCase.assertions(t, err)
		})
	}
}

func Test_webhook_ValidateUpdate(t *testing.T) {
	testCases := []struct {
		name       string
		webhook    *webhook
		assertions func(*testing.T, error)
	}{
		{
			name: "error validating spec",
			webhook: &webhook{
				validateSpecFn: func(*field.Path, kargoapi.StageSpec) field.ErrorList {
					return field.ErrorList{
						field.Invalid(field.NewPath(""), "", "something went wrong"),
					}
				},
			},
			assertions: func(t *testing.T, err error) {
				require.Error(t, err)
				var statusErr *apierrors.StatusError
				require.True(t, errors.As(err, &statusErr))
				require.Equal(t, metav1.StatusReasonInvalid, statusErr.ErrStatus.Reason)
				require.Contains(t, statusErr.ErrStatus.Message, "something went wrong")
			},
		},
		{
			name: "success",
			webhook: &webhook{
				validateSpecFn: func(*field.Path, kargoapi.StageSpec) field.ErrorList {
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
			_, err := testCase.webhook.ValidateUpdate(
				context.Background(),
				nil,
				&kargoapi.Stage{},
			)
			testCase.assertions(t, err)
		})
	}
}

func Test_webhook_ValidateDelete(t *testing.T) {
	w := &webhook{}
	_, err := w.ValidateDelete(context.Background(), nil)
	require.NoError(t, err)
}

func Test_webhook_ValidateSpec(t *testing.T) {
	testFreightRequest := kargoapi.FreightRequest{
		Origin: kargoapi.FreightOrigin{
			Kind: kargoapi.FreightOriginKindWarehouse,
			Name: "test-warehouse",
		},
	}
	testCases := []struct {
		name       string
		spec       kargoapi.StageSpec
		assertions func(*testing.T, kargoapi.StageSpec, field.ErrorList)
	}{
		{
			name: "invalid",
			spec: kargoapi.StageSpec{
				// Has multiple sources for one Freight origin...
				RequestedFreight: []kargoapi.FreightRequest{
					testFreightRequest,
					testFreightRequest,
				},
				PromotionTemplate: &kargoapi.PromotionTemplate{
					Spec: kargoapi.PromotionTemplateSpec{
						Steps: []kargoapi.PromotionStep{
							{As: "step-42"}, // This step alias matches a reserved pattern
							{As: "commit"},
							{As: "commit"}, // Duplicate!
						},
					},
				},
			},
			assertions: func(t *testing.T, spec kargoapi.StageSpec, errs field.ErrorList) {
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
						{
							Type:     field.ErrorTypeInvalid,
							Field:    "spec.promotionTemplate.spec.steps[2].as",
							BadValue: "commit",
							Detail:   "step alias duplicates that of spec.promotionTemplate.spec.steps[1]",
						},
					},
					errs,
				)
			},
		},
		{
			name: "valid",
			spec: kargoapi.StageSpec{
				RequestedFreight: []kargoapi.FreightRequest{
					testFreightRequest,
				},
				PromotionTemplate: &kargoapi.PromotionTemplate{
					Spec: kargoapi.PromotionTemplateSpec{
						Steps: []kargoapi.PromotionStep{
							{As: "foo"},
							{As: "bar"},
							{As: "baz"},
							{As: ""},
							{As: ""}, // optional not dup
						},
					},
				},
			},
			assertions: func(t *testing.T, _ kargoapi.StageSpec, errs field.ErrorList) {
				require.Nil(t, errs)
			},
		},
	}
	w := &webhook{}
	w.validatePromotionStepTaskRefsFn = w.validatePromotionStepTaskRefs
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

func Test_webhook_validateRequestedFreight(t *testing.T) {
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
				// Should only be warned once
				testFreightRequest,
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
			name: "issues validating Freight sources are surfaced",
			reqs: []kargoapi.FreightRequest{{
				Origin: testFreightRequest.Origin,
				Sources: kargoapi.FreightSources{
					Direct: true,
					AutoPromotionOptions: &kargoapi.AutoPromotionOptions{
						SelectionPolicy: kargoapi.AutoPromotionSelectionPolicyMatchUpstream,
					},
				},
			}},
			assertions: func(t *testing.T, _ []kargoapi.FreightRequest, errs field.ErrorList) {
				require.Equal(
					t,
					field.ErrorList{
						{
							Type:     field.ErrorTypeInvalid,
							Field:    "requestedFreight[0].sources.autoPromotionOptions.selectionPolicy",
							BadValue: kargoapi.AutoPromotionSelectionPolicyMatchUpstream,
							Detail: "selection policy 'MatchUpstream' cannot be used when " +
								"accepting Freight directly from its origin",
						},
						{
							Type:     field.ErrorTypeInvalid,
							Field:    "requestedFreight[0].sources.autoPromotionOptions.selectionPolicy",
							BadValue: kargoapi.AutoPromotionSelectionPolicyMatchUpstream,
							Detail: "selection policy 'MatchUpstream' requires exactly one " +
								"upstream Stage to be specified",
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

func Test_webhook_validateFreightSources(t *testing.T) {
	testCases := []struct {
		name       string
		sources    kargoapi.FreightSources
		assertions func(*testing.T, field.ErrorList)
	}{
		{
			name: "MatchUpstream auto-promotion selection policy used with direct source",
			sources: kargoapi.FreightSources{
				Direct: true,
				Stages: []string{"stage-1"},
				AutoPromotionOptions: &kargoapi.AutoPromotionOptions{
					SelectionPolicy: kargoapi.AutoPromotionSelectionPolicyMatchUpstream,
				},
			},
			assertions: func(t *testing.T, errs field.ErrorList) {
				require.Equal(
					t,
					field.ErrorList{{
						Type:     field.ErrorTypeInvalid,
						Field:    "sources.autoPromotionOptions.selectionPolicy",
						BadValue: kargoapi.AutoPromotionSelectionPolicyMatchUpstream,
						Detail: "selection policy 'MatchUpstream' cannot be used when " +
							"accepting Freight directly from its origin",
					}},
					errs,
				)
			},
		},
		{
			name: "MatchUpstream auto-promotion selection policy used with no upstream sources",
			sources: kargoapi.FreightSources{
				AutoPromotionOptions: &kargoapi.AutoPromotionOptions{
					SelectionPolicy: kargoapi.AutoPromotionSelectionPolicyMatchUpstream,
				},
			},
			assertions: func(t *testing.T, errs field.ErrorList) {
				require.Equal(
					t,
					field.ErrorList{{
						Type:     field.ErrorTypeInvalid,
						Field:    "sources.autoPromotionOptions.selectionPolicy",
						BadValue: kargoapi.AutoPromotionSelectionPolicyMatchUpstream,
						Detail: "selection policy 'MatchUpstream' requires exactly one upstream " +
							"Stage to be specified",
					}},
					errs,
				)
			},
		},
		{
			name: "MatchUpstream auto-promotion selection policy used with multiple upstream sources",
			sources: kargoapi.FreightSources{
				Stages: []string{"stage-1", "stage-2"},
				AutoPromotionOptions: &kargoapi.AutoPromotionOptions{
					SelectionPolicy: kargoapi.AutoPromotionSelectionPolicyMatchUpstream,
				},
			},
			assertions: func(t *testing.T, errs field.ErrorList) {
				require.Equal(
					t,
					field.ErrorList{{
						Type:     field.ErrorTypeInvalid,
						Field:    "sources.autoPromotionOptions.selectionPolicy",
						BadValue: kargoapi.AutoPromotionSelectionPolicyMatchUpstream,
						Detail: "selection policy 'MatchUpstream' requires exactly one upstream " +
							"Stage to be specified",
					}},
					errs,
				)
			},
		},
		{
			name: "MatchUpstream auto-promotion selection policy used with multiple problems",
			sources: kargoapi.FreightSources{
				Direct: true,
				Stages: []string{"stage-1", "stage-2"},
				AutoPromotionOptions: &kargoapi.AutoPromotionOptions{
					SelectionPolicy: kargoapi.AutoPromotionSelectionPolicyMatchUpstream,
				},
			},
			assertions: func(t *testing.T, errs field.ErrorList) {
				require.Equal(
					t,
					field.ErrorList{
						{
							Type:     field.ErrorTypeInvalid,
							Field:    "sources.autoPromotionOptions.selectionPolicy",
							BadValue: kargoapi.AutoPromotionSelectionPolicyMatchUpstream,
							Detail: "selection policy 'MatchUpstream' cannot be used when " +
								"accepting Freight directly from its origin",
						},
						{
							Type:     field.ErrorTypeInvalid,
							Field:    "sources.autoPromotionOptions.selectionPolicy",
							BadValue: kargoapi.AutoPromotionSelectionPolicyMatchUpstream,
							Detail: "selection policy 'MatchUpstream' requires exactly one upstream " +
								"Stage to be specified",
						},
					},
					errs,
				)
			},
		},
		{
			name: "valid Freight sources with default auto-promotion selection policy",
			sources: kargoapi.FreightSources{
				Direct: true,
				Stages: []string{"stage-1", "stage-2"},
			},
			assertions: func(t *testing.T, errs field.ErrorList) {
				require.Nil(t, errs)
			},
		},
		{
			name: "valid Freight sources with MatchUpstream auto-promotion selection policy",
			sources: kargoapi.FreightSources{
				Stages: []string{"stage-1"},
				AutoPromotionOptions: &kargoapi.AutoPromotionOptions{
					SelectionPolicy: kargoapi.AutoPromotionSelectionPolicyMatchUpstream,
				},
			},
			assertions: func(t *testing.T, errs field.ErrorList) {
				require.Nil(t, errs)
			},
		},
	}
	w := &webhook{}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				t,
				w.validateFreightSources(field.NewPath("sources"), testCase.sources),
			)
		})
	}
}

func Test_webhook_validatePromotionStepTaskRefs(t *testing.T) {
	testCases := []struct {
		name       string
		steps      []kargoapi.PromotionStep
		assertions func(*testing.T, field.ErrorList)
	}{
		{
			name: "step with task reference and 'if' condition should fail",
			steps: []kargoapi.PromotionStep{
				{
					As: "test-step",
					Task: &kargoapi.PromotionTaskReference{
						Name: "test-task",
					},
					If: "some-condition",
				},
			},
			assertions: func(t *testing.T, errs field.ErrorList) {
				require.Len(t, errs, 1)
				require.Equal(t, field.ErrorTypeForbidden, errs[0].Type)
				require.Equal(t, "steps[0].if", errs[0].Field)
				require.Equal(t, "PromotionTemplate step referencing a task cannot have an 'if' condition", errs[0].Detail)
			},
		},
		{
			name: "step with task reference and config should fail",
			steps: []kargoapi.PromotionStep{
				{
					As: "test-step",
					Task: &kargoapi.PromotionTaskReference{
						Name: "test-task",
					},
					Config: &apiextensionsv1.JSON{
						Raw: []byte(`{"key": "value"}`),
					},
				},
			},
			assertions: func(t *testing.T, errs field.ErrorList) {
				require.Len(t, errs, 1)
				require.Equal(t, field.ErrorTypeForbidden, errs[0].Type)
				require.Equal(t, "steps[0].config", errs[0].Field)
				require.Equal(t, "PromotionTemplate step referencing a task cannot have a config", errs[0].Detail)
			},
		},
		{
			name: "step with task reference, 'if' condition and config should fail with both errors",
			steps: []kargoapi.PromotionStep{
				{
					As: "test-step",
					Task: &kargoapi.PromotionTaskReference{
						Name: "test-task",
					},
					If: "some-condition",
					Config: &apiextensionsv1.JSON{
						Raw: []byte(`{"key": "value"}`),
					},
				},
			},
			assertions: func(t *testing.T, errs field.ErrorList) {
				require.Len(t, errs, 2)
				require.Equal(t, field.ErrorTypeForbidden, errs[0].Type)
				require.Equal(t, "steps[0].if", errs[0].Field)
				require.Equal(t, "PromotionTemplate step referencing a task cannot have an 'if' condition", errs[0].Detail)
				require.Equal(t, field.ErrorTypeForbidden, errs[1].Type)
				require.Equal(t, "steps[0].config", errs[1].Field)
				require.Equal(t, "PromotionTemplate step referencing a task cannot have a config", errs[1].Detail)
			},
		},
		{
			name: "multiple steps with task references and violations",
			steps: []kargoapi.PromotionStep{
				{
					As: "step-1",
					Task: &kargoapi.PromotionTaskReference{
						Name: "task-1",
					},
					If: "condition-1",
				},
				{
					As: "step-2",
					Task: &kargoapi.PromotionTaskReference{
						Name: "task-2",
					},
					Config: &apiextensionsv1.JSON{
						Raw: []byte(`{"key": "value"}`),
					},
				},
			},
			assertions: func(t *testing.T, errs field.ErrorList) {
				require.Len(t, errs, 2)
				require.Equal(t, field.ErrorTypeForbidden, errs[0].Type)
				require.Equal(t, "steps[0].if", errs[0].Field)
				require.Equal(t, "PromotionTemplate step referencing a task cannot have an 'if' condition", errs[0].Detail)
				require.Equal(t, field.ErrorTypeForbidden, errs[1].Type)
				require.Equal(t, "steps[1].config", errs[1].Field)
				require.Equal(t, "PromotionTemplate step referencing a task cannot have a config", errs[1].Detail)
			},
		},
		{
			name: "step with task reference but no violations should pass",
			steps: []kargoapi.PromotionStep{
				{
					As: "test-step",
					Task: &kargoapi.PromotionTaskReference{
						Name: "test-task",
					},
				},
			},
			assertions: func(t *testing.T, errs field.ErrorList) {
				require.Empty(t, errs)
			},
		},
		{
			name: "step without task reference can have 'if' condition and config",
			steps: []kargoapi.PromotionStep{
				{
					As:   "test-step",
					Uses: "some-action",
					If:   "some-condition",
					Config: &apiextensionsv1.JSON{
						Raw: []byte(`{"key": "value"}`),
					},
				},
			},
			assertions: func(t *testing.T, errs field.ErrorList) {
				require.Empty(t, errs)
			},
		},
		{
			name: "mixed steps - some with task references, some without",
			steps: []kargoapi.PromotionStep{
				{
					As: "step-with-task",
					Task: &kargoapi.PromotionTaskReference{
						Name: "test-task",
					},
				},
				{
					As:   "step-without-task",
					Uses: "some-action",
					If:   "some-condition",
					Config: &apiextensionsv1.JSON{
						Raw: []byte(`{"key": "value"}`),
					},
				},
				{
					As: "step-with-task-violation",
					Task: &kargoapi.PromotionTaskReference{
						Name: "another-task",
					},
					If: "another-condition",
				},
			},
			assertions: func(t *testing.T, errs field.ErrorList) {
				require.Len(t, errs, 1)
				require.Equal(t, field.ErrorTypeForbidden, errs[0].Type)
				require.Equal(t, "steps[2].if", errs[0].Field)
				require.Equal(t, "PromotionTemplate step referencing a task cannot have an 'if' condition", errs[0].Detail)
			},
		},
		{
			name:  "empty steps should pass",
			steps: []kargoapi.PromotionStep{},
			assertions: func(t *testing.T, errs field.ErrorList) {
				require.Empty(t, errs)
			},
		},
	}

	w := &webhook{}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				t,
				w.validatePromotionStepTaskRefs(
					field.NewPath("steps"),
					testCase.steps,
				),
			)
		})
	}
}
