package api

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/api/config"
	"github.com/akuity/kargo/internal/api/kubernetes"
	"github.com/akuity/kargo/internal/api/validation"
	libCreds "github.com/akuity/kargo/internal/credentials"
	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

func TestGetCredentials(t *testing.T) {
	testCases := map[string]struct {
		req        *svcv1alpha1.GetCredentialsRequest
		objects    []client.Object
		assertions func(*testing.T, *connect.Response[svcv1alpha1.GetCredentialsResponse], error)
	}{
		"empty project": {
			req: &svcv1alpha1.GetCredentialsRequest{
				Project: "",
				Name:    "",
			},
			objects: []client.Object{
				mustNewObject[corev1.Namespace]("testdata/namespace.yaml"),
			},
			assertions: func(t *testing.T, c *connect.Response[svcv1alpha1.GetCredentialsResponse], err error) {
				require.Error(t, err)
				require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
				require.Nil(t, c)
			},
		},
		"empty name": {
			req: &svcv1alpha1.GetCredentialsRequest{
				Project: "kargo-demo",
				Name:    "",
			},
			objects: []client.Object{
				mustNewObject[corev1.Namespace]("testdata/namespace.yaml"),
			},
			assertions: func(t *testing.T, c *connect.Response[svcv1alpha1.GetCredentialsResponse], err error) {
				require.Error(t, err)
				require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
				require.Nil(t, c)
			},
		},
		"non-existing project": {
			req: &svcv1alpha1.GetCredentialsRequest{
				Project: "kargo-x",
				Name:    "test",
			},
			objects: []client.Object{
				mustNewObject[corev1.Namespace]("testdata/namespace.yaml"),
			},
			assertions: func(t *testing.T, c *connect.Response[svcv1alpha1.GetCredentialsResponse], err error) {
				require.Error(t, err)
				require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
				require.Nil(t, c)
			},
		},
		"non-credential Secret": {
			req: &svcv1alpha1.GetCredentialsRequest{
				Project: "kargo-demo",
				Name:    "test",
			},
			objects: []client.Object{
				mustNewObject[corev1.Namespace]("testdata/namespace.yaml"),
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "kargo-demo",
						Name:      "test",
					},
				},
			},
			assertions: func(t *testing.T, c *connect.Response[svcv1alpha1.GetCredentialsResponse], err error) {
				require.Error(t, err)
				require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
				require.Nil(t, c)
			},
		},
		"existing Credentials": {
			req: &svcv1alpha1.GetCredentialsRequest{
				Project: "kargo-demo",
				Name:    "test",
			},
			objects: []client.Object{
				mustNewObject[corev1.Namespace]("testdata/namespace.yaml"),
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "kargo-demo",
						Name:      "test",
						Labels: map[string]string{
							kargoapi.CredentialTypeLabelKey: "repository",
						},
						Annotations: map[string]string{
							"last-applied-configuration": "fake-configuration",
						},
					},
					Data: map[string][]byte{
						libCreds.FieldRepoURL: []byte("fake-url"),
						"random-key":          []byte("random-value"),
					},
				},
			},
			assertions: func(t *testing.T, c *connect.Response[svcv1alpha1.GetCredentialsResponse], err error) {
				require.NoError(t, err)

				require.NotNil(t, c)
				require.Nil(t, c.Msg.GetRaw())

				require.NotNil(t, c.Msg.GetCredentials())
				require.Equal(t, "kargo-demo", c.Msg.GetCredentials().Namespace)
				require.Equal(t, "test", c.Msg.GetCredentials().Name)

				require.Equal(t, map[string]string{
					"last-applied-configuration": redacted,
				}, c.Msg.GetCredentials().Annotations)
				require.Equal(t, map[string]string{
					libCreds.FieldRepoURL: "fake-url",
					"random-key":          redacted,
				}, c.Msg.GetCredentials().StringData)
			},
		},
		"raw format JSON": {
			req: &svcv1alpha1.GetCredentialsRequest{
				Project: "kargo-demo",
				Name:    "test",
				Format:  svcv1alpha1.RawFormat_RAW_FORMAT_JSON,
			},
			objects: []client.Object{
				mustNewObject[corev1.Namespace]("testdata/namespace.yaml"),
				&corev1.Secret{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Secret",
						APIVersion: corev1.SchemeGroupVersion.String(),
					},
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "kargo-demo",
						Name:      "test",
						Labels: map[string]string{
							kargoapi.CredentialTypeLabelKey: "repository",
						},
						Annotations: map[string]string{
							"last-applied-configuration": "fake-configuration",
						},
					},
					Data: map[string][]byte{
						libCreds.FieldRepoURL: []byte("fake-url"),
						"random-key":          []byte("random-value"),
					},
				},
			},
			assertions: func(t *testing.T, c *connect.Response[svcv1alpha1.GetCredentialsResponse], err error) {
				require.NoError(t, err)

				require.NotNil(t, c)
				require.Nil(t, c.Msg.GetCredentials())
				require.NotNil(t, c.Msg.GetRaw())

				obj, _, err := scheme.Codecs.UniversalDeserializer().Decode(
					c.Msg.GetRaw(),
					nil,
					nil,
				)
				require.NoError(t, err)
				tObj, ok := obj.(*corev1.Secret)
				require.True(t, ok)
				require.Equal(t, "kargo-demo", tObj.Namespace)
				require.Equal(t, "test", tObj.Name)

				require.Equal(t, map[string]string{
					"last-applied-configuration": redacted,
				}, tObj.Annotations)
				require.Equal(t, map[string]string{
					libCreds.FieldRepoURL: "fake-url",
					"random-key":          redacted,
				}, tObj.StringData)
			},
		},
		"raw format YAML": {
			req: &svcv1alpha1.GetCredentialsRequest{
				Project: "kargo-demo",
				Name:    "test",
				Format:  svcv1alpha1.RawFormat_RAW_FORMAT_YAML,
			},
			objects: []client.Object{
				mustNewObject[corev1.Namespace]("testdata/namespace.yaml"),
				&corev1.Secret{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Secret",
						APIVersion: corev1.SchemeGroupVersion.String(),
					},
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "kargo-demo",
						Name:      "test",
						Labels: map[string]string{
							kargoapi.CredentialTypeLabelKey: "repository",
						},
						Annotations: map[string]string{
							"last-applied-configuration": "fake-configuration",
						},
					},
					Data: map[string][]byte{
						libCreds.FieldRepoURL: []byte("fake-url"),
						"random-key":          []byte("random-value"),
					},
				},
			},
			assertions: func(t *testing.T, c *connect.Response[svcv1alpha1.GetCredentialsResponse], err error) {
				require.NoError(t, err)

				require.NotNil(t, c)
				require.Nil(t, c.Msg.GetCredentials())
				require.NotNil(t, c.Msg.GetRaw())

				obj, _, err := scheme.Codecs.UniversalDeserializer().Decode(
					c.Msg.GetRaw(),
					nil,
					nil,
				)
				require.NoError(t, err)
				tObj, ok := obj.(*corev1.Secret)
				require.True(t, ok)
				require.Equal(t, "kargo-demo", tObj.Namespace)
				require.Equal(t, "test", tObj.Name)

				require.Equal(t, map[string]string{
					"last-applied-configuration": redacted,
				}, tObj.Annotations)
				require.Equal(t, map[string]string{
					libCreds.FieldRepoURL: "fake-url",
					"random-key":          redacted,
				}, tObj.StringData)
			},
		},
	}
	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()

			client, err := kubernetes.NewClient(
				ctx,
				&rest.Config{},
				kubernetes.ClientOptions{
					SkipAuthorization: true,
					NewInternalClient: func(
						_ context.Context,
						_ *rest.Config,
						scheme *runtime.Scheme,
					) (client.Client, error) {
						c := fake.NewClientBuilder().WithScheme(scheme)
						if len(testCase.objects) > 0 {
							c.WithObjects(testCase.objects...)
						}
						return c.Build(), nil
					},
				},
			)
			require.NoError(t, err)

			svr := &server{
				client:                    client,
				externalValidateProjectFn: validation.ValidateProject,
				cfg: config.ServerConfig{
					SecretManagementEnabled: true,
				},
			}
			res, err := (svr).GetCredentials(ctx, connect.NewRequest(testCase.req))
			testCase.assertions(t, res, err)
		})
	}
}

func TestSanitizeCredentialSecret(t *testing.T) {
	creds := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				"last-applied-configuration": "fake-configuration",
			},
		},
		Data: map[string][]byte{
			libCreds.FieldRepoURL:  []byte("fake-url"),
			libCreds.FieldUsername: []byte("fake-username"),
			libCreds.FieldPassword: []byte("fake-password"),
			"random-key":           []byte("random-value"),
		},
	}
	sanitizedCreds := sanitizeCredentialSecret(creds)
	require.Equal(
		t,
		map[string]string{
			"last-applied-configuration": redacted,
		},
		sanitizedCreds.Annotations,
	)
	require.Equal(
		t,
		map[string]string{
			libCreds.FieldRepoURL:  "fake-url",
			libCreds.FieldUsername: "fake-username",
			libCreds.FieldPassword: redacted,
			"random-key":           redacted,
		},
		sanitizedCreds.StringData,
	)
}
