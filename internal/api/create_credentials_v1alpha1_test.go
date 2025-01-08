package api

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/api/config"
	"github.com/akuity/kargo/internal/api/kubernetes"
	libCreds "github.com/akuity/kargo/internal/credentials"
	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

func TestCreateCredentials(t *testing.T) {

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
		client: cl,
		cfg:    cfg,
	}

	t.Run("create repo secret", func(t *testing.T) {
		t.Parallel()

		resp, err := s.CreateCredentials(ctx, connect.NewRequest(&svcv1alpha1.CreateCredentialsRequest{
			Project:     "kargo-demo",
			Name:        "repo",
			Description: "my repo secret",
			Type:        "git",
			RepoUrl:     "https://github.com/foo/bar",
			Username:    "username",
			Password:    "password",
		}))

		require.NoError(t, err)

		respSecret := resp.Msg.GetCredentials()

		assert.Equal(t, "kargo-demo", respSecret.Namespace)
		assert.Equal(t, "repo", respSecret.ObjectMeta.Name)
		assert.Equal(t, "my repo secret", respSecret.ObjectMeta.Annotations[kargoapi.AnnotationKeyDescription])
		assert.Equal(t, "https://github.com/foo/bar", respSecret.StringData[libCreds.FieldRepoURL])
		assert.Equal(t, "username", respSecret.StringData[libCreds.FieldUsername])
		assert.Equal(t, redacted, respSecret.StringData[libCreds.FieldPassword])

		kubernetesSecret := corev1.Secret{}

		require.NoError(t, cl.Get(ctx, types.NamespacedName{
			Namespace: "kargo-demo",
			Name:      "repo",
		}, &kubernetesSecret),
		)

		d := kubernetesSecret.DeepCopy().Data

		assert.Equal(t, "kargo-demo", kubernetesSecret.Namespace)
		assert.Equal(t, "repo", kubernetesSecret.ObjectMeta.Name)
		assert.Equal(t, "my repo secret", kubernetesSecret.ObjectMeta.Annotations[kargoapi.AnnotationKeyDescription])
		assert.Equal(t, "https://github.com/foo/bar", string(d[libCreds.FieldRepoURL]))
		assert.Equal(t, "username", string(d[libCreds.FieldUsername]))
		assert.Equal(t, "password", string(d[libCreds.FieldPassword]))
	})

	t.Run("validate credentials", func(t *testing.T) {
		t.Parallel()

		invalidCreds := specificCredentials{
			project:  "",
			name:     "test",
			credType: "git",
			repoURL:  "abc",
			username: "test",
			password: "test",
		}

		err := s.validateCredentials(invalidCreds)

		require.Error(t, err)

		invalidCreds = specificCredentials{
			project:  "kargo-demo",
			name:     "",
			credType: "git",
			repoURL:  "abc",
			username: "test",
			password: "test",
		}

		err = s.validateCredentials(invalidCreds)

		require.Error(t, err)

		invalidCreds = specificCredentials{
			project:  "kargo-demo",
			name:     "test",
			credType: "",
			repoURL:  "abc",
			username: "test",
			password: "test",
		}

		err = s.validateCredentials(invalidCreds)

		require.Error(t, err)

		invalidCreds = specificCredentials{
			project:  "kargo-demo",
			name:     "test",
			credType: "invalid",
			repoURL:  "abc",
			username: "test",
			password: "test",
		}

		err = s.validateCredentials(invalidCreds)

		require.Error(t, err)

		invalidCreds = specificCredentials{
			project:  "kargo-demo",
			name:     "test",
			credType: "git",
			repoURL:  "",
			username: "test",
			password: "test",
		}

		err = s.validateCredentials(invalidCreds)

		require.Error(t, err)

		invalidCreds = specificCredentials{
			project:  "kargo-demo",
			name:     "test",
			credType: "git",
			repoURL:  "https://github.com/akuity/kargo",
			username: "",
			password: "test",
		}

		err = s.validateCredentials(invalidCreds)

		require.Error(t, err)

		invalidCreds = specificCredentials{
			project:  "kargo-demo",
			name:     "test",
			credType: "git",
			repoURL:  "https://github.com/akuity/kargo",
			username: "test",
			password: "",
		}

		err = s.validateCredentials(invalidCreds)

		require.Error(t, err)

		validCreds := specificCredentials{
			project:  "kargo-demo",
			name:     "test",
			credType: "git",
			repoURL:  "https://github.com/akuity/kargo",
			username: "test",
			password: "test",
		}

		err = s.validateCredentials(validCreds)

		require.NoError(t, err)
	})

	t.Run("invalid secret", func(t *testing.T) {
		t.Parallel()

		_, err := s.CreateCredentials(ctx, connect.NewRequest(&svcv1alpha1.CreateCredentialsRequest{
			Project:     "kargo-demo",
			Name:        "invalid",
			Description: "my invalid secret",
			Type:        "invalid",
		}))

		require.Error(t, err)
	})
}
