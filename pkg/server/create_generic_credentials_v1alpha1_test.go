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

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/server/config"
	"github.com/akuity/kargo/pkg/server/kubernetes"
)

func TestCreateGenericCredentials(t *testing.T) {
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
		externalValidateProjectFn: func(context.Context, client.Client, string) error {
			return nil
		},
	}

	resp, err := s.CreateGenericCredentials(ctx, connect.NewRequest(&svcv1alpha1.CreateGenericCredentialsRequest{
		Project:     "kargo-demo",
		Name:        "secret",
		Description: "my secret",
		Data: map[string]string{
			"TOKEN_1": "foo",
			"TOKEN_2": "bar",
		},
	}))
	require.NoError(t, err)

	genCreds := resp.Msg.GetCredentials()
	assert.Equal(t, "kargo-demo", genCreds.Namespace)
	assert.Equal(t, "secret", genCreds.Name)
	assert.Equal(t, "my secret", genCreds.Annotations[kargoapi.AnnotationKeyDescription])
	assert.Equal(t, redacted, genCreds.StringData["TOKEN_1"])
	assert.Equal(t, redacted, genCreds.StringData["TOKEN_2"])

	secret := corev1.Secret{}
	err = cl.Get(ctx, types.NamespacedName{
		Namespace: "kargo-demo",
		Name:      "secret",
	},
		&secret,
	)
	require.NoError(t, err)

	data := secret.Data
	assert.Equal(t, "kargo-demo", secret.Namespace)
	assert.Equal(t, "secret", secret.Name)
	assert.Equal(t, "my secret", secret.Annotations[kargoapi.AnnotationKeyDescription])
	assert.Equal(t, "foo", string(data["TOKEN_1"]))
	assert.Equal(t, "bar", string(data["TOKEN_2"]))
}
