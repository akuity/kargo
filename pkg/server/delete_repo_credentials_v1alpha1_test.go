package server

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/server/config"
	"github.com/akuity/kargo/pkg/server/kubernetes"
	"github.com/akuity/kargo/pkg/server/validation"
)

func TestDeleteRepoCredentials(t *testing.T) {
	testScheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(testScheme))
	require.NoError(t, kargoapi.AddToScheme(testScheme))

	for _, tc := range []struct {
		name              string
		kClient           client.Client
		config            config.ServerConfig
		validateProjectFn func(ctx context.Context, client client.Client, project string) error
		req               *svcv1alpha1.DeleteRepoCredentialsRequest
		assertions        func(t *testing.T,
			resp *connect.Response[svcv1alpha1.DeleteRepoCredentialsResponse],
			err error,
		)
	}{
		{
			name:    "secret management disabled",
			kClient: fake.NewClientBuilder().WithScheme(testScheme).Build(),
			config: config.ServerConfig{
				SecretManagementEnabled: false,
			},
			req: &svcv1alpha1.DeleteRepoCredentialsRequest{},
			assertions: func(
				t *testing.T,
				_ *connect.Response[svcv1alpha1.DeleteRepoCredentialsResponse],
				err error,
			) {
				require.Error(t, err)
				require.Equal(t, connect.CodeUnimplemented, connect.CodeOf(err))
				require.ErrorContains(t, err, "secret management is not enabled")
			},
		},
		{
			name:    "project does not exist",
			kClient: fake.NewClientBuilder().WithScheme(testScheme).Build(),
			config: config.ServerConfig{
				SecretManagementEnabled: true,
			},
			validateProjectFn: validation.ValidateProject,
			req: &svcv1alpha1.DeleteRepoCredentialsRequest{
				Project: "nonexistent-project",
				Name:    "fake-secret",
			},
			assertions: func(
				t *testing.T,
				_ *connect.Response[svcv1alpha1.DeleteRepoCredentialsResponse],
				err error,
			) {
				require.Error(t, err)
				require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
				require.ErrorContains(t, err, "project not found")
			},
		},
		{
			name:    "name is empty",
			kClient: fake.NewClientBuilder().WithScheme(testScheme).Build(),
			validateProjectFn: func(_ context.Context, _ client.Client, _ string) error {
				return nil
			},
			config: config.ServerConfig{
				SecretManagementEnabled: true,
			},
			req: &svcv1alpha1.DeleteRepoCredentialsRequest{
				Project: "some-project",
				Name:    "",
			},
			assertions: func(
				t *testing.T,
				_ *connect.Response[svcv1alpha1.DeleteRepoCredentialsResponse],
				err error,
			) {
				require.Error(t, err)
				require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
				require.ErrorContains(t, err, "name should not be empty")
			},
		},
		{
			name:    "secret does not exist",
			kClient: fake.NewClientBuilder().WithScheme(testScheme).Build(),
			validateProjectFn: func(_ context.Context, _ client.Client, _ string) error {
				return nil
			},
			config: config.ServerConfig{
				SecretManagementEnabled: true,
			},
			req: &svcv1alpha1.DeleteRepoCredentialsRequest{
				Project: "some-project",
				Name:    "nonexistent-secret",
			},
			assertions: func(
				t *testing.T,
				_ *connect.Response[svcv1alpha1.DeleteRepoCredentialsResponse],
				err error,
			) {
				require.Error(t, err)
				require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
				require.ErrorContains(t, err, "get secret")
			},
		},
		{
			name: "returns not found if secret is not labeled as repo credentials",
			kClient: fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "some-project",
						Name:      "fake-secret",
					},
					Data: map[string][]byte{
						"username": []byte("fake-user"),
						"password": []byte("fake-pass"),
					},
				},
			).Build(),
			validateProjectFn: func(_ context.Context, _ client.Client, _ string) error {
				return nil
			},
			config: config.ServerConfig{
				SecretManagementEnabled: true,
			},
			req: &svcv1alpha1.DeleteRepoCredentialsRequest{
				Project: "some-project",
				Name:    "fake-secret",
			},
			assertions: func(
				t *testing.T,
				_ *connect.Response[svcv1alpha1.DeleteRepoCredentialsResponse],
				err error,
			) {
				require.Error(t, err)
				require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
				require.ErrorContains(t, err, "exists, but is not labeled with")
			},
		},
		{
			name: "delete secret failure",
			kClient: fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "some-project",
						Name:      "fake-secret",
						Labels: map[string]string{
							kargoapi.LabelKeyCredentialType: kargoapi.LabelValueCredentialTypeGeneric,
						},
					},
					Data: map[string][]byte{
						"username": []byte("fake-user"),
						"password": []byte("fake-pass"),
					},
				},
			).WithInterceptorFuncs(
				interceptor.Funcs{
					Delete: func(
						_ context.Context,
						_ client.WithWatch,
						_ client.Object,
						_ ...client.DeleteOption,
					) error {
						return errors.New("something went wrong")
					},
				},
			).Build(),
			validateProjectFn: func(_ context.Context, _ client.Client, _ string) error {
				return nil
			},
			config: config.ServerConfig{
				SecretManagementEnabled: true,
			},
			req: &svcv1alpha1.DeleteRepoCredentialsRequest{
				Project: "some-project",
				Name:    "fake-secret",
			},
			assertions: func(
				t *testing.T,
				_ *connect.Response[svcv1alpha1.DeleteRepoCredentialsResponse],
				err error,
			) {
				require.Error(t, err)
				require.Equal(t, connect.CodeInternal, connect.CodeOf(err))
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "success - project unset should default to shared resources namespace",
			kClient: fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "kargo-shared-resources",
						Name:      "fake-secret",
						Labels: map[string]string{
							kargoapi.LabelKeyCredentialType: kargoapi.LabelValueCredentialTypeGeneric,
						},
					},
					Data: map[string][]byte{
						"username": []byte("fake-user"),
						"password": []byte("fake-pass"),
					},
				},
			).Build(),
			config: config.ServerConfig{
				SecretManagementEnabled:  true,
				SharedResourcesNamespace: "kargo-shared-resources",
			},
			req: &svcv1alpha1.DeleteRepoCredentialsRequest{Name: "fake-secret"},
			assertions: func(
				t *testing.T,
				_ *connect.Response[svcv1alpha1.DeleteRepoCredentialsResponse],
				err error,
			) {
				require.NoError(t, err)
			},
		},
		{
			name: "success with project set",
			kClient: fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-namespace",
						Labels: map[string]string{
							kargoapi.LabelKeyProject: kargoapi.LabelValueTrue,
						},
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "test-namespace",
						Name:      "fake-secret",
						Labels: map[string]string{
							kargoapi.LabelKeyCredentialType: kargoapi.LabelValueCredentialTypeGeneric,
						},
					},
					Data: map[string][]byte{
						"username": []byte("fake-user"),
						"password": []byte("fake-pass"),
					},
				},
			).Build(),
			config:            config.ServerConfig{SecretManagementEnabled: true},
			validateProjectFn: validation.ValidateProject,
			req: &svcv1alpha1.DeleteRepoCredentialsRequest{
				Project: "test-namespace",
				Name:    "fake-secret",
			},
			assertions: func(
				t *testing.T,
				_ *connect.Response[svcv1alpha1.DeleteRepoCredentialsResponse],
				err error,
			) {
				require.NoError(t, err)
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			kc, err := kubernetes.NewClient(t.Context(), new(rest.Config), kubernetes.ClientOptions{
				SkipAuthorization: true,
				NewInternalClient: func(_ context.Context, _ *rest.Config, _ *runtime.Scheme) (client.Client, error) {
					return tc.kClient, nil
				},
			})
			require.NoError(t, err)
			resp, err := (&server{
				client:                    kc,
				cfg:                       tc.config,
				externalValidateProjectFn: tc.validateProjectFn,
			}).DeleteRepoCredentials(
				t.Context(),
				&connect.Request[svcv1alpha1.DeleteRepoCredentialsRequest]{Msg: tc.req},
			)
			tc.assertions(t, resp, err)
		})
	}
}
