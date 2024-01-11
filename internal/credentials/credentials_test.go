package credentials

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestNewKubernetesDatabase(t *testing.T) {
	testClient := fake.NewClientBuilder().Build()
	testCfg := KubernetesDatabaseConfig{
		ArgoCDNamespace:             "fake-namespace",
		GlobalCredentialsNamespaces: []string{"another-fake-namespace"},
	}
	d := NewKubernetesDatabase(testClient, testClient, testCfg)
	require.NotNil(t, d)
	k, ok := d.(*kubernetesDatabase)
	require.True(t, ok)
	require.Same(t, testClient, k.kargoClient)
	require.Same(t, testClient, k.argoClient)
	require.Equal(t, testCfg, k.cfg)
}

func TestGetCredentialsSecret(t *testing.T) {
	const testSecretName = "fake-secret"
	const testNamespace = "fake-namespace"
	testSecretMetadata := metav1.ObjectMeta{
		Name:      testSecretName,
		Namespace: testNamespace,
	}
	testCases := []struct {
		name          string
		secrets       []client.Object
		repoType      Type
		repoURL       string
		prefixMatch   bool
		shouldBeFound bool
	}{
		{
			name:          "no secrets found",
			secrets:       []client.Object{},
			repoType:      TypeGit,
			repoURL:       "https://github.com/example",
			shouldBeFound: false,
		},
		{
			name: "no secrets of correct type",
			secrets: []client.Object{
				&corev1.Secret{
					ObjectMeta: testSecretMetadata,
					Data: map[string][]byte{
						"type": []byte(TypeImage),
						// This is not a realistic URL for an image, but we're trying to
						// prove that type matters.
						"url": []byte("https://github.com/example/example"),
					},
				},
			},
			repoType:      TypeGit,
			repoURL:       "https://github.com/example/example",
			shouldBeFound: false,
		},
		{
			name: "exact git repo URL match",
			secrets: []client.Object{
				&corev1.Secret{
					ObjectMeta: testSecretMetadata,
					Data: map[string][]byte{
						"type": []byte(TypeGit),
						"url":  []byte("https://github.com/example/example"),
					},
				},
			},
			repoType:      TypeGit,
			repoURL:       "https://github.com/example/example",
			shouldBeFound: true,
		},
		{
			name: "normalized git repo URL match",
			secrets: []client.Object{
				&corev1.Secret{
					ObjectMeta: testSecretMetadata,
					Data: map[string][]byte{
						"type": []byte(TypeGit),
						"url":  []byte("https://github.com/example/example"),
					},
				},
			},
			repoType:      TypeGit,
			repoURL:       "https://github.com/example/example.git",
			shouldBeFound: true,
		},
		{
			name: "git repo URL prefix match",
			secrets: []client.Object{
				&corev1.Secret{
					ObjectMeta: testSecretMetadata,
					Data: map[string][]byte{
						"type": []byte(TypeGit),
						"url":  []byte("https://github.com/example"),
					},
				},
			},
			repoType:      TypeGit,
			repoURL:       "https://github.com/example/example",
			prefixMatch:   true,
			shouldBeFound: true,
		},
		{
			name: "exact image repo URL match",
			secrets: []client.Object{
				&corev1.Secret{
					ObjectMeta: testSecretMetadata,
					Data: map[string][]byte{
						"type": []byte(TypeImage),
						"url":  []byte("ghcr.io/example/example"),
					},
				},
			},
			repoType:      TypeImage,
			repoURL:       "ghcr.io/example/example",
			shouldBeFound: true,
		},
		{
			name: "image repo URL prefix match",
			secrets: []client.Object{
				&corev1.Secret{
					ObjectMeta: testSecretMetadata,
					Data: map[string][]byte{
						"type": []byte(TypeImage),
						"url":  []byte("ghcr.io/example"),
					},
				},
			},
			repoType:      TypeImage,
			repoURL:       "ghcr.io/example/example",
			prefixMatch:   true,
			shouldBeFound: true,
		},
		{
			name: "exact chart repo https URL match",
			secrets: []client.Object{
				&corev1.Secret{
					ObjectMeta: testSecretMetadata,
					Data: map[string][]byte{
						"type": []byte(TypeHelm),
						"url":  []byte("https://chart-museum.example.com/example"),
					},
				},
			},
			repoType:      TypeHelm,
			repoURL:       "https://chart-museum.example.com/example",
			shouldBeFound: true,
		},
		{
			name: "chart repo https URL prefix match",
			secrets: []client.Object{
				&corev1.Secret{
					ObjectMeta: testSecretMetadata,
					Data: map[string][]byte{
						"type": []byte(TypeHelm),
						"url":  []byte("https://chart-museum.example.com"),
					},
				},
			},
			repoType:      TypeHelm,
			repoURL:       "https://chart-museum.example.com/example",
			prefixMatch:   true,
			shouldBeFound: true,
		},
		{
			name: "exact chart repo oci URL match",
			secrets: []client.Object{
				&corev1.Secret{
					ObjectMeta: testSecretMetadata,
					Data: map[string][]byte{
						"type": []byte(TypeHelm),
						"url":  []byte("oci://ghcr.io/example/example"),
					},
				},
			},
			repoType:      TypeHelm,
			repoURL:       "oci://ghcr.io/example/example",
			shouldBeFound: true,
		},
		{
			name: "chart repo oci URL prefix match",
			secrets: []client.Object{
				&corev1.Secret{
					ObjectMeta: testSecretMetadata,
					Data: map[string][]byte{
						"type": []byte(TypeHelm),
						"url":  []byte("oci://ghcr.io/example"),
					},
				},
			},
			repoType:      TypeHelm,
			repoURL:       "oci://ghcr.io/example/example",
			prefixMatch:   true,
			shouldBeFound: true,
		},
		{
			name: "normalized chart repo oci URL match",
			secrets: []client.Object{
				&corev1.Secret{
					ObjectMeta: testSecretMetadata,
					Data: map[string][]byte{
						"type": []byte(TypeHelm),
						"url":  []byte("ghcr.io/example/example"),
					},
				},
			},
			repoType:      TypeHelm,
			repoURL:       "oci://ghcr.io/example/example",
			shouldBeFound: true,
		},
		{
			name: "normalized chart repo oci URL prefix match",
			secrets: []client.Object{
				&corev1.Secret{
					ObjectMeta: testSecretMetadata,
					Data: map[string][]byte{
						"type": []byte(TypeHelm),
						"url":  []byte("ghcr.io/example"),
					},
				},
			},
			repoType:      TypeHelm,
			repoURL:       "oci://ghcr.io/example/example",
			prefixMatch:   true,
			shouldBeFound: true,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			secret, err := getCredentialsSecret(
				context.Background(),
				fake.NewClientBuilder().WithObjects(testCase.secrets...).Build(),
				testNamespace,
				labels.Everything(),
				testCase.repoType,
				testCase.repoURL,
				testCase.prefixMatch,
			)
			require.NoError(t, err)
			if testCase.shouldBeFound {
				require.NotNil(t, secret)
			} else {
				require.Nil(t, secret)
			}
		})
	}
}

func TestSecretToCreds(t *testing.T) {
	secret := &corev1.Secret{
		Data: map[string][]byte{
			"username":      []byte("fake-username"),
			"password":      []byte("fake-password"),
			"sshPrivateKey": []byte("fake-ssh-private-key"),
		},
	}
	creds := secretToCreds(secret)
	require.Equal(t, string(secret.Data["username"]), creds.Username)
	require.Equal(t, string(secret.Data["password"]), creds.Password)
	require.Equal(t, string(secret.Data["sshPrivateKey"]), creds.SSHPrivateKey)
}
