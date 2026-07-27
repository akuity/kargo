package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	rbacapi "github.com/akuity/kargo/api/rbac/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func Test_server_createProjectAPIToken(t *testing.T) {
	testProject := &kargoapi.Project{
		ObjectMeta: metav1.ObjectMeta{Name: "fake-project"},
	}
	testSA := &corev1.ServiceAccount{ // Underlying ServiceAccountToken for a Kargo Role virtual resource
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testProject.Name,
			Name:      "fake-role",
		},
	}
	testToken := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testProject.Name,
			Name:      "fake-token",
			Labels: map[string]string{
				rbacapi.LabelKeyAPIToken: rbacapi.LabelValueTrue,
			},
			Annotations: map[string]string{
				"kubernetes.io/service-account.name": testSA.Name,
				rbacapi.AnnotationKeyManaged:         rbacapi.AnnotationValueTrue,
			},
		},
		Data: map[string][]byte{"token": []byte("fake-token-data")},
	}
	testRESTEndpoint(
		t, nil,
		http.MethodPost, "/v1beta1/projects/"+testProject.Name+"/roles/"+testSA.Name+"/api-tokens",
		[]restTestCase{
			{
				name: "Project does not exist",
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name:          "invalid JSON in request body",
				body:          bytes.NewBufferString("{invalid json"),
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusBadRequest, w.Code)
				},
			},
			{
				name:          "missing name in request body",
				body:          mustJSONBody(createAPITokenRequest{}),
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusBadRequest, w.Code)
				},
			},
			{
				name: "ServiceAccount does not exist",
				body: mustJSONBody(createAPITokenRequest{
					Name: testToken.Name,
				}),
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name: "Secret already exists",
				body: mustJSONBody(createAPITokenRequest{
					Name: testToken.Name,
				}),
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testProject,
					testSA,
					testToken,
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusConflict, w.Code)
				},
			},
			{
				name: "creates token",
				body: mustJSONBody(createAPITokenRequest{
					Name: testToken.Name,
				}),
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testProject,
					testSA,
				).WithInterceptorFuncs(interceptor.Funcs{
					// The method under test has a simple retry loop that waits for the
					// new Secret's data to be populated. We need to populate the Secret's
					// data ourselves because the fake client doesn't do it.
					Get: func(
						ctx context.Context,
						client client.WithWatch,
						key client.ObjectKey,
						obj client.Object,
						opts ...client.GetOption,
					) error {
						if s, ok := obj.(*corev1.Secret); ok {
							newS := &corev1.Secret{}
							if err := client.Get(ctx, key, newS); err != nil {
								return err
							}
							newS.Data = testToken.Data
							*s = *newS
							return nil
						}
						return client.Get(ctx, key, obj, opts...)
					},
				}),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, c client.Client) {
					require.Equal(t, http.StatusCreated, w.Code)

					// Examine the Secret in the response
					resSecret := &corev1.Secret{}
					err := json.Unmarshal(w.Body.Bytes(), resSecret)
					require.NoError(t, err)
					require.Equal(t, testProject.Name, resSecret.Namespace)
					require.Equal(t, testToken.Name, resSecret.Name)
					require.Equal(t, testToken.Data, resSecret.Data)

					// Verify the Secret was created in the cluster
					secret := &corev1.Secret{}
					err = c.Get(
						t.Context(),
						client.ObjectKeyFromObject(resSecret),
						secret,
					)
					require.NoError(t, err)
					require.Equal(t, testToken.Data, secret.Data)
				},
			},
		})
}

func Test_server_createSystemAPIToken(t *testing.T) {
	testSA := &corev1.ServiceAccount{ // Underlying ServiceAccountToken for a Kargo Role virtual resource
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testKargoNamespace,
			Name:      "fake-service-account",
			Labels: map[string]string{
				rbacapi.LabelKeySystemRole: rbacapi.LabelValueTrue,
			},
		},
	}
	testToken := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testKargoNamespace,
			Name:      "fake-token",
			Labels: map[string]string{
				rbacapi.LabelKeyAPIToken: rbacapi.LabelValueTrue,
			},
			Annotations: map[string]string{
				"kubernetes.io/service-account.name": testSA.Name,
				rbacapi.AnnotationKeyManaged:         rbacapi.AnnotationValueTrue,
			},
		},
		Data: map[string][]byte{"token": []byte("fake-token-data")},
	}
	testRESTEndpoint(
		t, nil,
		http.MethodPost, "/v1beta1/system/roles/"+testSA.Name+"/api-tokens",
		[]restTestCase{
			{
				name: "invalid JSON in request body",
				body: bytes.NewBufferString("{invalid json"),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusBadRequest, w.Code)
				},
			},
			{
				name: "missing name in request body",
				body: mustJSONBody(createAPITokenRequest{}),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusBadRequest, w.Code)
				},
			},
			{
				name: "ServiceAccount does not exist",
				body: mustJSONBody(createAPITokenRequest{
					Name: testToken.Name,
				}),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name: "Secret already exists",
				body: mustJSONBody(createAPITokenRequest{
					Name: testToken.Name,
				}),
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testSA,
					testToken,
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusConflict, w.Code)
				},
			},
			{
				name: "creates token",
				body: mustJSONBody(createAPITokenRequest{
					Name: testToken.Name,
				}),
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testSA,
				).WithInterceptorFuncs(interceptor.Funcs{
					// The method under test has a simple retry loop that waits for the
					// new Secret's data to be populated. We need populate the Secret's
					// data ourselves because the fake client doesn't do it.
					Get: func(
						ctx context.Context,
						client client.WithWatch,
						key client.ObjectKey,
						obj client.Object,
						opts ...client.GetOption,
					) error {
						if s, ok := obj.(*corev1.Secret); ok {
							newS := &corev1.Secret{}
							if err := client.Get(ctx, key, newS); err != nil {
								return err
							}
							newS.Data = testToken.Data
							*s = *newS
							return nil
						}
						return client.Get(ctx, key, obj, opts...)
					},
				}),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, c client.Client) {
					require.Equal(t, http.StatusCreated, w.Code)

					// Examine the Secret in the response
					resSecret := &corev1.Secret{}
					err := json.Unmarshal(w.Body.Bytes(), resSecret)
					require.NoError(t, err)
					require.Equal(t, testKargoNamespace, resSecret.Namespace)
					require.Equal(t, testToken.Name, resSecret.Name)
					require.Equal(t, testToken.Data, resSecret.Data)

					// Verify the Secret was created in the cluster
					secret := &corev1.Secret{}
					err = c.Get(
						t.Context(),
						client.ObjectKeyFromObject(resSecret),
						secret,
					)
					require.NoError(t, err)
					require.Equal(t, testToken.Data, secret.Data)
				},
			},
		},
	)
}
