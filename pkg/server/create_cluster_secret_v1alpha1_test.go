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
	"github.com/akuity/kargo/pkg/server/config"
	"github.com/akuity/kargo/pkg/server/kubernetes"
)

func TestCreateClusterSecret(t *testing.T) {
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
		cfg: config.ServerConfig{
			SecretManagementEnabled: true,
			ClusterSecretNamespace:  "",
		},
	}

	payload := connect.NewRequest(
		&svcv1alpha1.CreateClusterSecretRequest{
			Name: "secret-1",
			Data: map[string]string{
				"foo": "bar",
				"baz": "bax",
			},
		},
	)

	_, err = s.CreateClusterSecret(ctx, payload)
	require.Error(t, err)

	s.cfg.ClusterSecretNamespace = "kargo-cluster-secrts"

	resp, err := s.CreateClusterSecret(
		ctx,
		connect.NewRequest(
			&svcv1alpha1.CreateClusterSecretRequest{
				Name: "secret-1",
				Data: map[string]string{
					"foo": "bar",
					"baz": "bax",
				},
			},
		),
	)
	require.NoError(t, err)

	secret := resp.Msg.GetSecret()
	assert.Equal(t, "kargo-cluster-secrts", secret.Namespace)
	assert.Equal(t, "secret-1", secret.Name)
	assert.Equal(t, redacted, secret.StringData["foo"])
	assert.Equal(t, redacted, secret.StringData["baz"])

	k8sSecret := corev1.Secret{}
	err = cl.Get(
		ctx,
		types.NamespacedName{
			Namespace: "kargo-cluster-secrts",
			Name:      "secret-1",
		},
		&k8sSecret,
	)
	require.NoError(t, err)

	data := k8sSecret.Data
	assert.Equal(t, "bar", string(data["foo"]))
	assert.Equal(t, "bax", string(data["baz"]))
}

func TestValidateSecrets(t *testing.T) {
	s := &server{}

	err := s.validateClusterSecret(clusterSecret{
		name: "",
	})
	require.Error(t, err)

	err = s.validateClusterSecret(clusterSecret{
		name: "foo",
		data: map[string]string{},
	})
	require.Error(t, err)

	err = s.validateClusterSecret(clusterSecret{
		name: "foo",
		data: map[string]string{
			"foo": "bar",
		},
	})
	require.NoError(t, err)
}
