package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	libCreds "github.com/akuity/kargo/pkg/credentials"
	"github.com/akuity/kargo/pkg/server/config"
	"github.com/akuity/kargo/pkg/server/kubernetes"
	"github.com/akuity/kargo/pkg/server/validation"
)

func TestGetRepoCredentials(t *testing.T) {
	testCases := map[string]struct {
		req        *svcv1alpha1.GetRepoCredentialsRequest
		objects    []client.Object
		assertions func(*testing.T, *connect.Response[svcv1alpha1.GetRepoCredentialsResponse], error)
	}{
		"empty name": {
			req: &svcv1alpha1.GetRepoCredentialsRequest{
				Project: "kargo-demo",
				Name:    "",
			},
			objects: []client.Object{
				mustNewObject[corev1.Namespace]("testdata/namespace.yaml"),
			},
			assertions: func(t *testing.T, c *connect.Response[svcv1alpha1.GetRepoCredentialsResponse], err error) {
				require.Error(t, err)
				require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
				require.Nil(t, c)
			},
		},
		"non-existing project": {
			req: &svcv1alpha1.GetRepoCredentialsRequest{
				Project: "kargo-x",
				Name:    "test",
			},
			objects: []client.Object{
				mustNewObject[corev1.Namespace]("testdata/namespace.yaml"),
			},
			assertions: func(t *testing.T, c *connect.Response[svcv1alpha1.GetRepoCredentialsResponse], err error) {
				require.Error(t, err)
				require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
				require.Nil(t, c)
			},
		},
		"non-credential Secret": {
			req: &svcv1alpha1.GetRepoCredentialsRequest{
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
			assertions: func(t *testing.T, c *connect.Response[svcv1alpha1.GetRepoCredentialsResponse], err error) {
				require.Error(t, err)
				require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
				require.Nil(t, c)
			},
		},
		"existing shared Credentials": {
			req: &svcv1alpha1.GetRepoCredentialsRequest{
				Project: "",
				Name:    "test",
			},
			objects: []client.Object{
				mustNewObject[corev1.Namespace]("testdata/namespace.yaml"),
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "kargo-shared-resources",
						Name:      "test",
						Labels: map[string]string{
							kargoapi.LabelKeyCredentialType: "repository",
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
			assertions: func(t *testing.T, c *connect.Response[svcv1alpha1.GetRepoCredentialsResponse], err error) {
				require.NoError(t, err)

				require.NotNil(t, c)
				require.Nil(t, c.Msg.GetRaw())

				require.NotNil(t, c.Msg.GetCredentials())
				require.Equal(t, "kargo-shared-resources", c.Msg.GetCredentials().Namespace)
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
		"existing Project Credentials": {
			req: &svcv1alpha1.GetRepoCredentialsRequest{
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
							kargoapi.LabelKeyCredentialType: "repository",
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
			assertions: func(t *testing.T, c *connect.Response[svcv1alpha1.GetRepoCredentialsResponse], err error) {
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
			req: &svcv1alpha1.GetRepoCredentialsRequest{
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
							kargoapi.LabelKeyCredentialType: "repository",
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
			assertions: func(t *testing.T, c *connect.Response[svcv1alpha1.GetRepoCredentialsResponse], err error) {
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
			req: &svcv1alpha1.GetRepoCredentialsRequest{
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
							kargoapi.LabelKeyCredentialType: "repository",
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
			assertions: func(t *testing.T, c *connect.Response[svcv1alpha1.GetRepoCredentialsResponse], err error) {
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
					) (client.WithWatch, error) {
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
					SecretManagementEnabled:  true,
					SharedResourcesNamespace: "kargo-shared-resources",
				},
			}
			res, err := (svr).GetRepoCredentials(ctx, connect.NewRequest(testCase.req))
			testCase.assertions(t, res, err)
		})
	}
}

func Test_server_getProjectRepoCredentials(t *testing.T) {
	testProject := &kargoapi.Project{
		ObjectMeta: metav1.ObjectMeta{Name: "fake-project"},
	}
	testCreds := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testProject.Name,
			Name:      "fake-credential",
			Labels: map[string]string{
				kargoapi.LabelKeyCredentialType: "git",
			},
		},
	}
	testRESTEndpoint(
		t, &config.ServerConfig{},
		http.MethodGet, "/v1beta1/projects/"+testProject.Name+"/repo-credentials/"+testCreds.Name,
		[]restTestCase{
			{
				name: "Project does not exist",
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name:          "credentials do not exist",
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name: "Secret exists but is not labeled as credentials",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testProject,
					func() *corev1.Secret {
						secret := testCreds.DeepCopy()
						delete(secret.Labels, kargoapi.LabelKeyCredentialType)
						return secret
					}(),
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusConflict, w.Code)
				},
			},
			{
				name: "Secret exists but is labeled as generic credentials",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testProject,
					func() *corev1.Secret {
						secret := testCreds.DeepCopy()
						secret.Labels[kargoapi.LabelKeyCredentialType] =
							kargoapi.LabelValueCredentialTypeGeneric
						return secret
					}(),
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusConflict, w.Code)
				},
			},
			{
				name: "gets credentials",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testProject,
					testCreds,
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)

					// Examine the Secret in the response
					secret := &corev1.Secret{}
					err := json.Unmarshal(w.Body.Bytes(), secret)
					require.NoError(t, err)
					require.Equal(t, testProject.Name, secret.Namespace)
					require.Equal(t, testCreds.Name, secret.Name)
				},
			},
		},
	)
}

func Test_server_getSharedRepoCredentials(t *testing.T) {
	testCreds := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testSharedResourcesNamespace,
			Name:      "fake-credential",
			Labels: map[string]string{
				kargoapi.LabelKeyCredentialType: "git",
			},
		},
	}
	testRESTEndpoint(
		t, &config.ServerConfig{SharedResourcesNamespace: testSharedResourcesNamespace},
		http.MethodGet, "/v1beta1/shared/repo-credentials/"+testCreds.Name,
		[]restTestCase{
			{
				name:          "credentials do not exist",
				clientBuilder: fake.NewClientBuilder(),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name: "Secret exists but is not labeled as credentials",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					func() *corev1.Secret {
						secret := testCreds.DeepCopy()
						delete(secret.Labels, kargoapi.LabelKeyCredentialType)
						return secret
					}(),
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusConflict, w.Code)
				},
			},
			{
				name: "Secret exists but is labeled as generic credentials",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					func() *corev1.Secret {
						secret := testCreds.DeepCopy()
						secret.Labels[kargoapi.LabelKeyCredentialType] =
							kargoapi.LabelValueCredentialTypeGeneric
						return secret
					}(),
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusConflict, w.Code)
				},
			},
			{
				name: "gets credentials",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testCreds,
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)

					// Examine the Secret in the response
					secret := &corev1.Secret{}
					err := json.Unmarshal(w.Body.Bytes(), secret)
					require.NoError(t, err)
					require.Equal(t, testSharedResourcesNamespace, secret.Namespace)
					require.Equal(t, testCreds.Name, secret.Name)
				},
			},
		},
	)
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
