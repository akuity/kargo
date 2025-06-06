package server

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/server/config"
	"github.com/akuity/kargo/internal/server/kubernetes"
	"github.com/akuity/kargo/internal/server/validation"
)

func TestUpdateProjectSecret(t *testing.T) {
	ctx := context.Background()

	cl, err := kubernetes.NewClient(
		ctx,
		&rest.Config{},
		kubernetes.ClientOptions{
			SkipAuthorization: true,
			NewInternalClient: func(_ context.Context, _ *rest.Config, s *runtime.Scheme) (client.Client, error) {
				return fake.NewClientBuilder().
					WithScheme(s).
					WithObjects(
						mustNewObject[corev1.Namespace]("testdata/namespace.yaml"),
						&corev1.Secret{
							ObjectMeta: metav1.ObjectMeta{
								Namespace: "kargo-demo",
								Name:      "secret",
								Labels: map[string]string{
									kargoapi.CredentialTypeLabelKey: kargoapi.CredentialTypeLabelValueGeneric,
								},
							},
							StringData: map[string]string{
								"TOKEN_1": "foo",
								"TOKEN_2": "baz",
							},
						},
					).
					Build(), nil
			},
		},
	)
	require.NoError(t, err)

	s := &server{
		client:                    cl,
		cfg:                       config.ServerConfig{SecretManagementEnabled: true},
		externalValidateProjectFn: validation.ValidateProject,
	}

	_, err = s.UpdateProjectSecret(ctx, connect.NewRequest(&svcv1alpha1.UpdateProjectSecretRequest{
		Project: "kargo-demo",
		Name:    "secret",
		Data: map[string]string{
			"TOKEN_1": "bar",
		},
	}))
	require.NoError(t, err)

	secret := corev1.Secret{}

	require.NoError(t, s.client.Get(ctx, types.NamespacedName{
		Namespace: "kargo-demo",
		Name:      "secret",
	}, &secret))

	secret1, ok := secret.Data["TOKEN_1"]
	require.True(t, ok)
	require.Equal(t, "bar", string(secret1))

	_, ok = secret.Data["TOKEN_2"]
	require.False(t, ok)
}

func TestApplyProjectSecretUpdateToK8sSecret(t *testing.T) {
	baseSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "kargo-demo",
			Name:      "secret",
		},
		Data: map[string][]byte{
			"TOKEN_1": []byte("foo"),
			"TOKEN_2": []byte("bar"),
		},
	}

	t.Run("remove key from project secret", func(t *testing.T) {
		expectedSecret := baseSecret.DeepCopy()
		delete(expectedSecret.Data, "TOKEN_1")
		secret := baseSecret.DeepCopy()
		applyProjectSecretUpdateToK8sSecret(
			secret,
			projectSecret{
				data: map[string]string{
					"TOKEN_2": "bar",
				},
			},
		)
		require.Equal(t, expectedSecret, secret)
	})

	t.Run("add key in project secret", func(t *testing.T) {
		expectedSecret := baseSecret.DeepCopy()
		expectedSecret.Data["TOKEN_3"] = []byte("baz")
		secret := baseSecret.DeepCopy()
		applyProjectSecretUpdateToK8sSecret(secret, projectSecret{
			data: map[string]string{
				"TOKEN_1": "",
				"TOKEN_2": "",
				"TOKEN_3": "baz",
			},
		})
		require.Equal(t, expectedSecret, secret)
	})

	t.Run("edit key in project secret", func(t *testing.T) {
		expectedSecret := baseSecret.DeepCopy()
		expectedSecret.Data["TOKEN_2"] = []byte("baz")
		secret := baseSecret.DeepCopy()
		applyProjectSecretUpdateToK8sSecret(secret, projectSecret{
			data: map[string]string{
				"TOKEN_1": "",
				"TOKEN_2": "baz",
			},
		})
		require.Equal(t, expectedSecret, secret)
	})
}
