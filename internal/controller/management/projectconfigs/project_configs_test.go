package projectconfigs

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/logging"
	"github.com/akuity/kargo/internal/webhook/external"
)

func TestNewReconciler(t *testing.T) {
	testCfg := ReconcilerConfig{}
	r := newReconciler(fake.NewClientBuilder().Build(), testCfg)
	require.Equal(t, testCfg, r.cfg)
	require.NotNil(t, r.client)
	require.NotNil(t, r.syncWebhookReceivers)
}

func TestReconciler_syncProjectConfig(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	require.NoError(t, kargoapi.AddToScheme(scheme))

	for _, test := range []struct {
		name          string
		projectConfig *kargoapi.ProjectConfig
		reconciler    func() *reconciler
		assertions    func(*testing.T, kargoapi.ProjectConfigStatus, error)
	}{
		{
			name: "failure",
			reconciler: func() *reconciler {
				r := newReconciler(
					fake.NewClientBuilder().
						WithScheme(scheme).
						WithObjects(
							&corev1.Secret{
								ObjectMeta: metav1.ObjectMeta{
									Name:      "secret-that-exists",
									Namespace: "fake-namespace",
								},
								Data: map[string][]byte{
									"token": []byte("fake-secret-data"),
								},
							},
						).
						Build(),
					ReconcilerConfig{},
				)
				r.syncWebhookReceiversFn = func(
					_ context.Context,
					_ *kargoapi.ProjectConfig,
				) ([]kargoapi.WebhookReceiver, error) {
					return nil, fmt.Errorf("secret not found")
				}
				return r
			},
			projectConfig: &kargoapi.ProjectConfig{
				Status: kargoapi.ProjectConfigStatus{
					WebhookReceivers: []kargoapi.WebhookReceiver{},
				},
			},
			assertions: func(t *testing.T, pcs kargoapi.ProjectConfigStatus, err error) {
				require.Error(t, err)
				require.Len(t, pcs.WebhookReceivers, 0)
				require.Len(t, pcs.Conditions, 2)
				require.Equal(t, pcs.Conditions[0].Type, kargoapi.ConditionTypeReconciling)
				require.Equal(t, pcs.Conditions[0].Status, metav1.ConditionTrue)
				require.Equal(t, pcs.Conditions[1].Type, kargoapi.ConditionTypeReady)
				require.Equal(t, pcs.Conditions[1].Status, metav1.ConditionFalse)
			},
		},
		{
			name: "success",
			reconciler: func() *reconciler {
				return newReconciler(
					fake.NewClientBuilder().
						WithScheme(scheme).
						WithObjects(
							&corev1.Secret{
								ObjectMeta: metav1.ObjectMeta{
									Name:      "secret-that-exists",
									Namespace: "fake-namespace",
								},
								Data: map[string][]byte{
									"token": []byte("fake-secret-data"),
								},
							},
						).
						Build(),
					ReconcilerConfig{},
				)
			},
			projectConfig: &kargoapi.ProjectConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "fake-project-config",
					Namespace: "fake-namespace",
				},
				Spec: kargoapi.ProjectConfigSpec{
					WebhookReceivers: []kargoapi.WebhookReceiverConfig{
						{
							GitHub: &kargoapi.GitHubWebhookReceiver{
								SecretRef: corev1.LocalObjectReference{
									Name: "secret-that-exists",
								},
							},
						},
					},
				},
				Status: kargoapi.ProjectConfigStatus{
					WebhookReceivers: []kargoapi.WebhookReceiver{},
				},
			},
			assertions: func(t *testing.T, pcs kargoapi.ProjectConfigStatus, err error) {
				require.NoError(t, err)
				require.Len(t, pcs.WebhookReceivers, 1)
				require.Len(t, pcs.Conditions, 1)
				require.Equal(t, pcs.Conditions[0].Type, kargoapi.ConditionTypeReady)
				require.Equal(t, pcs.Conditions[0].Status, metav1.ConditionTrue)
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			r := test.reconciler()
			l := logging.NewLogger(logging.DebugLevel)
			ctx := logging.ContextWithLogger(t.Context(), l)
			status, err := r.syncProjectConfig(ctx, test.projectConfig)
			test.assertions(t, status, err)
		})
	}
}

