package api

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

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/api/config"
	"github.com/akuity/kargo/internal/api/kubernetes"
	"github.com/akuity/kargo/internal/api/validation"
	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

func TestApplyGenericCredentialsUpdateToSecret(t *testing.T) {
	ctx := context.Background()

	cfg := config.ServerConfigFromEnv()

	cfg.SecretManagementEnabled = true

	cl, err := kubernetes.NewClient(ctx, &rest.Config{}, kubernetes.ClientOptions{
		SkipAuthorization: true,
		NewInternalClient: func(_ context.Context, _ *rest.Config, s *runtime.Scheme) (client.Client, error) {
			return fake.NewClientBuilder().
				WithScheme(s).
				WithObjects(
					mustNewObject[corev1.Namespace]("testdata/namespace.yaml"),
					mustNewObject[corev1.Namespace]("testdata/stage.yaml"),
				).Build(), nil
		},
	})

	require.NoError(t, err)

	s := &server{
		client:                    cl,
		cfg:                       cfg,
		externalValidateProjectFn: validation.ValidateProject,
	}

	require.NoError(
		t,
		s.client.Create(ctx, s.genericCredentialsToSecret(genericCredentials{
			project: "kargo-demo",
			name:    "secret-1",
			data: map[string]string{
				"TOKEN_1": "foo",
				"TOKEN_2": "baz",
			},
		})),
	)

	_, err = s.UpdateSecrets(ctx, connect.NewRequest(&svcv1alpha1.UpdateSecretsRequest{
		Project: "kargo-demo",
		Name:    "secret-1",
		Data: map[string]string{
			"TOKEN_1": "bar",
		},
	}))

	require.NoError(t, err)

	secret := corev1.Secret{}

	require.NoError(t, s.client.Get(ctx, types.NamespacedName{
		Namespace: "kargo-demo",
		Name:      "secret-1",
	}, &secret))

	secret1, ok := secret.Data["TOKEN_1"]
	require.True(t, ok)
	require.Equal(t, "bar", string(secret1))

	_, ok = secret.Data["TOKEN_2"]
	require.False(t, ok)

	genericSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				kargoapi.CredentialTypeLabelKey: kargoapi.CredentialTypeLabelValueGeneric,
			},
		},
		Data: map[string][]byte{
			"TOKEN_1": []byte("foo"),
			"TOKEN_2": []byte("bar"),
		},
	}

	t.Run("remove key from generic secret", func(t *testing.T) {
		expectedSecret := genericSecret.DeepCopy()
		delete(expectedSecret.Data, "TOKEN_1")
		secret := genericSecret.DeepCopy()

		applyGenericCredentialsUpdateToSecret(secret, genericCredentials{
			data: map[string]string{
				"TOKEN_2": "bar",
			},
		})

		require.Equal(t, expectedSecret, secret)
	})

	t.Run("add key in generic secret", func(t *testing.T) {
		expectedSecret := genericSecret.DeepCopy()
		expectedSecret.Data["TOKEN_3"] = []byte("baz")
		secret := genericSecret.DeepCopy()

		redacted := ""

		applyGenericCredentialsUpdateToSecret(secret, genericCredentials{
			data: map[string]string{
				"TOKEN_1": redacted,
				"TOKEN_2": redacted,
				"TOKEN_3": "baz",
			},
		})

		require.Equal(t, expectedSecret, secret)
	})

	t.Run("edit key in generic secret", func(t *testing.T) {
		expectedSecret := genericSecret.DeepCopy()
		expectedSecret.Data["TOKEN_2"] = []byte("ba")
		secret := genericSecret.DeepCopy()

		redacted := ""

		applyGenericCredentialsUpdateToSecret(secret, genericCredentials{
			data: map[string]string{
				"TOKEN_1": redacted,
				"TOKEN_2": "ba",
			},
		})

		require.Equal(t, expectedSecret, secret)
	})
}
