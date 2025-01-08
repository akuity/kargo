package api

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/akuity/kargo/internal/api/config"
	"github.com/akuity/kargo/internal/api/kubernetes"
	"github.com/akuity/kargo/internal/api/validation"
	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

func TestDeleteSecrets(t *testing.T) {
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
		s.client.Create(ctx, credentialsToSecret(specificCredentials{
			project:  "kargo-demo",
			name:     "secret-1",
			credType: "git",
			repoURL:  "https://github.com/akuity/kargo",
			username: "username",
			password: "password",
		})),
	)

	require.NoError(
		t,
		s.client.Create(ctx, s.genericCredentialsToSecret(genericCredentials{
			project: "kargo-demo",
			name:    "secret-2",
			data: map[string]string{
				"secret-key": "secret-value",
			},
		})),
	)

	_, err = s.DeleteSecrets(ctx, connect.NewRequest(&svcv1alpha1.DeleteSecretsRequest{
		Project: "kargo-demo",
		Name:    "secret-1",
	}))

	require.NoError(t, err)

	secret := corev1.Secret{}

	require.Error(
		t,
		s.client.Get(ctx, types.NamespacedName{
			Namespace: "kargo-demo",
			Name:      "secret-1",
		}, &secret),
	)

	_, err = s.DeleteSecrets(ctx, connect.NewRequest(&svcv1alpha1.DeleteSecretsRequest{
		Project: "kargo-demo",
		Name:    "secret-2",
	}))

	require.NoError(t, err)

	require.Error(
		t,
		s.client.Get(ctx, types.NamespacedName{
			Namespace: "kargo-demo",
			Name:      "secret-2",
		}, &secret),
	)
}
