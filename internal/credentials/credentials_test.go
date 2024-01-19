package credentials

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestNewKubernetesDatabase(t *testing.T) {
	const testArgoCDNameSpace = "argocd"
	testClient := fake.NewClientBuilder().Build()
	d := NewKubernetesDatabase(
		testClient,
		WithArgoCDNamespace(testArgoCDNameSpace),
		WithArgoClient(testClient),
	)
	require.NotNil(t, d)
	k, ok := d.(*kubernetesDatabase)
	require.True(t, ok)
	require.Equal(t, testArgoCDNameSpace, k.ArgoCDNamespace)
	require.Same(t, testClient, k.kargoClient)
	require.Same(t, testClient, k.argoClient)
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
		ObjectMeta: v1.ObjectMeta{
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
		ObjectMeta: v1.ObjectMeta{
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
		ObjectMeta: v1.ObjectMeta{
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
		ObjectMeta: v1.ObjectMeta{
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
		ObjectMeta: v1.ObjectMeta{
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
		ObjectMeta: v1.ObjectMeta{
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
		ObjectMeta: v1.ObjectMeta{
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
		ObjectMeta: v1.ObjectMeta{
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
				WithArgoCDNamespace(testArgoCDNameSpace),
				WithArgoClient(testClient),
				WithGlobalCredentialsNamespaces(testGlobalNamespaces),
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
	const testNamespace = "fake-namespace"
	const testURLPrefix = "https://github.com/example"
	const testURL = testURLPrefix + "/example.git"
	const bogusTestURL = "https://github.com/bogus/bogus.git"
	testClient := fake.NewClientBuilder().WithObjects(
		&corev1.Secret{ // Should never match because it has no data
			ObjectMeta: v1.ObjectMeta{
				Name:      "creds-0",
				Namespace: testNamespace,
			},
		},
		&corev1.Secret{ // Should never match because its the wrong type of repo
			ObjectMeta: v1.ObjectMeta{
				Name:      "creds-1",
				Namespace: testNamespace,
			},
			Data: map[string][]byte{
				"type": []byte(TypeImage),
				"url":  []byte(testURL),
			},
		},
		&corev1.Secret{ // Should never match because its missing the url field
			ObjectMeta: v1.ObjectMeta{
				Name:      "creds-2",
				Namespace: testNamespace,
			},
			Data: map[string][]byte{
				"type": []byte(TypeGit),
			},
		},
		&corev1.Secret{ // Should be an exact match
			ObjectMeta: v1.ObjectMeta{
				Name:      "creds-3",
				Namespace: testNamespace,
			},
			Data: map[string][]byte{
				"type": []byte(TypeGit),
				"url":  []byte(testURL),
			},
		},
		&corev1.Secret{ // Should be a prefix match
			ObjectMeta: v1.ObjectMeta{
				Name:      "creds-4",
				Namespace: testNamespace,
			},
			Data: map[string][]byte{
				"type": []byte(TypeGit),
				"url":  []byte(testURLPrefix),
			},
		},
	).Build()
	testCases := []struct {
		name              string
		repoURL           string
		acceptPrefixMatch bool
		assertions        func(*corev1.Secret, error)
	}{
		{
			name:              "exact match not found",
			repoURL:           bogusTestURL,
			acceptPrefixMatch: false,
			assertions: func(secret *corev1.Secret, err error) {
				require.NoError(t, err)
				require.Nil(t, secret)
			},
		},
		{
			name:              "exact match found",
			repoURL:           testURL,
			acceptPrefixMatch: false,
			assertions: func(secret *corev1.Secret, err error) {
				require.NoError(t, err)
				require.NotNil(t, secret)
			},
		},
		{
			name:              "prefix match not found",
			repoURL:           bogusTestURL,
			acceptPrefixMatch: true,
			assertions: func(secret *corev1.Secret, err error) {
				require.NoError(t, err)
				require.Nil(t, secret)
			},
		},
		{
			name:              "prefix match found",
			repoURL:           testURL,
			acceptPrefixMatch: true,
			assertions: func(secret *corev1.Secret, err error) {
				require.NoError(t, err)
				require.NotNil(t, secret)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				getCredentialsSecret(
					context.Background(),
					testClient,
					testNamespace,
					labels.Everything(),
					TypeGit,
					testCase.repoURL,
					testCase.acceptPrefixMatch,
				),
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
