package promotion

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
	"github.com/akuity/kargo/pkg/api"
	"github.com/akuity/kargo/pkg/controller/git"
)

func TestGitUserResolver_Resolve(t *testing.T) {
	const testNamespace = "kargo-system"

	fallbackUser := git.User{
		Name:           "Fallback",
		Email:          "fallback@example.com",
		SigningKeyType: git.SigningKeyTypeGPG,
		SigningKeyPath: "/etc/kargo/git/signingKey",
	}

	testCases := []struct {
		name    string
		objects func() []client.Object
		assert  func(*testing.T, git.User, error)
	}{
		{
			name:    "no ClusterConfig",
			objects: func() []client.Object { return nil },
			assert: func(t *testing.T, user git.User, err error) {
				require.NoError(t, err)
				require.Equal(t, fallbackUser, user)
			},
		},
		{
			name: "ClusterConfig without gitClient",
			objects: func() []client.Object {
				return []client.Object{
					&kargoapi.ClusterConfig{
						ObjectMeta: metav1.ObjectMeta{Name: api.ClusterConfigName},
					},
				}
			},
			assert: func(t *testing.T, user git.User, err error) {
				require.NoError(t, err)
				require.Equal(t, fallbackUser, user)
			},
		},
		{
			name: "ClusterConfig with gitClient but no signingKeySecret",
			objects: func() []client.Object {
				return []client.Object{
					&kargoapi.ClusterConfig{
						ObjectMeta: metav1.ObjectMeta{Name: api.ClusterConfigName},
						Spec: kargoapi.ClusterConfigSpec{
							GitClient: &kargoapi.GitClientConfig{
								Name:  "Kargo",
								Email: "kargo@example.com",
							},
						},
					},
				}
			},
			assert: func(t *testing.T, user git.User, err error) {
				require.NoError(t, err)
				require.Equal(t, "Kargo", user.Name)
				require.Equal(t, "kargo@example.com", user.Email)
				// Signing config falls back
				require.Equal(t, git.SigningKeyTypeGPG, user.SigningKeyType)
				require.Equal(t, "/etc/kargo/git/signingKey", user.SigningKeyPath)
				require.Empty(t, user.SigningKey)
			},
		},
		{
			name: "ClusterConfig with signingKeySecret",
			objects: func() []client.Object {
				return []client.Object{
					&kargoapi.ClusterConfig{
						ObjectMeta: metav1.ObjectMeta{Name: api.ClusterConfigName},
						Spec: kargoapi.ClusterConfigSpec{
							GitClient: &kargoapi.GitClientConfig{
								Name:  "Kargo",
								Email: "kargo@example.com",
								SigningKeySecret: &corev1.LocalObjectReference{
									Name: "my-gpg-key",
								},
							},
						},
					},
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: testNamespace,
							Name:      "my-gpg-key",
						},
						Data: map[string][]byte{
							signingKeyDataKey: []byte("fake-gpg-key-material"),
						},
					},
				}
			},
			assert: func(t *testing.T, user git.User, err error) {
				require.NoError(t, err)
				require.Equal(t, "Kargo", user.Name)
				require.Equal(t, "kargo@example.com", user.Email)
				require.Equal(t, git.SigningKeyTypeGPG, user.SigningKeyType)
				require.Equal(t, "fake-gpg-key-material", user.SigningKey)
				require.Empty(t, user.SigningKeyPath)
			},
		},
		{
			name: "Secret not found",
			objects: func() []client.Object {
				return []client.Object{
					&kargoapi.ClusterConfig{
						ObjectMeta: metav1.ObjectMeta{Name: api.ClusterConfigName},
						Spec: kargoapi.ClusterConfigSpec{
							GitClient: &kargoapi.GitClientConfig{
								Name:  "Kargo",
								Email: "kargo@example.com",
								SigningKeySecret: &corev1.LocalObjectReference{
									Name: "nonexistent",
								},
							},
						},
					},
				}
			},
			assert: func(t *testing.T, _ git.User, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "error getting signing key Secret")
			},
		},
		{
			name: "Secret missing expected data key",
			objects: func() []client.Object {
				return []client.Object{
					&kargoapi.ClusterConfig{
						ObjectMeta: metav1.ObjectMeta{Name: api.ClusterConfigName},
						Spec: kargoapi.ClusterConfigSpec{
							GitClient: &kargoapi.GitClientConfig{
								Name:  "Kargo",
								Email: "kargo@example.com",
								SigningKeySecret: &corev1.LocalObjectReference{
									Name: "bad-secret",
								},
							},
						},
					},
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: testNamespace,
							Name:      "bad-secret",
						},
						Data: map[string][]byte{
							"wrongKey": []byte("data"),
						},
					},
				}
			},
			assert: func(t *testing.T, _ git.User, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "does not contain expected key")
			},
		},
	}

	scheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(scheme))
	require.NoError(t, corev1.AddToScheme(scheme))

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			objs := tc.objects()
			c := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(objs...).
				Build()
			resolver := NewGitUserResolver(c, testNamespace, fallbackUser)
			user, err := resolver.Resolve(context.Background())
			tc.assert(t, user, err)
		})
	}
}
