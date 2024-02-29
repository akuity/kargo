package credentials

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestNewKubernetesDatabase(t *testing.T) {
	testClient := fake.NewClientBuilder().Build()
	testCfg := KubernetesDatabaseConfig{
		GlobalCredentialsNamespaces: []string{"fake-namespace"},
	}
	d := NewKubernetesDatabase(testClient, testCfg)
	require.NotNil(t, d)
	k, ok := d.(*kubernetesDatabase)
	require.True(t, ok)
	require.Same(t, testClient, k.kargoClient)
	require.Equal(t, testCfg, k.cfg)
}

// TestGet simply validates that, given a set of valid/matching secrets in
// various namespaces, the correct secret is returned (order of precedence)
func TestGet(t *testing.T) {
	const (
		testProjectNamespace = "fake-namespace"
		testGlobalNamespace  = "another-fake-namespace"

		testCredType = TypeGit

		// This deliberately omits the trailing .git to test normalization
		testRepoURL     = "https://github.com/akuity/kargo"
		insecureTestURL = "http://github.com/akuity/bogus.git"

		testRepoURLPattern = "https://github\\.com/akuity/.*"
	)

	testLabels := map[string]string{
		credentialTypeLabelKey: string(testCredType),
	}

	projectCredentialWithRepoURL := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "project-credential-repo-url",
			Namespace: testProjectNamespace,
			Labels:    testLabels,
		},
		Data: map[string][]byte{
			"repoURL":  []byte(testRepoURL),
			"username": []byte("project-exact"),
			"password": []byte("fake-password"),
		},
	}

	projectCredentialWithRepoURLPattern := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "project-credential-repo-url-pattern",
			Namespace: testProjectNamespace,
			Labels:    testLabels,
		},
		Data: map[string][]byte{
			"repoURLPattern": []byte(testRepoURLPattern),
			"username":       []byte("project-pattern"),
			"password":       []byte("fake-password"),
		},
	}

	// It would normally not make sense to store a credential like this because
	// Kargo will refuse to look for credentials for insecure URLs. However,
	// this is a secret that WOULD be matched if not for that check. This helps
	// us test that the check is working.
	projectCredentialWithInsecureRepoURL := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "project-credential-insecure-repo-url",
			Namespace: testProjectNamespace,
			Labels:    testLabels,
		},
		Data: map[string][]byte{
			"repoURL":  []byte(insecureTestURL),
			"username": []byte("project-insecure"),
			"password": []byte("fake-password"),
		},
	}

	globalCredentialWithRepoURL := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "global-credential-repo-url",
			Namespace: testGlobalNamespace,
			Labels:    testLabels,
		},
		Data: map[string][]byte{
			"repoURL":  []byte(testRepoURL),
			"username": []byte("global-exact"),
			"password": []byte("fake-password"),
		},
	}

	globalCredentialWithRepoURLPattern := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "global-credential-repo-url-pattern",
			Namespace: testGlobalNamespace,
			Labels:    testLabels,
		},
		Data: map[string][]byte{
			"repoURLPattern": []byte(testRepoURLPattern),
			"username":       []byte("global-pattern"),
			"password":       []byte("fake-password"),
		},
	}

	testCases := []struct {
		name     string
		secrets  []client.Object
		repoURL  string
		expected *corev1.Secret
	}{
		{
			name:     "exact match in project namespace",
			secrets:  []client.Object{projectCredentialWithRepoURL},
			repoURL:  testRepoURL,
			expected: projectCredentialWithRepoURL,
		},
		{
			name:     "pattern match in project namespace",
			secrets:  []client.Object{projectCredentialWithRepoURLPattern},
			repoURL:  testRepoURL,
			expected: projectCredentialWithRepoURLPattern,
		},
		{
			name:     "exact match in global namespace",
			secrets:  []client.Object{globalCredentialWithRepoURL},
			repoURL:  testRepoURL,
			expected: globalCredentialWithRepoURL,
		},
		{
			name:     "pattern match in global namespace",
			secrets:  []client.Object{globalCredentialWithRepoURLPattern},
			repoURL:  testRepoURL,
			expected: globalCredentialWithRepoURLPattern,
		},
		{
			name: "precedence: exact match in project namespace over pattern match",
			secrets: []client.Object{
				projectCredentialWithRepoURL,
				projectCredentialWithRepoURLPattern,
			},
			repoURL:  testRepoURL,
			expected: projectCredentialWithRepoURL,
		},
		{
			name: "precedence: exact match in global namespace over pattern match",
			secrets: []client.Object{
				globalCredentialWithRepoURL,
				globalCredentialWithRepoURLPattern,
			},
			repoURL:  testRepoURL,
			expected: globalCredentialWithRepoURL,
		},
		{
			name: "precedence: match in project namespace over match in global namespace",
			secrets: []client.Object{
				projectCredentialWithRepoURL,
				globalCredentialWithRepoURL,
			},
			repoURL:  testRepoURL,
			expected: projectCredentialWithRepoURL,
		},
		{
			name: "no match",
			secrets: []client.Object{
				projectCredentialWithRepoURL,
				projectCredentialWithRepoURLPattern,
				globalCredentialWithRepoURL,
				globalCredentialWithRepoURLPattern,
			},
			repoURL:  "http://github.com/no/secrets/should/match/this.git",
			expected: nil,
		},
		{
			name: "insecure HTTP endpoint",
			// Would match if not for the insecure URL check
			secrets:  []client.Object{projectCredentialWithInsecureRepoURL},
			repoURL:  insecureTestURL,
			expected: nil,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			creds, found, err := NewKubernetesDatabase(
				fake.NewClientBuilder().WithObjects(testCase.secrets...).Build(),
				KubernetesDatabaseConfig{
					GlobalCredentialsNamespaces: []string{testGlobalNamespace},
				},
			).Get(
				context.Background(),
				testProjectNamespace,
				testCredType,
				testCase.repoURL,
			)
			require.NoError(t, err)

			if testCase.expected == nil {
				require.False(t, found)
				require.Empty(t, creds)
				return
			}

			require.True(t, found)
			require.Equal(
				t,
				string(testCase.expected.Data["username"]),
				creds.Username,
			)
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
