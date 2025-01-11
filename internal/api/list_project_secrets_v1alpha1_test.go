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

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/api/config"
	"github.com/akuity/kargo/internal/api/kubernetes"
	"github.com/akuity/kargo/internal/api/validation"
	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

func TestListProjectSecrets(t *testing.T) {
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
		client:                    cl,
		cfg:                       config.ServerConfig{SecretManagementEnabled: true},
		externalValidateProjectFn: validation.ValidateProject,
	}

	// not labeled as a project secret
	// this shouldn't be in the list
	err = s.client.Create(
		ctx,
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "kargo-demo",
				Name:      "secret-1",
			},
		},
	)
	require.NoError(t, err)

	// project secret
	// this should be in the list
	err = s.client.Create(
		ctx,
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "kargo-demo",
				Name:      "secret-2",
				Labels: map[string]string{
					kargoapi.ProjectSecretLabelKey: kargoapi.LabelTrueValue,
				},
			},
			Data: map[string][]byte{
				"PROJECT_SECRET": []byte("PROJECT_SECRET_VALUE"),
			},
		},
	)
	require.NoError(t, err)

	resp, err := s.ListProjectSecrets(
		ctx,
		connect.NewRequest(&svcv1alpha1.ListProjectSecretsRequest{Project: "kargo-demo"}),
	)
	require.NoError(t, err)

	secrets := resp.Msg.GetSecrets()
	require.Len(t, secrets, 1)
	require.Equal(t, "secret-2", secrets[0].GetName())
	require.Empty(t, secrets[0].Data)
	require.Equal(t, redacted, secrets[0].StringData["PROJECT_SECRET"])
}
