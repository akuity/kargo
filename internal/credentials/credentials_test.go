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
	require.Same(t, testClient, k.argocdClient)
	require.Equal(t, testCfg, k.cfg)
}

// TestGet simply validates that, given a set of valid/matching secrets in
// various namespaces, the correct secret is returned (order of precedence)
func TestGet(t *testing.T) {
	const testArgoCDNameSpace = "argocd"
	const testNamespace = "fake-namespace"
	var testGlobalNamespaces = []string{"kargo"}
	const testURLPrefix = "myrepo.com"
	const testURL = testURLPrefix + "/myrepo/myimage"
	secretInNamespaceExact := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "in-namespace-exact",
			Namespace: testNamespace,
			Labels: map[string]string{
				kargoSecretTypeLabelKey: repositorySecretTypeLabelValue,
			},
		},
		Data: map[string][]byte{
			"type":     []byte(TypeImage),
			"username": []byte("in-namespace-exact"),
			"password": []byte("fake-password"),
			"url":      []byte(testURL),
		},
	}
	secretInNamespacePrefix := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "in-namespace-prefix",
			Namespace: testNamespace,
			Labels: map[string]string{
				kargoSecretTypeLabelKey: repoCredsSecretTypeLabelValue,
			},
		},
		Data: map[string][]byte{
			"type":     []byte(TypeImage),
			"username": []byte("in-namespace-prefix"),
			"password": []byte("fake-password"),
			"url":      []byte(testURLPrefix),
		},
	}
	secretInGlobalExact := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "in-global-exact",
			Namespace: testGlobalNamespaces[0],
			Labels: map[string]string{
				kargoSecretTypeLabelKey: repositorySecretTypeLabelValue,
			},
		},
		Data: map[string][]byte{
			"type":     []byte(TypeImage),
			"username": []byte("in-global-exact"),
			"password": []byte("fake-password"),
			"url":      []byte(testURL),
		},
	}
	secretInGlobalPrefix := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "in-global-prefix",
			Namespace: testGlobalNamespaces[0],
			Labels: map[string]string{
				kargoSecretTypeLabelKey: repoCredsSecretTypeLabelValue,
			},
		},
		Data: map[string][]byte{
			"type":     []byte(TypeImage),
			"username": []byte("in-global-prefix"),
			"password": []byte("fake-password"),
			"url":      []byte(testURLPrefix),
		},
	}
	secretInArgoCDNamespaceExact := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "in-argocd-exact",
			Namespace: testArgoCDNameSpace,
			Labels: map[string]string{
				argoCDSecretTypeLabelKey: repositorySecretTypeLabelValue,
			},
			Annotations: map[string]string{
				authorizedProjectsAnnotationKey: testNamespace,
			},
		},
		Data: map[string][]byte{
			"type":     []byte(TypeImage),
			"username": []byte("in-argocd-exact"),
			"password": []byte("fake-password"),
			"url":      []byte(testURL),
		},
	}
	secretInArgoCDNamespacePrefix := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "in-argocd-prefix",
			Namespace: testArgoCDNameSpace,
			Labels: map[string]string{
				argoCDSecretTypeLabelKey: repoCredsSecretTypeLabelValue,
			},
			Annotations: map[string]string{
				authorizedProjectsAnnotationKey: testNamespace,
			},
		},
		Data: map[string][]byte{
			"type":     []byte(TypeImage),
			"username": []byte("in-argocd-prefix"),
			"password": []byte("fake-password"),
			"url":      []byte(testURLPrefix),
		},
	}
	secretInArgoCDNamespacePrefixMissingAuthorization := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "in-argocd-prefix",
			Namespace: testArgoCDNameSpace,
			Labels: map[string]string{
				argoCDSecretTypeLabelKey: repoCredsSecretTypeLabelValue,
			},
		},
		Data: map[string][]byte{
			"type":     []byte(TypeImage),
			"username": []byte("in-argocd-prefix"),
			"password": []byte("fake-password"),
			"url":      []byte(testURLPrefix),
		},
	}
	secretInArgoCDNamespacePrefixWrongAuthorization := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "in-argocd-prefix",
			Namespace: testArgoCDNameSpace,
			Labels: map[string]string{
				argoCDSecretTypeLabelKey: repoCredsSecretTypeLabelValue,
			},
			Annotations: map[string]string{
				authorizedProjectsAnnotationKey: "someotherproject",
			},
		},
		Data: map[string][]byte{
			"type":     []byte(TypeImage),
			"username": []byte("in-argocd-prefix"),
			"password": []byte("fake-password"),
			"url":      []byte(testURLPrefix),
		},
	}

	testCases := []struct {
		name     string
		secrets  []client.Object
		expected *corev1.Secret
		found    bool
	}{
		{
			name:     "single secret in namespace exact",
			secrets:  []client.Object{secretInNamespaceExact},
			expected: secretInNamespaceExact,
			found:    true,
		},
		{
			name:     "single secret in namespace prefix",
			secrets:  []client.Object{secretInNamespacePrefix},
			expected: secretInNamespacePrefix,
			found:    true,
		},
		{
			name:     "single secret in global namespace exact",
			secrets:  []client.Object{secretInGlobalExact},
			expected: secretInGlobalExact,
			found:    true,
		},
		{
			name:     "single secret in global namespace prefix",
			secrets:  []client.Object{secretInGlobalPrefix},
			expected: secretInGlobalPrefix,
			found:    true,
		},
		{
			name:     "single secret in argocd namespace exact",
			secrets:  []client.Object{secretInArgoCDNamespaceExact},
			expected: secretInArgoCDNamespaceExact,
			found:    true,
		},
		{
			name:     "single secret in argocd namespace prefix",
			secrets:  []client.Object{secretInArgoCDNamespacePrefix},
			expected: secretInArgoCDNamespacePrefix,
			found:    true,
		},
		{
			name:     "in namespace exact before prefix",
			secrets:  []client.Object{secretInNamespaceExact, secretInNamespacePrefix},
			expected: secretInNamespaceExact,
			found:    true,
		},
		{
			name:     "global exact before prefix",
			secrets:  []client.Object{secretInGlobalExact, secretInGlobalPrefix},
			expected: secretInGlobalExact,
			found:    true,
		},
		{
			name:     "argocd exact before prefix",
			secrets:  []client.Object{secretInArgoCDNamespaceExact, secretInArgoCDNamespacePrefix},
			expected: secretInArgoCDNamespaceExact,
			found:    true,
		},
		{
			name:     "namespace before global",
			secrets:  []client.Object{secretInNamespacePrefix, secretInGlobalPrefix},
			expected: secretInNamespacePrefix,
			found:    true,
		},
		{
			name:     "global before argocd",
			secrets:  []client.Object{secretInGlobalPrefix, secretInArgoCDNamespacePrefix},
			expected: secretInGlobalPrefix,
			found:    true,
		},
		{
			name:    "argocd credential with missing auth",
			secrets: []client.Object{secretInArgoCDNamespacePrefixMissingAuthorization},
			found:   false,
		},
		{
			name:    "argocd credential with wrong auth",
			secrets: []client.Object{secretInArgoCDNamespacePrefixWrongAuthorization},
			found:   false,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testClient := fake.NewClientBuilder().WithObjects(testCase.secrets...).Build()

			d := NewKubernetesDatabase(
				testClient,
				testClient,
				KubernetesDatabaseConfig{
					ArgoCDNamespace:             testArgoCDNameSpace,
					GlobalCredentialsNamespaces: testGlobalNamespaces,
				},
			)

			creds, ok, err := d.Get(context.Background(), testNamespace, TypeImage, testURL)
			require.NoError(t, err)
			require.Equal(t, testCase.found, ok)
			if testCase.found {
				require.Equal(t, string(testCase.expected.Data["username"]), creds.Username)
			}
		})
	}

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
