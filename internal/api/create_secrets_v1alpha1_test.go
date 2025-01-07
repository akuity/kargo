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

func TestCreateSecrets(t *testing.T) {

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

	t.Run("create generic secret", func(t *testing.T) {
		t.Parallel()

		resp, err := s.CreateSecrets(ctx, connect.NewRequest(&svcv1alpha1.CreateSecretsRequest{
			Project:     "kargo-demo",
			Name:        "external",
			Description: "my external secret",
			Data: map[string]string{
				"TOKEN_1": "foo",
				"TOKEN_2": "bar",
			},
		}))

		require.NoError(t, err)

		respSecret := resp.Msg.GetSecret()

		assert.Equal(t, "kargo-demo", respSecret.Namespace)
		assert.Equal(t, "external", respSecret.ObjectMeta.Name)
		assert.Equal(t, "my external secret", respSecret.ObjectMeta.Annotations[kargoapi.AnnotationKeyDescription])
		assert.Equal(t, redacted, respSecret.StringData["TOKEN_1"])
		assert.Equal(t, redacted, respSecret.StringData["TOKEN_2"])

		kubernetesSecret := corev1.Secret{}

		require.NoError(t, cl.Get(ctx, types.NamespacedName{
			Namespace: "kargo-demo",
			Name:      "external",
		}, &kubernetesSecret),
		)

		d := kubernetesSecret.DeepCopy().Data

		assert.Equal(t, "kargo-demo", kubernetesSecret.Namespace)
		assert.Equal(t, "external", kubernetesSecret.ObjectMeta.Name)
		assert.Equal(t, "my external secret", kubernetesSecret.ObjectMeta.Annotations[kargoapi.AnnotationKeyDescription])
		assert.Equal(t, "foo", string(d["TOKEN_1"]))
		assert.Equal(t, "bar", string(d["TOKEN_2"]))
	})

}
