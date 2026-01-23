package kubernetes

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/credentials"
	"github.com/akuity/kargo/pkg/credentials/basic"
)

func TestNewKubernetesDatabase(t *testing.T) {
	testControlPlaneClient := fake.NewClientBuilder().Build()
	testLocalClusterClient := fake.NewClientBuilder().Build()
	testCredentialProviderRegistry := credentials.MustNewProviderRegistry()
	testCfg := DatabaseConfig{
		SharedResourcesNamespace: "fake-namespace",
	}
	d := NewDatabase(
		testControlPlaneClient,
		testLocalClusterClient,
		testCredentialProviderRegistry,
		testCfg,
	)
	require.NotNil(t, d)
	k, ok := d.(*database)
	require.True(t, ok)
	require.Same(t, testControlPlaneClient, k.controlPlaneClient)
	require.Same(t, testLocalClusterClient, k.localClusterClient)
	require.Same(t, testCredentialProviderRegistry, k.credentialProvidersRegistry)
	require.Equal(t, testCfg, k.cfg)
}

// TestGet simply validates that, given a set of valid/matching secrets in
// various namespaces, the correct secret is returned (order of precedence)
func TestGet(t *testing.T) {
	const (
		testProjectNamespace = "fake-namespace"
		testSharedNamespace  = "shared-namespace"

		// This deliberately omits the trailing .git to test normalization
		testGitRepoURL     = "https://github.com/akuity/kargo"
		testInsecureGitURL = "http://github.com/akuity/bogus.git"

		// This is deliberately an image URL that could be mistaken for an SCP-style
		// Git URL to verify that Git URL normalization is not applied to image
		// URLs.
		testImageURL = "my-registry.io:5000/image"

		ociRepoURL = "oci://foo/bar/something"
	)

	testGitLabels := map[string]string{
		kargoapi.LabelKeyCredentialType: credentials.TypeGit.String(),
	}

	testImageLabels := map[string]string{
		kargoapi.LabelKeyCredentialType: credentials.TypeImage.String(),
	}

	testChartLabels := map[string]string{
		kargoapi.LabelKeyCredentialType: credentials.TypeHelm.String(),
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

	sharedGitCredentialWithRepoURL := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "shared-credential-git-repo-url",
			Namespace: testSharedNamespace,
			Labels:    testGitLabels,
		},
		Data: map[string][]byte{
			credentials.FieldRepoURL:  []byte(testGitRepoURL),
			credentials.FieldUsername: []byte("shared-exact"),
			credentials.FieldPassword: []byte("fake-password"),
		},
	}

	sharedGitCredentialWithRepoURLPattern := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "shared-credential-git-repo-url-pattern",
			Namespace: testSharedNamespace,
			Labels:    testGitLabels,
		},
		Data: map[string][]byte{
			credentials.FieldRepoURL:        []byte(testGitRepoURL),
			credentials.FieldRepoURLIsRegex: []byte("true"),
			credentials.FieldUsername:       []byte("shared-pattern"),
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

	sharedImageCredentialWithRepoURL := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "shared-credential-image-repo-url",
			Namespace: testSharedNamespace,
			Labels:    testImageLabels,
		},
		Data: map[string][]byte{
			credentials.FieldRepoURL:  []byte(testImageURL),
			credentials.FieldUsername: []byte("shared-exact"),
			credentials.FieldPassword: []byte("fake-password"),
		},
	}

	sharedImageCredentialWithRepoURLPattern := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "shared-credential-image-repo-url-pattern",
			Namespace: testSharedNamespace,
			Labels:    testImageLabels,
		},
		Data: map[string][]byte{
			credentials.FieldRepoURL:        []byte(testImageURL),
			credentials.FieldRepoURLIsRegex: []byte("true"),
			credentials.FieldUsername:       []byte("shared-pattern"),
			credentials.FieldPassword:       []byte("fake-password"),
		},
	}

	projectChartCredentialWithRepoURLPattern := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "project-credential-chart-repo-url-pattern",
			Namespace: testProjectNamespace,
			Labels:    testChartLabels,
		},
		Data: map[string][]byte{
			credentials.FieldRepoURL:        []byte("^oci://foo/bar"),
			credentials.FieldRepoURLIsRegex: []byte("true"),
			credentials.FieldUsername:       []byte("username"),
			credentials.FieldPassword:       []byte("password"),
		},
	}

	testCases := []struct {
		name         string
		namespace    string
		interceptors *interceptor.Funcs
		secrets      []client.Object
		cfg          DatabaseConfig
		credType     credentials.Type
		repoURL      string
		assertions   func(t *testing.T, creds *credentials.Credentials, err error)
	}{
		{
			name:      "git URL exact match in project namespace",
			namespace: testProjectNamespace,
			secrets:   []client.Object{projectGitCredentialWithRepoURL},
			credType:  credentials.TypeGit,
			repoURL:   testGitRepoURL,
			assertions: func(t *testing.T, creds *credentials.Credentials, err error) {
				require.NoError(t, err)
				require.NotNil(t, creds)
				requireSecretMatchesCreds(t, projectGitCredentialWithRepoURL, creds)
			},
		},
		{
			name:      "git URL pattern match in project namespace",
			namespace: testProjectNamespace,
			secrets:   []client.Object{projectGitCredentialWithRepoURLPattern},
			credType:  credentials.TypeGit,
			repoURL:   testGitRepoURL,
			assertions: func(t *testing.T, creds *credentials.Credentials, err error) {
				require.NoError(t, err)
				require.NotNil(t, creds)
				requireSecretMatchesCreds(t, projectGitCredentialWithRepoURLPattern, creds)
			},
		},
		{
			name:      "git URL exact match in shared namespace",
			namespace: testSharedNamespace,
			secrets:   []client.Object{sharedGitCredentialWithRepoURL},
			cfg: DatabaseConfig{
				SharedResourcesNamespace: testSharedNamespace,
			},
			credType: credentials.TypeGit,
			repoURL:  testGitRepoURL,
			assertions: func(t *testing.T, creds *credentials.Credentials, err error) {
				require.NoError(t, err)
				require.NotNil(t, creds)
				requireSecretMatchesCreds(t, sharedGitCredentialWithRepoURL, creds)
			},
		},
		{
			name:    "git URL pattern match in shared namespace",
			secrets: []client.Object{sharedGitCredentialWithRepoURLPattern},
			cfg: DatabaseConfig{
				SharedResourcesNamespace: testSharedNamespace,
			},
			credType: credentials.TypeGit,
			repoURL:  testGitRepoURL,
			assertions: func(t *testing.T, creds *credentials.Credentials, err error) {
				require.NoError(t, err)
				require.NotNil(t, creds)
				requireSecretMatchesCreds(t, sharedGitCredentialWithRepoURLPattern, creds)
			},
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
			assertions: func(t *testing.T, creds *credentials.Credentials, err error) {
				require.NoError(t, err)
				require.NotNil(t, creds)
				requireSecretMatchesCreds(t, projectImageCredentialWithRepoURL, creds)
			},
		},
		{
			name:     "image URL pattern match in project namespace",
			secrets:  []client.Object{projectImageCredentialWithRepoURLPattern},
			credType: credentials.TypeImage,
			repoURL:  testImageURL,
			assertions: func(t *testing.T, creds *credentials.Credentials, err error) {
				require.NoError(t, err)
				require.NotNil(t, creds)
				requireSecretMatchesCreds(t, projectImageCredentialWithRepoURLPattern, creds)
			},
		},
		{
			name:    "image URL exact match in shared namespace",
			secrets: []client.Object{sharedImageCredentialWithRepoURL},
			cfg: DatabaseConfig{
				SharedResourcesNamespace: testSharedNamespace,
			},
			credType: credentials.TypeImage,
			repoURL:  testImageURL,
			assertions: func(t *testing.T, creds *credentials.Credentials, err error) {
				require.NoError(t, err)
				require.NotNil(t, creds)
				requireSecretMatchesCreds(t, sharedImageCredentialWithRepoURL, creds)
			},
		},
		{
			name:    "image URL pattern match in shared namespace",
			secrets: []client.Object{sharedImageCredentialWithRepoURLPattern},
			cfg: DatabaseConfig{
				SharedResourcesNamespace: testSharedNamespace,
			},
			credType: credentials.TypeImage,
			repoURL:  testImageURL,
			assertions: func(t *testing.T, creds *credentials.Credentials, err error) {
				require.NoError(t, err)
				require.NotNil(t, creds)
				requireSecretMatchesCreds(t, sharedImageCredentialWithRepoURLPattern, creds)
			},
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
			assertions: func(t *testing.T, creds *credentials.Credentials, err error) {
				require.NoError(t, err)
				require.NotNil(t, creds)
				requireSecretMatchesCreds(t, projectGitCredentialWithRepoURL, creds)
			},
		},
		{
			name: "precedence: exact match in shared namespace over pattern match",
			secrets: []client.Object{
				sharedGitCredentialWithRepoURL,
				sharedGitCredentialWithRepoURLPattern,
			},
			cfg: DatabaseConfig{
				SharedResourcesNamespace: testSharedNamespace,
			},
			credType: credentials.TypeGit,
			repoURL:  testGitRepoURL,
			assertions: func(t *testing.T, creds *credentials.Credentials, err error) {
				require.NoError(t, err)
				require.NotNil(t, creds)
				requireSecretMatchesCreds(t, sharedGitCredentialWithRepoURL, creds)
			},
		},
		{
			name: "precedence: match in project namespace over match in shared namespace",
			secrets: []client.Object{
				projectGitCredentialWithRepoURL,
				sharedGitCredentialWithRepoURL,
			},
			credType: credentials.TypeGit,
			repoURL:  testGitRepoURL,
			assertions: func(t *testing.T, creds *credentials.Credentials, err error) {
				require.NoError(t, err)
				require.NotNil(t, creds)
				requireSecretMatchesCreds(t, projectGitCredentialWithRepoURL, creds)
			},
		},
		{
			name: "no match",
			secrets: []client.Object{
				projectGitCredentialWithRepoURL,
				projectGitCredentialWithRepoURLPattern,
				sharedGitCredentialWithRepoURL,
				sharedGitCredentialWithRepoURLPattern,
			},
			credType: credentials.TypeGit,
			repoURL:  "http://github.com/no/secrets/should/match/this.git",
			assertions: func(t *testing.T, creds *credentials.Credentials, err error) {
				require.NoError(t, err)
				require.Nil(t, creds)
			},
		},
		{
			name: "insecure HTTP endpoint",
			// Would match if not for the insecure URL check
			secrets:  []client.Object{projectGitCredentialWithInsecureRepoURL},
			credType: credentials.TypeGit,
			repoURL:  testInsecureGitURL,
			assertions: func(t *testing.T, creds *credentials.Credentials, err error) {
				require.NoError(t, err)
				require.Nil(t, creds)
			},
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
			assertions: func(t *testing.T, creds *credentials.Credentials, err error) {
				require.NoError(t, err)
				require.NotNil(t, creds)
				requireSecretMatchesCreds(t, projectGitCredentialWithInsecureRepoURL, creds)
			},
		},
		{
			name:     "regex normalization",
			secrets:  []client.Object{projectChartCredentialWithRepoURLPattern},
			credType: credentials.TypeHelm,
			repoURL:  ociRepoURL,
			assertions: func(t *testing.T, creds *credentials.Credentials, err error) {
				require.NoError(t, err)
				require.NotNil(t, creds)
				requireSecretMatchesCreds(t, projectChartCredentialWithRepoURLPattern, creds)
			},
		},
		{
			name:      "error getting namespace secret",
			namespace: testProjectNamespace,
			interceptors: &interceptor.Funcs{
				List: func(
					_ context.Context,
					_ client.WithWatch,
					_ client.ObjectList,
					_ ...client.ListOption,
				) error {
					return errors.New("something went wrong")
				},
			},
			secrets:  nil,
			credType: credentials.TypeGit,
			repoURL:  testGitRepoURL,
			assertions: func(t *testing.T, _ *credentials.Credentials, err error) {
				require.Error(t, err)
				require.ErrorContains(t, err,
					"failed to get git creds for https://github.com/akuity/kargo in namespace \"fake-namespace\"",
				)
			},
		},
		{
			name:      "error getting shared namespace secret",
			namespace: testProjectNamespace,
			cfg: DatabaseConfig{
				SharedResourcesNamespace: testSharedNamespace,
			},
			interceptors: &interceptor.Funcs{
				List: func(
					_ context.Context,
					_ client.WithWatch,
					_ client.ObjectList,
					opts ...client.ListOption,
				) error {
					o := opts[0]
					var list client.ListOptions
					o.ApplyToList(&list)
					if list.Namespace == testSharedNamespace {
						return errors.New("something went wrong")
					}
					return nil
				},
			},
			secrets:  nil,
			credType: credentials.TypeGit,
			repoURL:  testGitRepoURL,
			assertions: func(t *testing.T, _ *credentials.Credentials, err error) {
				require.Error(t, err)
				require.ErrorContains(t, err,
					"failed to get git creds for https://github.com/akuity/kargo in shared namespace \"shared-namespace\"",
				)
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			provider := &basic.CredentialProvider{}
			c := fake.NewClientBuilder().WithObjects(testCase.secrets...)
			if testCase.interceptors != nil {
				c.WithInterceptorFuncs(*testCase.interceptors)
			}
			creds, err := NewDatabase(
				c.Build(),
				nil,
				credentials.MustNewProviderRegistry(credentials.ProviderRegistration{
					Predicate: provider.Supports,
					Value:     provider,
				}),
				testCase.cfg,
			).Get(
				t.Context(),
				testCase.namespace,
				testCase.credType,
				testCase.repoURL,
			)
			testCase.assertions(t, creds, err)
		})
	}
}

func requireSecretMatchesCreds(t *testing.T, s *corev1.Secret, creds *credentials.Credentials) {
	t.Helper()
	require.NotNil(t, s)
	require.NotNil(t, creds)
	require.Equal(t, string(s.Data[credentials.FieldUsername]), creds.Username)
	require.Equal(t, string(s.Data[credentials.FieldPassword]), creds.Password)
}
