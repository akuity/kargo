package server

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

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	"github.com/akuity/kargo/pkg/server/config"
	"github.com/akuity/kargo/pkg/server/kubernetes"
	"github.com/akuity/kargo/pkg/server/validation"
)

func TestListSystemSecrets(t *testing.T) {
	ctx := context.Background()

	testData := map[string][]byte{
		"SYSTEM_SECRET": []byte("Soylent Green is people!"),
	}

	cl, err := kubernetes.NewClient(
		ctx,
		&rest.Config{},
		kubernetes.ClientOptions{
			SkipAuthorization: true,
			NewInternalClient: func(_ context.Context, _ *rest.Config, s *runtime.Scheme) (client.Client, error) {
				return fake.NewClientBuilder().
					WithScheme(s).
					WithObjects(
						mustNewObject[corev1.Namespace]("testdata/system-resources-namespace.yaml"),
						&corev1.Secret{
							ObjectMeta: metav1.ObjectMeta{
								Namespace: "kargo-system-resources",
								Name:      "secret-a",
							},
						},
						&corev1.Secret{
							ObjectMeta: metav1.ObjectMeta{
								Namespace: "kargo-system-resources",
								Name:      "secret-b",
							},
							Data: testData,
						},
					).
					Build(), nil
			},
		},
	)
	require.NoError(t, err)

	s := &server{
		client: cl,
		cfg: config.ServerConfig{
			SecretManagementEnabled:  true,
			SystemResourcesNamespace: "kargo-system-resources"},
		externalValidateProjectFn: validation.ValidateProject,
	}

	resp, err := s.ListSystemSecrets(
		ctx,
		connect.NewRequest(&svcv1alpha1.ListSystemSecretsRequest{}),
	)
	require.NoError(t, err)

	secrets := resp.Msg.GetSecrets()
	require.Len(t, secrets, 2)

	require.Equal(t, "secret-a", secrets[0].Name)
	require.Equal(t, "secret-b", secrets[1].Name)
	require.Equal(t, redacted, secrets[1].StringData["SYSTEM_SECRET"])
}
