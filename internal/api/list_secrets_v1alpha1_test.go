package api

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/akuity/kargo/internal/api/config"
	"github.com/akuity/kargo/internal/api/kubernetes"
	"github.com/akuity/kargo/internal/api/validation"
	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

func TestListSecrets(t *testing.T) {
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

	// specific secret
	// this shouldn't be in the list
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

	// normal secret
	// this shouldn't be in the list
	require.NoError(
		t,
		s.client.Create(ctx, &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "kargo-demo",
				Name:      "secret-2",
			},
			Data: map[string][]byte{
				"SECRET_FOO": []byte("SECRET_FOO"),
			},
		}),
	)

	// generic secret
	// this should be in the list
	require.NoError(
		t,
		s.client.Create(ctx, s.genericCredentialsToSecret(genericCredentials{
			project: "kargo-demo",
			name:    "secret-3",
			data: map[string]string{
				"SECRET_GENERIC": "SECRET_GENERIC_VALUE",
			},
		})),
	)

	// generic secret
	// this should be in the list
	require.NoError(
		t,
		s.client.Create(ctx, s.genericCredentialsToSecret(genericCredentials{
			project: "kargo-demo",
			name:    "secret-4",
			data: map[string]string{
				"SECRET_GENERIC_4": "SECRET_GENERIC_VALUE_4",
			},
		})),
	)

	secretsResp, err := s.ListSecrets(ctx, connect.NewRequest(&svcv1alpha1.ListSecretsRequest{
		Project: "kargo-demo",
	}))

	require.NoError(t, err)

	secrets := secretsResp.Msg.GetSecrets()
	require.Len(t, secrets, 2)

	require.Equal(t, "secret-3", secrets[0].GetName())
	require.Equal(t, "secret-4", secrets[1].GetName())

	require.Equal(t, redacted, secrets[0].StringData["SECRET_GENERIC"])
	require.Equal(t, redacted, secrets[1].StringData["SECRET_GENERIC_4"])
}
