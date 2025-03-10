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
	"github.com/akuity/kargo/internal/server/config"
	"github.com/akuity/kargo/internal/server/kubernetes"
)

func TestCreateProjectSecret(t *testing.T) {
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

	resp, err := s.CreateProjectSecret(ctx, connect.NewRequest(&svcv1alpha1.CreateProjectSecretRequest{
		Project:     "kargo-demo",
		Name:        "secret",
		Description: "my secret",
		Data: map[string]string{
			"TOKEN_1": "foo",
			"TOKEN_2": "bar",
		},
	}))
	require.NoError(t, err)

	projSecret := resp.Msg.GetSecret()
	assert.Equal(t, "kargo-demo", projSecret.Namespace)
	assert.Equal(t, "secret", projSecret.ObjectMeta.Name)
	assert.Equal(t, "my secret", projSecret.ObjectMeta.Annotations[kargoapi.AnnotationKeyDescription])
	assert.Equal(t, redacted, projSecret.StringData["TOKEN_1"])
	assert.Equal(t, redacted, projSecret.StringData["TOKEN_2"])

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
	assert.Equal(t, "secret", secret.ObjectMeta.Name)
	assert.Equal(t, "my secret", secret.ObjectMeta.Annotations[kargoapi.AnnotationKeyDescription])
	assert.Equal(t, "foo", string(data["TOKEN_1"]))
	assert.Equal(t, "bar", string(data["TOKEN_2"]))
}
