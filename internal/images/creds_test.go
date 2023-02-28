package images

import (
	"context"
	"testing"

	"github.com/argoproj-labs/argocd-image-updater/pkg/image"
	"github.com/argoproj-labs/argocd-image-updater/pkg/registry"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

func TestGetRegistryCredentials(t *testing.T) {
	testCases := []struct {
		name            string
		pullSecret      string
		setupKubeClient func(kubernetes.Interface)
		assertions      func(image.Credential, error)
	}{
		{
			name: "success with no image pull secret",
			assertions: func(creds image.Credential, err error) {
				require.NoError(t, err)
				// Username and Password should be blank
				require.Equal(t, image.Credential{}, creds)
			},
		},

		{
			name:       "error getting image pull secret",
			pullSecret: "fake-pull-secret",
			assertions: func(_ image.Credential, err error) {
				require.Error(t, err)
				require.Contains(
					t,
					err.Error(),
					"error getting credentials for image",
				)
				require.Contains(t, err.Error(), "from image pull secret")
				require.Contains(t, err.Error(), "could not fetch secret")
			},
		},

		{
			name:       "success with image pull secret",
			pullSecret: "fake-pull-secret",
			setupKubeClient: func(kubeClient kubernetes.Interface) {
				_, err := kubeClient.CoreV1().Secrets("argo-cd").Create(
					context.Background(),
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Name: "fake-pull-secret",
						},
						Type: corev1.SecretTypeDockerConfigJson,
						Data: map[string][]byte{
							// These are properly formatted, but completely made up
							// credentials
							".dockerconfigjson": []byte(`{"auths":{"https://registry-1.docker.io":{"username":"fake-user","password":"fake-password","email":"fake.user@example.com","auth":"ZmFrZS11c2VyOmZha2UtcGFzc3dvcmQ="}}}`), // nolint: lll
						},
					},
					metav1.CreateOptions{},
				)
				require.NoError(t, err)
			},
			assertions: func(creds image.Credential, err error) {
				require.NoError(t, err)
				require.Equal(
					t,
					image.Credential{
						Username: "fake-user",
						Password: "fake-password",
					},
					creds,
				)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			kubeClient := fake.NewSimpleClientset()
			if testCase.setupKubeClient != nil {
				testCase.setupKubeClient(kubeClient)
			}
			testCase.assertions(
				getRegistryCredentials(
					context.Background(),
					kubeClient,
					"fake-url",
					testCase.pullSecret,
					&registry.RegistryEndpoint{},
				),
			)
		})
	}
}
