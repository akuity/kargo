package credentials

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
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