func TestReconciler_syncWebhookReceivers(t *testing.T) {
	for _, test := range []struct {
		name          string
		reconciler    func() *reconciler
		projectConfig *kargoapi.ProjectConfig
		assertions    func(*testing.T, *kargoapi.ProjectConfig, error)
	}{
		{
			name: "secret-ref not found",
			reconciler: func() *reconciler {
				scheme := runtime.NewScheme()
				require.NoError(t, corev1.AddToScheme(scheme))
				require.NoError(t, kargoapi.AddToScheme(scheme))
				return newReconciler(
					fake.NewClientBuilder().
						WithScheme(scheme).
						WithObjects(
							&kargoapi.ProjectConfig{
								ObjectMeta: metav1.ObjectMeta{
									Name:      "fake-project",
									Namespace: "fake-namespace",
								},
								Spec: kargoapi.ProjectConfigSpec{
									WebhookReceivers: []kargoapi.WebhookReceiverConfig{
										{
											GitHub: &kargoapi.GitHubWebhookReceiver{
												SecretRef: corev1.LocalObjectReference{
													Name: "secret-ref-that-does-not-exist",
												},
											},
										},
									},
								},
							},
						).
						Build(),
					ReconcilerConfig{},
				)
			},
			projectConfig: &kargoapi.ProjectConfig{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-namespace",
					Name:      "fake-project",
				},
				Spec: kargoapi.ProjectConfigSpec{
					WebhookReceivers: []kargoapi.WebhookReceiverConfig{
						{
							GitHub: &kargoapi.GitHubWebhookReceiver{
								SecretRef: corev1.LocalObjectReference{
									Name: "secret-that-does-not-exist",
								},
							},
						},
					},
				},
			},
			assertions: func(t *testing.T, _ *kargoapi.ProjectConfig, err error) {
				require.ErrorContains(t, err, "not found")
			},
		},
		{
			name: "success - github",
			reconciler: func() *reconciler {
				scheme := runtime.NewScheme()
				require.NoError(t, corev1.AddToScheme(scheme))
				require.NoError(t, kargoapi.AddToScheme(scheme))
				return newReconciler(
					fake.NewClientBuilder().
						WithScheme(scheme).
						WithObjects(
							&kargoapi.ProjectConfig{
								ObjectMeta: metav1.ObjectMeta{
									Name:      "fake-project",
									Namespace: "fake-namespace",
								},
								Spec: kargoapi.ProjectConfigSpec{
									WebhookReceivers: []kargoapi.WebhookReceiverConfig{
										{
											GitHub: &kargoapi.GitHubWebhookReceiver{
												SecretRef: corev1.LocalObjectReference{
													Name: "secret-that-exists",
												},
											},
										},
									},
								},
							},
							&corev1.Secret{
								ObjectMeta: metav1.ObjectMeta{
									Name:      "secret-that-exists",
									Namespace: "fake-namespace",
								},
								Data: map[string][]byte{
									kargoapi.WebhookReceiverSecretKeyGithub: []byte("fake-secret-data"),
								},
							},
						).
						Build(),
					ReconcilerConfig{},
				)
			},
			projectConfig: &kargoapi.ProjectConfig{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-namespace",
					Name:      "fake-project",
				},
				Spec: kargoapi.ProjectConfigSpec{
					WebhookReceivers: []kargoapi.WebhookReceiverConfig{
						{
							Name: "fake-webhook-receiver-name",
							GitHub: &kargoapi.GitHubWebhookReceiver{
								SecretRef: corev1.LocalObjectReference{
									Name: "secret-that-exists",
								},
							},
						},
					},
				},
			},
			assertions: func(t *testing.T, pc *kargoapi.ProjectConfig, err error) {
				require.NoError(t, err)
				require.Len(t, pc.Status.WebhookReceivers, 1)
				require.NotNil(t, pc.Spec.WebhookReceivers[0].GitHub)
				require.Equal(t,
					external.GenerateWebhookPath(
						"fake-webhook-receiver-name",
						pc.Name,
						kargoapi.WebhookReceiverTypeGitHub,
						"fake-secret-data",
					),
					pc.Status.WebhookReceivers[0].Path,
				)
			},
		},
		{
			name: "success - quay",
			reconciler: func() *reconciler {
				scheme := runtime.NewScheme()
				require.NoError(t, corev1.AddToScheme(scheme))
				require.NoError(t, kargoapi.AddToScheme(scheme))
				return newReconciler(
					fake.NewClientBuilder().
						WithScheme(scheme).
						WithObjects(
							&kargoapi.ProjectConfig{
								ObjectMeta: metav1.ObjectMeta{
									Name:      "fake-project",
									Namespace: "fake-namespace",
								},
								Spec: kargoapi.ProjectConfigSpec{
									WebhookReceivers: []kargoapi.WebhookReceiverConfig{
										{
											Quay: &kargoapi.QuayWebhookReceiver{
												SecretRef: corev1.LocalObjectReference{
													Name: "secret-that-exists",
												},
											},
										},
									},
								},
							},
							&corev1.Secret{
								ObjectMeta: metav1.ObjectMeta{
									Name:      "secret-that-exists",
									Namespace: "fake-namespace",
								},
								Data: map[string][]byte{
									kargoapi.WebhookReceiverSecretKeyQuay: []byte("fake-secret-data"),
								},
							},
						).
						Build(),
					ReconcilerConfig{},
				)
			},
			projectConfig: &kargoapi.ProjectConfig{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-namespace",
					Name:      "fake-project",
				},
				Spec: kargoapi.ProjectConfigSpec{
					WebhookReceivers: []kargoapi.WebhookReceiverConfig{
						{
							Name: "fake-webhook-receiver-name",
							Quay: &kargoapi.QuayWebhookReceiver{
								SecretRef: corev1.LocalObjectReference{
									Name: "secret-that-exists",
								},
							},
						},
					},
				},
			},
			assertions: func(t *testing.T, pc *kargoapi.ProjectConfig, err error) {
				require.NoError(t, err)
				require.Len(t, pc.Status.WebhookReceivers, 1)
				require.NotNil(t, pc.Spec.WebhookReceivers[0].Quay)
				require.Equal(t,
					external.GenerateWebhookPath(
						"fake-webhook-receiver-name",
						pc.Name,
						kargoapi.WebhookReceiverTypeQuay,
						"fake-secret-data",
					),
					pc.Status.WebhookReceivers[0].Path,
				)
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			r := test.reconciler()
			l := logging.NewLogger(logging.DebugLevel)
			ctx := logging.ContextWithLogger(t.Context(), l)
			whReceivers, err := r.syncWebhookReceiversFn(ctx, test.projectConfig)
			test.projectConfig.Status.WebhookReceivers = whReceivers
			test.assertions(t, test.projectConfig, err)
		})
	}
}
