package server

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
	libCreds "github.com/akuity/kargo/internal/credentials"
	"github.com/akuity/kargo/internal/server/config"
	"github.com/akuity/kargo/internal/server/kubernetes"
	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

func TestCreateCredentials(t *testing.T) {
	ctx := context.Background()

	cl, err := kubernetes.NewClient(
		ctx,
		&rest.Config{},
		kubernetes.ClientOptions{
			SkipAuthorization: true,
			NewInternalClient: func(_ context.Context, _ *rest.Config, s *runtime.Scheme) (client.Client, error) {
				return fake.NewClientBuilder().
					WithScheme(s).
					WithObjects(mustNewObject[corev1.Namespace]("testdata/namespace.yaml")).
					Build(), nil
			},
		},
	)
	require.NoError(t, err)

	s := &server{
		client: cl,
		cfg:    config.ServerConfig{SecretManagementEnabled: true},
	}

	resp, err := s.CreateCredentials(
		ctx,
		connect.NewRequest(
			&svcv1alpha1.CreateCredentialsRequest{
				Project:     "kargo-demo",
				Name:        "creds",
				Description: "my credentials",
				Type:        "git",
				RepoUrl:     "https://github.com/example/repo",
				Username:    "username",
				Password:    "password",
			},
		),
	)
	require.NoError(t, err)

	creds := resp.Msg.GetCredentials()
	assert.Equal(t, "kargo-demo", creds.Namespace)
	assert.Equal(t, "creds", creds.ObjectMeta.Name)
	assert.Equal(t, "my credentials", creds.ObjectMeta.Annotations[kargoapi.AnnotationKeyDescription])
	assert.Equal(t, "https://github.com/example/repo", creds.StringData[libCreds.FieldRepoURL])
	assert.Equal(t, "username", creds.StringData[libCreds.FieldUsername])
	assert.Equal(t, redacted, creds.StringData[libCreds.FieldPassword])

	secret := corev1.Secret{}
	err = cl.Get(
		ctx,
		types.NamespacedName{
			Namespace: "kargo-demo",
			Name:      "creds",
		},
		&secret,
	)
	require.NoError(t, err)

	data := secret.Data
	assert.Equal(t, "kargo-demo", secret.Namespace)
	assert.Equal(t, "creds", secret.ObjectMeta.Name)
	assert.Equal(t, "my credentials", secret.ObjectMeta.Annotations[kargoapi.AnnotationKeyDescription])
	assert.Equal(t, "https://github.com/example/repo", string(data[libCreds.FieldRepoURL]))
	assert.Equal(t, "username", string(data[libCreds.FieldUsername]))
	assert.Equal(t, "password", string(data[libCreds.FieldPassword]))
}

func TestValidateCredentials(t *testing.T) {
	s := &server{}

	err := s.validateCredentials(
		credentials{
			project:  "",
			name:     "test",
			credType: "git",
			repoURL:  "abc",
			username: "test",
			password: "test",
		},
	)
	require.Error(t, err)

	err = s.validateCredentials(
		credentials{
			project:  "kargo-demo",
			name:     "",
			credType: "git",
			repoURL:  "abc",
			username: "test",
			password: "test",
		},
	)
	require.Error(t, err)

	err = s.validateCredentials(
		credentials{
			project:  "kargo-demo",
			name:     "test",
			credType: "",
			repoURL:  "abc",
			username: "test",
			password: "test",
		},
	)
	require.Error(t, err)

	err = s.validateCredentials(
		credentials{
			project:  "kargo-demo",
			name:     "test",
			credType: "invalid",
			repoURL:  "abc",
			username: "test",
			password: "test",
		},
	)
	require.Error(t, err)

	err = s.validateCredentials(
		credentials{
			project:  "kargo-demo",
			name:     "test",
			credType: "git",
			repoURL:  "",
			username: "test",
			password: "test",
		},
	)
	require.Error(t, err)

	err = s.validateCredentials(
		credentials{
			project:  "kargo-demo",
			name:     "test",
			credType: "git",
			repoURL:  "https://github.com/akuity/kargo",
			username: "",
			password: "test",
		},
	)
	require.Error(t, err)

	err = s.validateCredentials(
		credentials{
			project:  "kargo-demo",
			name:     "test",
			credType: "git",
			repoURL:  "https://github.com/akuity/kargo",
			username: "test",
			password: "",
		},
	)
	require.Error(t, err)

	err = s.validateCredentials(
		credentials{
			project:  "kargo-demo",
			name:     "test",
			credType: "git",
			repoURL:  "https://github.com/akuity/kargo",
			username: "test",
			password: "test",
		},
	)
	require.NoError(t, err)
}
