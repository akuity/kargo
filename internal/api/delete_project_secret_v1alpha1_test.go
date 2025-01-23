package api

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/api/config"
	"github.com/akuity/kargo/internal/api/kubernetes"
	"github.com/akuity/kargo/internal/api/validation"
	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

func TestDeleteProjectSecret(t *testing.T) {
	ctx := context.Background()

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
						&corev1.Secret{
							ObjectMeta: metav1.ObjectMeta{
								Namespace: "kargo-demo",
								Name:      "secret-a",
								Labels: map[string]string{
									kargoapi.CredentialTypeLabelKey: kargoapi.CredentialTypeLabelGeneric,
								},
							},
						},
						&corev1.Secret{
							ObjectMeta: metav1.ObjectMeta{
								Namespace: "kargo-demo",
								Name:      "secret-b",
								Labels: map[string]string{
									kargoapi.ProjectSecretLabelKey: kargoapi.LabelTrueValue, // Legacy label
								},
							},
						},
					).Build(), nil
			},
		},
	)
	require.NoError(t, err)

	s := &server{
		client:                    cl,
		cfg:                       config.ServerConfig{SecretManagementEnabled: true},
		externalValidateProjectFn: validation.ValidateProject,
	}

	_, err = s.DeleteProjectSecret(
		ctx,
		connect.NewRequest(
			&svcv1alpha1.DeleteProjectSecretRequest{
				Project: "kargo-demo",
				Name:    "secret-a",
			},
		),
	)
	require.NoError(t, err)

	secret := corev1.Secret{}
	err = s.client.Get(
		ctx,
		types.NamespacedName{
			Namespace: "kargo-demo",
			Name:      "secret-a",
		},
		&secret,
	)
	require.Error(t, err)

	_, err = s.DeleteProjectSecret(
		ctx,
		connect.NewRequest(
			&svcv1alpha1.DeleteProjectSecretRequest{
				Project: "kargo-demo",
				Name:    "secret-b", // Has the legacy label
			},
		),
	)
	require.NoError(t, err)

	err = s.client.Get(
		ctx,
		types.NamespacedName{
			Namespace: "kargo-demo",
			Name:      "secret-b",
		},
		&secret,
	)
	require.Error(t, err)

}
