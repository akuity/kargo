package credentials

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func TestNewKubernetesDatabase(t *testing.T) {
	testClient := fake.NewClientBuilder().Build()
	testCfg := KubernetesDatabaseConfig{
		GlobalCredentialsNamespaces: []string{"fake-namespace"},
	}
	d := NewKubernetesDatabase(context.Background(), testClient, testCfg)
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
	)

	testLabels := map[string]string{
		kargoapi.CredentialTypeLabelKey: testCredType.String(),
	}

	projectCredentialWithRepoURL := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "project-credential-repo-url",
			Namespace: testProjectNamespace,
			Labels:    testLabels,
		},
		Data: map[string][]byte{
			FieldRepoURL:  []byte(testRepoURL),
			FieldUsername: []byte("project-exact"),
			FieldPassword: []byte("fake-password"),
		},
	}

	projectCredentialWithRepoURLPattern := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "project-credential-repo-url-pattern",
			Namespace: testProjectNamespace,
			Labels:    testLabels,
		},
		Data: map[string][]byte{
			FieldRepoURL:        []byte(testRepoURL),
			FieldRepoURLIsRegex: []byte("true"),
			FieldUsername:       []byte("project-pattern"),
			FieldPassword:       []byte("fake-password"),
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
			FieldRepoURL:  []byte(insecureTestURL),
			FieldUsername: []byte("project-insecure"),
			FieldPassword: []byte("fake-password"),
		},
	}

	globalCredentialWithRepoURL := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "global-credential-repo-url",
			Namespace: testGlobalNamespace,
			Labels:    testLabels,
		},
		Data: map[string][]byte{
			FieldRepoURL:  []byte(testRepoURL),
			FieldUsername: []byte("global-exact"),
			FieldPassword: []byte("fake-password"),
		},
	}

	globalCredentialWithRepoURLPattern := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "global-credential-repo-url-pattern",
			Namespace: testGlobalNamespace,
			Labels:    testLabels,
		},
		Data: map[string][]byte{
			FieldRepoURL:        []byte(testRepoURL),
			FieldRepoURLIsRegex: []byte("true"),
			FieldUsername:       []byte("global-pattern"),
			FieldPassword:       []byte("fake-password"),
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
				context.Background(),
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
	const (
		testUsername = "fake-username"
		testPassword = "fake-password"
	)
	testFoundCreds := Credentials{
		Username: testUsername,
		Password: testPassword,
	}
	testCases := []struct {
		name       string
		db         *kubernetesDatabase
		credType   Type
		assertions func(t *testing.T, creds Credentials, err error)
	}{
		{
			name:     "error from github app helper",
			credType: TypeGit,
			db: &kubernetesDatabase{
				ghAppHelperFn: func(*corev1.Secret) (string, string, error) {
					return "", "", errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, creds Credentials, err error) {
				require.ErrorContains(t, err, "something went wrong")
				require.Empty(t, creds)
			},
		},
		{
			name:     "github app helper finds credentials",
			credType: TypeGit,
			db: &kubernetesDatabase{
				ghAppHelperFn: func(*corev1.Secret) (string, string, error) {
					return testUsername, testPassword, nil
				},
			},
			assertions: func(t *testing.T, creds Credentials, err error) {
				require.NoError(t, err)
				require.Equal(t, testFoundCreds, creds)
			},
		},
		{
			name:     "github app helper finds no credentials",
			credType: TypeGit,
			db: &kubernetesDatabase{
				ghAppHelperFn: func(*corev1.Secret) (string, string, error) {
					return "", "", nil
				},
			},
			assertions: func(t *testing.T, creds Credentials, err error) {
				require.NoError(t, err)
				require.Empty(t, creds)
			},
		},
		{
			name:     "error from ecr access key helper",
			credType: TypeImage,
			db: &kubernetesDatabase{
				ecrAKHelperFn: func(context.Context, *corev1.Secret) (string, string, error) {
					return "", "", errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, creds Credentials, err error) {
				require.ErrorContains(t, err, "something went wrong")
				require.Empty(t, creds)
			},
		},
		{
			name:     "ecr access key helper finds credentials",
			credType: TypeImage,
			db: &kubernetesDatabase{
				ecrAKHelperFn: func(context.Context, *corev1.Secret) (string, string, error) {
					return testUsername, testPassword, nil
				},
			},
			assertions: func(t *testing.T, creds Credentials, err error) {
				require.NoError(t, err)
				require.Equal(t, testFoundCreds, creds)
			},
		},
		{
			name:     "error from gcp service account key helper",
			credType: TypeImage,
			db: &kubernetesDatabase{
				ecrAKHelperFn: func(context.Context, *corev1.Secret) (string, string, error) {
					return "", "", nil
				},
				gcpSAKHelperFn: func(context.Context, *corev1.Secret) (string, string, error) {
					return "", "", errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, creds Credentials, err error) {
				require.ErrorContains(t, err, "something went wrong")
				require.Empty(t, creds)
			},
		},
		{
			name:     "gcp service account key helper finds credentials",
			credType: TypeImage,
			db: &kubernetesDatabase{
				ecrAKHelperFn: func(context.Context, *corev1.Secret) (string, string, error) {
					return "", "", nil
				},
				gcpSAKHelperFn: func(context.Context, *corev1.Secret) (string, string, error) {
					return testUsername, testPassword, nil
				},
			},
			assertions: func(t *testing.T, creds Credentials, err error) {
				require.NoError(t, err)
				require.Equal(t, testFoundCreds, creds)
			},
		},
		{
			name:     "no image credential helpers find credentials",
			credType: TypeImage,
			db: &kubernetesDatabase{
				ecrAKHelperFn: func(context.Context, *corev1.Secret) (string, string, error) {
					return "", "", nil
				},
				gcpSAKHelperFn: func(context.Context, *corev1.Secret) (string, string, error) {
					return "", "", nil
				},
			},
			assertions: func(t *testing.T, creds Credentials, err error) {
				require.NoError(t, err)
				require.Empty(t, creds)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			creds, err := testCase.db.secretToCreds(
				context.Background(), testCase.credType, &corev1.Secret{},
			)
			testCase.assertions(t, creds, err)
		})
	}
}
