package external

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func TestNewReceiver(t *testing.T) {
	testScheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(testScheme))

	const testProject = "fake-project"
	const testBaseURL = "https://webhooks.kargo.example.com"

	testCases := []struct {
		name       string
		client     client.Client
		baseURL    string
		cfg        kargoapi.WebhookReceiverConfig
		assertions func(*testing.T, WebhookReceiver, error)
	}{
		{
			name: "no configuration for a known receiver type",
			assertions: func(t *testing.T, _ WebhookReceiver, err error) {
				require.EqualError(
					t,
					err,
					"WebhookReceiverConfig has no configuration for a known receiver type",
				)
			},
		},
		{
			name: "error getting Secret",
			// The Secret doesn't exist
			client: fake.NewClientBuilder().WithScheme(testScheme).Build(),
			cfg: kargoapi.WebhookReceiverConfig{
				GitHub: &kargoapi.GitHubWebhookReceiverConfig{
					SecretRef: corev1.LocalObjectReference{Name: "fake-secret"},
				},
			},
			assertions: func(t *testing.T, _ WebhookReceiver, err error) {
				require.ErrorContains(t, err, "error getting Secret")
			},
		},
		{
			name: "error extracting required Secret values",
			client: fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProject,
						Name:      "fake-secret",
					},
					// No data / doesn't contain the required key
				},
			).Build(),
			cfg: kargoapi.WebhookReceiverConfig{
				GitHub: &kargoapi.GitHubWebhookReceiverConfig{
					SecretRef: corev1.LocalObjectReference{Name: "fake-secret"},
				},
			},
			assertions: func(t *testing.T, _ WebhookReceiver, err error) {
				require.ErrorContains(
					t,
					err,
					"error extracting secret values from Secret",
				)
			},
		},
		{
			name: "success",
			client: fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProject,
						Name:      "fake-secret",
					},
					Data: map[string][]byte{GithubSecretDataKey: []byte("my-super-secret-token")},
				},
			).Build(),
			cfg: kargoapi.WebhookReceiverConfig{
				Name: "fake-github-receiver",
				GitHub: &kargoapi.GitHubWebhookReceiverConfig{
					SecretRef: corev1.LocalObjectReference{Name: "fake-secret"},
				},
			},
			assertions: func(t *testing.T, receiver WebhookReceiver, err error) {
				require.NoError(t, err)
				r, ok := receiver.(*githubWebhookReceiver)
				require.True(t, ok)
				require.NotNil(t, r.client)
				require.Equal(t, testProject, r.project)
				require.Equal(t, "fake-secret", r.secretName)
				require.Equal(t, "fake-github-receiver", r.details.Name)
				require.NotEmpty(t, r.details.Path)
				require.NotEmpty(t, r.details.URL)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			receiver, err := NewReceiver(
				context.Background(),
				testCase.client,
				testBaseURL,
				testProject,
				testProject,
				testCase.cfg,
			)
			testCase.assertions(t, receiver, err)
		})
	}
}
