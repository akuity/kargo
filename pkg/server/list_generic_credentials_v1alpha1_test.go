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
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/server/config"
	"github.com/akuity/kargo/pkg/server/kubernetes"
	"github.com/akuity/kargo/pkg/server/validation"
)

func TestListGenericCredentials(t *testing.T) {
	ctx := context.Background()

	testData := map[string][]byte{
		"PROJECT_SECRET": []byte("Soylent Green is people!"),
	}

	cl, err := kubernetes.NewClient(
		ctx,
		&rest.Config{},
		kubernetes.ClientOptions{
			SkipAuthorization: true,
			NewInternalClient: func(_ context.Context, _ *rest.Config, s *runtime.Scheme) (client.WithWatch, error) {
				return fake.NewClientBuilder().
					WithScheme(s).
					WithObjects(
						mustNewObject[corev1.Namespace]("testdata/namespace.yaml"),
						&corev1.Secret{ // Should not be in the list (not labeled as generic credentials)
							ObjectMeta: metav1.ObjectMeta{
								Namespace: "kargo-demo",
								Name:      "secret-a",
							},
						},
						&corev1.Secret{ // Labeled as generic credentials; should be in the list
							ObjectMeta: metav1.ObjectMeta{
								Namespace: "kargo-demo",
								Name:      "secret-b",
								Labels: map[string]string{
									kargoapi.LabelKeyCredentialType: kargoapi.LabelValueCredentialTypeGeneric,
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

	resp, err := s.ListGenericCredentials(
		ctx,
		connect.NewRequest(&svcv1alpha1.ListGenericCredentialsRequest{Project: "kargo-demo"}),
	)
	require.NoError(t, err)

	credentials := resp.Msg.GetCredentials()
	require.Len(t, credentials, 1)
	require.Equal(t, "secret-b", credentials[0].Name)
	for _, creds := range credentials {
		require.Equal(t, redacted, creds.StringData["PROJECT_SECRET"])
	}
}

func Test_server_listProjectGenericCredentials(t *testing.T) {
	testProject := &kargoapi.Project{
		ObjectMeta: metav1.ObjectMeta{Name: "fake-project"},
	}
	testRESTEndpoint(
		t, &config.ServerConfig{},
		http.MethodGet, "/v1beta1/projects/"+testProject.Name+"/generic-credentials",
		[]restTestCase{
			{
				name: "Project does not exist",
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name:          "no Secrets exist",
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)
					list := &corev1.SecretList{}
					err := json.Unmarshal(w.Body.Bytes(), list)
					require.NoError(t, err)
					require.Empty(t, list.Items)
				},
			},
			{
				name: "lists Secrets",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testProject,
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: testProject.Name,
							Name:      "secret-1",
							Labels: map[string]string{
								kargoapi.LabelKeyCredentialType: kargoapi.LabelValueCredentialTypeGeneric,
							},
						},
					},
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: testProject.Name,
							Name:      "secret-2",
							Labels: map[string]string{
								kargoapi.LabelKeyCredentialType: kargoapi.LabelValueCredentialTypeGeneric,
							},
						},
					},
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)

					// Examine the Secrets in the response
					secrets := &corev1.SecretList{}
					err := json.Unmarshal(w.Body.Bytes(), secrets)
					require.NoError(t, err)
					require.Len(t, secrets.Items, 2)
				},
			},
		},
	)
}

func Test_server_listSystemGenericCredentials(t *testing.T) {
	testRESTEndpoint(
		t, &config.ServerConfig{SystemResourcesNamespace: testSystemResourcesNamespace},
		http.MethodGet, "/v1beta1/system/generic-credentials",
		[]restTestCase{
			{
				name: "no cluster Secrets exist",
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)
					list := &corev1.SecretList{}
					err := json.Unmarshal(w.Body.Bytes(), list)
					require.NoError(t, err)
					require.Empty(t, list.Items)
				},
			},
			{
				name: "lists cluster Secrets",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: testSystemResourcesNamespace,
							Name:      "secret-1",
							Labels: map[string]string{
								kargoapi.LabelKeyCredentialType: kargoapi.LabelValueCredentialTypeGeneric,
							},
						},
					},
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: testSystemResourcesNamespace,
							Name:      "secret-2",
							Labels: map[string]string{
								kargoapi.LabelKeyCredentialType: kargoapi.LabelValueCredentialTypeGeneric,
							},
						},
					},
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)

					// Examine the Secrets in the response
					secrets := &corev1.SecretList{}
					err := json.Unmarshal(w.Body.Bytes(), secrets)
					require.NoError(t, err)
					require.Len(t, secrets.Items, 2)
				},
			},
		},
	)
}

func Test_server_listSharedGenericCredentials(t *testing.T) {
	testRESTEndpoint(
		t, &config.ServerConfig{SharedResourcesNamespace: testSharedResourcesNamespace},
		http.MethodGet, "/v1beta1/shared/generic-credentials",
		[]restTestCase{
			{
				name: "no shared Secrets exist",
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)
					list := &corev1.SecretList{}
					err := json.Unmarshal(w.Body.Bytes(), list)
					require.NoError(t, err)
					require.Empty(t, list.Items)
				},
			},
			{
				name: "lists shared Secrets",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: testSharedResourcesNamespace,
							Name:      "secret-1",
							Labels: map[string]string{
								kargoapi.LabelKeyCredentialType: kargoapi.LabelValueCredentialTypeGeneric,
							},
						},
					},
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: testSharedResourcesNamespace,
							Name:      "secret-2",
							Labels: map[string]string{
								kargoapi.LabelKeyCredentialType: kargoapi.LabelValueCredentialTypeGeneric,
							},
						},
					},
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)

					// Examine the Secrets in the response
					secrets := &corev1.SecretList{}
					err := json.Unmarshal(w.Body.Bytes(), secrets)
					require.NoError(t, err)
					require.Len(t, secrets.Items, 2)
				},
			},
		},
	)
}
