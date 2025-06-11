package clusterconfigs

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/conditions"
	"github.com/akuity/kargo/internal/webhook/external"
)

func TestNewReconciler(t *testing.T) {
	testCfg := ReconcilerConfig{}
	r := newReconciler(fake.NewClientBuilder().Build(), testCfg)
	require.Equal(t, testCfg, r.cfg)
	require.NotNil(t, r.client)
}

func TestReconciler_syncWebhookReceivers(t *testing.T) {
	testScheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(testScheme))

	testSecretNamespace := "fake-namespace"

	testCases := []struct {
		name       string
		reconciler *reconciler
		clusterCfg *kargoapi.ClusterConfig
		assertions func(*testing.T, kargoapi.ClusterConfigStatus, error)
	}{
		{
			name:       "cluster config does not define any webhook receivers",
			reconciler: &reconciler{},
			clusterCfg: &kargoapi.ClusterConfig{},
			assertions: func(t *testing.T, status kargoapi.ClusterConfigStatus, err error) {
				require.NoError(t, err)
				require.Empty(t, status.WebhookReceivers)
				require.Len(t, status.Conditions, 1)
				readyCondition := conditions.Get(&status, kargoapi.ConditionTypeReady)
				require.NotNil(t, readyCondition)
				require.Equal(t, metav1.ConditionTrue, readyCondition.Status)
				require.Equal(t, "Synced", readyCondition.Reason)
			},
		},
		{
			name:       "cluster secrets namespace not specified",
			reconciler: &reconciler{},
			clusterCfg: &kargoapi.ClusterConfig{
				ObjectMeta: metav1.ObjectMeta{Name: "cluster"},
				Spec: kargoapi.ClusterConfigSpec{
					WebhookReceivers: []kargoapi.WebhookReceiverConfig{{}},
				},
			},
			assertions: func(t *testing.T, _ kargoapi.ClusterConfigStatus, err error) {
				require.ErrorContains(
					t,
					err,
					"no namespace is designated for storing Secrets referenced by "+
						"cluster-level resources",
				)
			},
		},
		{
			name: "error building receiver",
			reconciler: &reconciler{
				cfg: ReconcilerConfig{
					ClusterSecretsNamespace: testSecretNamespace,
				},
				client: fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: testSecretNamespace,
							Name:      "fake-token-secret",
						},
						Data: map[string][]byte{external.GithubSecretDataKey: []byte("fake-token")},
					},
				).Build(),
			},
			clusterCfg: &kargoapi.ClusterConfig{
				ObjectMeta: metav1.ObjectMeta{Name: "cluster"},
				Spec: kargoapi.ClusterConfigSpec{
					WebhookReceivers: []kargoapi.WebhookReceiverConfig{
						{
							Name: "invalid-receiver",
							GitHub: &kargoapi.GitHubWebhookReceiverConfig{
								SecretRef: corev1.LocalObjectReference{
									Name: "non-existent-secret",
								},
							},
						},
						{
							Name: "valid-receiver",
							GitHub: &kargoapi.GitHubWebhookReceiverConfig{
								SecretRef: corev1.LocalObjectReference{
									Name: "fake-token-secret",
								},
							},
						},
					},
				},
			},
			assertions: func(t *testing.T, status kargoapi.ClusterConfigStatus, err error) {
				// We should get an error because the first receiver's SecretRef could
				// not be resolved.
				require.ErrorContains(t, err, "not found")

				// But the second receiver should still have been processed.
				require.Len(t, status.WebhookReceivers, 1)
				require.Equal(t, "valid-receiver", status.WebhookReceivers[0].Name)
				require.NotEmpty(t, status.WebhookReceivers[0].Path)
				require.NotEmpty(t, status.WebhookReceivers[0].URL)

				// The conditions should reflect the error and that the ClusterConfig is
				// still syncing.
				require.Len(t, status.Conditions, 2)
				readyCondition := conditions.Get(&status, kargoapi.ConditionTypeReady)
				require.NotNil(t, readyCondition)
				require.Equal(t, metav1.ConditionFalse, readyCondition.Status)
				require.Equal(t, "SyncWebhookReceiversFailed", readyCondition.Reason)
				reconcilingCondition := conditions.Get(&status, kargoapi.ConditionTypeReconciling)
				require.NotNil(t, reconcilingCondition)
				require.Equal(t, metav1.ConditionTrue, reconcilingCondition.Status)
				require.Equal(t, "Syncing WebhookReceivers", reconcilingCondition.Message)
			},
		},
		{
			name: "great success!",
			reconciler: &reconciler{
				cfg: ReconcilerConfig{
					ClusterSecretsNamespace: testSecretNamespace,
				},
				client: fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: testSecretNamespace,
							Name:      "fake-token-secret",
						},
						Data: map[string][]byte{external.GithubSecretDataKey: []byte("fake-token")},
					},
				).Build(),
			},
			clusterCfg: &kargoapi.ClusterConfig{
				ObjectMeta: metav1.ObjectMeta{Name: "cluster"},
				Spec: kargoapi.ClusterConfigSpec{
					WebhookReceivers: []kargoapi.WebhookReceiverConfig{
						{
							Name: "valid-receiver",
							GitHub: &kargoapi.GitHubWebhookReceiverConfig{
								SecretRef: corev1.LocalObjectReference{
									Name: "fake-token-secret",
								},
							},
						},
					},
				},
			},
			assertions: func(t *testing.T, status kargoapi.ClusterConfigStatus, err error) {
				require.NoError(t, err)

				// But the second receiver should still have been processed.
				require.Len(t, status.WebhookReceivers, 1)
				require.Equal(t, "valid-receiver", status.WebhookReceivers[0].Name)
				require.NotEmpty(t, status.WebhookReceivers[0].Path)
				require.NotEmpty(t, status.WebhookReceivers[0].URL)

				// The conditions should reflect success.
				require.Len(t, status.Conditions, 1)
				readyCondition := conditions.Get(&status, kargoapi.ConditionTypeReady)
				require.NotNil(t, readyCondition)
				require.Equal(t, metav1.ConditionTrue, readyCondition.Status)
				require.Equal(t, "Synced", readyCondition.Reason)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			status, err := testCase.reconciler.syncWebhookReceivers(
				context.Background(),
				testCase.clusterCfg,
			)
			testCase.assertions(t, status, err)
		})
	}
}
