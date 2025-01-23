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

	testData := map[string][]byte{
		"PROJECT_SECRET": []byte("Soylent Green is people!"),
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
						mustNewObject[corev1.Namespace]("testdata/namespace.yaml"),
						&corev1.Secret{ // Should not be in the list (not labeled as a project secret)
							ObjectMeta: metav1.ObjectMeta{
								Namespace: "kargo-demo",
								Name:      "secret-a",
							},
						},
						&corev1.Secret{ // Labeled as a project secret; should be in the list
							ObjectMeta: metav1.ObjectMeta{
								Namespace: "kargo-demo",
								Name:      "secret-b",
								Labels: map[string]string{
									kargoapi.CredentialTypeLabelKey: kargoapi.CredentialTypeLabelGeneric,
								},
							},
							Data: testData,
						},
						&corev1.Secret{ // Labeled with the legacy project secret label; should be in the list
							ObjectMeta: metav1.ObjectMeta{
								Namespace: "kargo-demo",
								Name:      "secret-c",
								Labels: map[string]string{
									kargoapi.ProjectSecretLabelKey: kargoapi.LabelTrueValue,
								},
							},
							Data: testData,
						},
						&corev1.Secret{ // Labeled both ways; should be in the list ONCE
							ObjectMeta: metav1.ObjectMeta{
								Namespace: "kargo-demo",
								Name:      "secret-d",
								Labels: map[string]string{
									kargoapi.CredentialTypeLabelKey: kargoapi.CredentialTypeLabelGeneric,
									kargoapi.ProjectSecretLabelKey:  kargoapi.LabelTrueValue,
								},
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
		client:                    cl,
		cfg:                       config.ServerConfig{SecretManagementEnabled: true},
		externalValidateProjectFn: validation.ValidateProject,
	}

	resp, err := s.ListProjectSecrets(
		ctx,
		connect.NewRequest(&svcv1alpha1.ListProjectSecretsRequest{Project: "kargo-demo"}),
	)
	require.NoError(t, err)

	secrets := resp.Msg.GetSecrets()
	require.Len(t, secrets, 3)
	require.Equal(t, "secret-b", secrets[0].Name)
	require.Equal(t, "secret-c", secrets[1].Name)
	require.Equal(t, "secret-d", secrets[2].Name)
	for _, secret := range secrets {
		require.Equal(t, redacted, secret.StringData["PROJECT_SECRET"])
	}
}
