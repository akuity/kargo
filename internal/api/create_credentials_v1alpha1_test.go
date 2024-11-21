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
	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

func TestCreateCredentials(t *testing.T) {
	t.Run("create generic secret", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()

		cfg := config.ServerConfigFromEnv()

		cfg.SecretManagementEnabled = true

		client, err := kubernetes.NewClient(ctx, &rest.Config{}, kubernetes.ClientOptions{
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
			client: client,
			cfg:    cfg,
		}

		resp, err := s.CreateCredentials(ctx, connect.NewRequest[svcv1alpha1.CreateCredentialsRequest](&svcv1alpha1.CreateCredentialsRequest{
			Project:     "kargo-demo",
			Name:        "external",
			Description: "my external secret",
			Type:        "generic",
			Data: map[string]string{
				"TOKEN_1": "foo",
				"TOKEN_2": "bar",
			},
		}))

		require.NoError(t, err)

		respSecret := resp.Msg.GetCredentials()

		assert.Equal(t, respSecret.Namespace, "kargo-demo")
		assert.Equal(t, respSecret.ObjectMeta.Name, "external")
		assert.Equal(t, respSecret.ObjectMeta.Annotations[kargoapi.AnnotationKeyDescription], "my external secret")
		assert.Equal(t, redacted, respSecret.StringData["TOKEN_1"])
		assert.Equal(t, redacted, respSecret.StringData["TOKEN_2"])

		kubernetesSecret := corev1.Secret{}

		require.NoError(t, client.Get(ctx, types.NamespacedName{
			Namespace: "kargo-demo",
			Name:      "external",
		}, &kubernetesSecret),
		)

		d := kubernetesSecret.DeepCopy().Data

		assert.Equal(t, kubernetesSecret.Namespace, "kargo-demo")
		assert.Equal(t, kubernetesSecret.ObjectMeta.Name, "external")
		assert.Equal(t, kubernetesSecret.ObjectMeta.Annotations[kargoapi.AnnotationKeyDescription], "my external secret")
		assert.Equal(t, "foo", string(d["TOKEN_1"]))
		assert.Equal(t, "bar", string(d["TOKEN_2"]))
	})
}
