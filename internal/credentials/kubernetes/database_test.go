package kubernetes

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/credentials"
)

func TestNewKubernetesDatabase(t *testing.T) {
	testClient := fake.NewClientBuilder().Build()
	testCfg := DatabaseConfig{
		GlobalCredentialsNamespaces: []string{"fake-namespace"},
	}
	d := NewDatabase(context.Background(), testClient, testCfg)
	require.NotNil(t, d)
	k, ok := d.(*database)
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

		// This deliberately omits the trailing .git to test normalization
		testGitRepoURL     = "https://github.com/akuity/kargo"
		testInsecureGitURL = "http://github.com/akuity/bogus.git"

		// This is deliberately an image URL that could be mistaken for an SCP-style
		// Git URL to verify that Git URL normalization is not applied to image
		// URLs.
		testImageURL = "my-registry.io:5000/image"
	)

	testGitLabels := map[string]string{
		kargoapi.LabelKeyCredentialType: credentials.TypeGit.String(),
	}

	testImageLabels := map[string]string{
		kargoapi.LabelKeyCredentialType: credentials.TypeImage.String(),
	}

	projectGitCredentialWithRepoURL := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "project-credential-git-repo-url",
			Namespace: testProjectNamespace,
			Labels:    testGitLabels,
		},
		Data: map[string][]byte{
			credentials.FieldRepoURL:  []byte(testGitRepoURL),
			credentials.FieldUsername: []byte("project-exact"),
			credentials.FieldPassword: []byte("fake-password"),
		},
	}

	projectGitCredentialWithRepoURLPattern := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "project-credential-git-repo-url-pattern",
			Namespace: testProjectNamespace,
			Labels:    testGitLabels,
		},
		Data: map[string][]byte{
			credentials.FieldRepoURL:        []byte(testGitRepoURL),
			credentials.FieldRepoURLIsRegex: []byte("true"),
			credentials.FieldUsername:       []byte("project-pattern"),
			credentials.FieldPassword:       []byte("fake-password"),
		},
	}

	// It would normally not make sense to store a credential like this because
	// Kargo will refuse to look for credentials for insecure URLs. However,
	// this is a secret that WOULD be matched if not for that check. This helps
	// us test that the check is working.
	projectGitCredentialWithInsecureRepoURL := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "project-credential-git-insecure-repo-url",
			Namespace: testProjectNamespace,
			Labels:    testGitLabels,
		},
		Data: map[string][]byte{
			credentials.FieldRepoURL:  []byte(testInsecureGitURL),
			credentials.FieldUsername: []byte("project-insecure"),
			credentials.FieldPassword: []byte("fake-password"),
		},
	}

	globalGitCredentialWithRepoURL := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "global-credential-git-repo-url",
			Namespace: testGlobalNamespace,
			Labels:    testGitLabels,
		},
		Data: map[string][]byte{
			credentials.FieldRepoURL:  []byte(testGitRepoURL),
			credentials.FieldUsername: []byte("global-exact"),
			credentials.FieldPassword: []byte("fake-password"),
		},
	}

	globalGitCredentialWithRepoURLPattern := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "global-credential-git-repo-url-pattern",
			Namespace: testGlobalNamespace,
			Labels:    testGitLabels,
		},
		Data: map[string][]byte{
			credentials.FieldRepoURL:        []byte(testGitRepoURL),
			credentials.FieldRepoURLIsRegex: []byte("true"),
			credentials.FieldUsername:       []byte("global-pattern"),
			credentials.FieldPassword:       []byte("fake-password"),
		},
	}

	projectImageCredentialWithRepoURL := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "project-credential-image-repo-url",
			Namespace: testProjectNamespace,
			Labels:    testImageLabels,
		},
		Data: map[string][]byte{
			credentials.FieldRepoURL:  []byte(testImageURL),
			credentials.FieldUsername: []byte("project-exact"),
			credentials.FieldPassword: []byte("fake-password"),
		},
	}

	projectImageCredentialWithRepoURLPattern := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "project-credential-image-repo-url-pattern",
			Namespace: testProjectNamespace,
			Labels:    testImageLabels,
		},
		Data: map[string][]byte{
			credentials.FieldRepoURL:        []byte(testImageURL),
			credentials.FieldRepoURLIsRegex: []byte("true"),
			credentials.FieldUsername:       []byte("project-pattern"),
			credentials.FieldPassword:       []byte("fake-password"),
		},
	}

	globalImageCredentialWithRepoURL := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "global-credential-image-repo-url",
			Namespace: testGlobalNamespace,
			Labels:    testImageLabels,
		},
		Data: map[string][]byte{
			credentials.FieldRepoURL:  []byte(testImageURL),
			credentials.FieldUsername: []byte("global-exact"),
			credentials.FieldPassword: []byte("fake-password"),
		},
	}

	globalImageCredentialWithRepoURLPattern := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "global-credential-image-repo-url-pattern",
			Namespace: testGlobalNamespace,
			Labels:    testImageLabels,
		},
		Data: map[string][]byte{
			credentials.FieldRepoURL:        []byte(testImageURL),
			credentials.FieldRepoURLIsRegex: []byte("true"),
			credentials.FieldUsername:       []byte("global-pattern"),
			credentials.FieldPassword:       []byte("fake-password"),
		},
	}

	testCases := []struct {
		name     string
		secrets  []client.Object
		cfg      DatabaseConfig
		credType credentials.Type
		repoURL  string
		expected *corev1.Secret
	}{
		{
			name:     "git URL exact match in project namespace",
			secrets:  []client.Object{projectGitCredentialWithRepoURL},
			credType: credentials.TypeGit,
			repoURL:  testGitRepoURL,
			expected: projectGitCredentialWithRepoURL,
		},
		{
			name:     "git URL pattern match in project namespace",
			secrets:  []client.Object{projectGitCredentialWithRepoURLPattern},
			credType: credentials.TypeGit,
			repoURL:  testGitRepoURL,
			expected: projectGitCredentialWithRepoURLPattern,
		},
		{
			name:    "git URL exact match in global namespace",
			secrets: []client.Object{globalGitCredentialWithRepoURL},
			cfg: DatabaseConfig{
				GlobalCredentialsNamespaces: []string{testGlobalNamespace},
			},
			credType: credentials.TypeGit,
			repoURL:  testGitRepoURL,
			expected: globalGitCredentialWithRepoURL,
		},
		{
			name:    "git URL pattern match in global namespace",
			secrets: []client.Object{globalGitCredentialWithRepoURLPattern},
			cfg: DatabaseConfig{
				GlobalCredentialsNamespaces: []string{testGlobalNamespace},
			},
			credType: credentials.TypeGit,
			repoURL:  testGitRepoURL,
			expected: globalGitCredentialWithRepoURLPattern,
		},
		// Image URLs of the form host:port/image can be mistaken for SCP-style Git
		// URLs. The next several test cases verify that Git URL normalization is
		// not being applied to image URLs and incorrectly normalizing
		// host:port/image as ssh://host:port/image.
		{
			name:     "image URL exact match in project namespace",
			secrets:  []client.Object{projectImageCredentialWithRepoURL},
			credType: credentials.TypeImage,
			repoURL:  testImageURL,
			expected: projectImageCredentialWithRepoURL,
		},
		{
			name:     "image URL pattern match in project namespace",
			secrets:  []client.Object{projectImageCredentialWithRepoURLPattern},
			credType: credentials.TypeImage,
			repoURL:  testImageURL,
			expected: projectImageCredentialWithRepoURLPattern,
		},
		{
			name:    "image URL exact match in global namespace",
			secrets: []client.Object{globalImageCredentialWithRepoURL},
			cfg: DatabaseConfig{
				GlobalCredentialsNamespaces: []string{testGlobalNamespace},
			},
			credType: credentials.TypeImage,
			repoURL:  testImageURL,
			expected: globalImageCredentialWithRepoURL,
		},
		{
			name:    "image URL pattern match in global namespace",
			secrets: []client.Object{globalImageCredentialWithRepoURLPattern},
			cfg: DatabaseConfig{
				GlobalCredentialsNamespaces: []string{testGlobalNamespace},
			},
			credType: credentials.TypeImage,
			repoURL:  testImageURL,
			expected: globalImageCredentialWithRepoURLPattern,
		},
		// The next several tests cases confirm the precedence rules for credential
		// matching.
		{
			name: "precedence: exact match in project namespace over pattern match",
			secrets: []client.Object{
				projectGitCredentialWithRepoURL,
				projectGitCredentialWithRepoURLPattern,
			},
			credType: credentials.TypeGit,
			repoURL:  testGitRepoURL,
			expected: projectGitCredentialWithRepoURL,
		},
		{
			name: "precedence: exact match in global namespace over pattern match",
			secrets: []client.Object{
				globalGitCredentialWithRepoURL,
				globalGitCredentialWithRepoURLPattern,
			},
			cfg: DatabaseConfig{
				GlobalCredentialsNamespaces: []string{testGlobalNamespace},
			},
			credType: credentials.TypeGit,
			repoURL:  testGitRepoURL,
			expected: globalGitCredentialWithRepoURL,
		},
		{
			name: "precedence: match in project namespace over match in global namespace",
			secrets: []client.Object{
				projectGitCredentialWithRepoURL,
				globalGitCredentialWithRepoURL,
			},
			credType: credentials.TypeGit,
			repoURL:  testGitRepoURL,
			expected: projectGitCredentialWithRepoURL,
		},
		{
			name: "no match",
			secrets: []client.Object{
				projectGitCredentialWithRepoURL,
				projectGitCredentialWithRepoURLPattern,
				globalGitCredentialWithRepoURL,
				globalGitCredentialWithRepoURLPattern,
			},
			credType: credentials.TypeGit,
			repoURL:  "http://github.com/no/secrets/should/match/this.git",
			expected: nil,
		},
		{
			name: "insecure HTTP endpoint",
			// Would match if not for the insecure URL check
			secrets:  []client.Object{projectGitCredentialWithInsecureRepoURL},
			credType: credentials.TypeGit,
			repoURL:  testInsecureGitURL,
			expected: nil,
		},
		{
			name: "insecure HTTP endpoint allowed",
			// Matches because credentials for insecure URLs are allowed
			secrets: []client.Object{projectGitCredentialWithInsecureRepoURL},
			cfg: DatabaseConfig{
				AllowCredentialsOverHTTP: true,
			},
			credType: credentials.TypeGit,
			repoURL:  testInsecureGitURL,
			expected: projectGitCredentialWithInsecureRepoURL,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			creds, err := NewDatabase(
				context.Background(),
				fake.NewClientBuilder().WithObjects(testCase.secrets...).Build(),
				testCase.cfg,
			).Get(
				context.Background(),
				testProjectNamespace,
				testCase.credType,
				testCase.repoURL,
			)
			require.NoError(t, err)

			if testCase.expected == nil {
				require.Nil(t, creds)
				return
			}

			require.NotNil(t, creds)
			require.Equal(
				t,
				string(testCase.expected.Data["username"]),
				creds.Username,
			)
		})
	}
}
