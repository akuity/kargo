package projectconfigs

import (
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
	require.NotNil(t, r.ensureWebhookReceivers)
	require.NotNil(t, r.reconcileFn)
}

func TestReconciler_ensureReceivers(t *testing.T) {
	for _, test := range []struct {
		name          string
		reconciler    func() *reconciler
		projectConfig *kargoapi.ProjectConfig
		assertions    func(*testing.T, *kargoapi.ProjectConfig, []kargoapi.WebhookReceiver, error)
	}{
		{
			name: "project config not found",
			reconciler: func() *reconciler {
				scheme := runtime.NewScheme()
				require.NoError(t, corev1.AddToScheme(scheme))
				require.NoError(t, kargoapi.AddToScheme(scheme))
				return newReconciler(
					fake.NewClientBuilder().WithScheme(scheme).Build(),
					ReconcilerConfig{},
				)
			},
			projectConfig: &kargoapi.ProjectConfig{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-namespace",
					Name:      "fake-project",
				},
			},
			assertions: func(t *testing.T, _ *kargoapi.ProjectConfig, _ []kargoapi.WebhookReceiver, err error) {
				require.ErrorContains(t, err, "error getting ProjectConfig")
			},
		},
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
									WebhookReceiverConfigs: []kargoapi.WebhookReceiverConfig{
										{
											Type:      kargoapi.WebhookReceiverTypeGitHub,
											SecretRef: "secret-ref-that-does-not-exist",
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
					WebhookReceiverConfigs: []kargoapi.WebhookReceiverConfig{
						{
							Type:      kargoapi.WebhookReceiverTypeGitHub,
							SecretRef: "secret-ref-that-does-not-exist",
						},
					},
				},
			},
			assertions: func(t *testing.T, _ *kargoapi.ProjectConfig, _ []kargoapi.WebhookReceiver, err error) {
				require.ErrorContains(t, err, "not found")
			},
		},
		{
			name: "success",
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
									Name:      "fake-name",
									Namespace: "fake-namespace",
								},
								Spec: kargoapi.ProjectConfigSpec{},
							},
							&kargoapi.ProjectConfig{
								ObjectMeta: metav1.ObjectMeta{
									Name:      "fake-project",
									Namespace: "fake-namespace",
								},
								Spec: kargoapi.ProjectConfigSpec{
									WebhookReceiverConfigs: []kargoapi.WebhookReceiverConfig{
										{
											Type:      kargoapi.WebhookReceiverTypeGitHub,
											SecretRef: "secret-that-exists",
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
									"seed": []byte("fake-secret-data"),
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
					WebhookReceiverConfigs: []kargoapi.WebhookReceiverConfig{
						{
							Type:      kargoapi.WebhookReceiverTypeGitHub,
							SecretRef: "secret-that-exists",
						},
					},
				},
			},
			assertions: func(t *testing.T, pc *kargoapi.ProjectConfig, whReceivers []kargoapi.WebhookReceiver, err error) {
				require.NoError(t, err)
				require.Len(t, whReceivers, 1)
				require.Equal(t,
					kargoapi.WebhookReceiverTypeGitHub,
					pc.Spec.WebhookReceiverConfigs[0].Type, // nolint: staticcheck
				)
				require.Equal(t,
					external.GenerateWebhookPath(
						pc.Name,
						kargoapi.WebhookReceiverTypeGitHub,
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
			whReceivers, err := r.ensureWebhookReceiversFn(ctx, test.projectConfig)
			test.assertions(t, test.projectConfig, whReceivers, err)
		})
	}
}
