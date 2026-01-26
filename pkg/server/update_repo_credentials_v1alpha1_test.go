package server

import (
	"bytes"
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
	"k8s.io/apimachinery/pkg/types"
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

func TestUpdateRepoCredentials(t *testing.T) {
	ctx := context.Background()

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
						&corev1.Secret{
							ObjectMeta: metav1.ObjectMeta{
								Namespace: "kargo-demo",
								Name:      "secret",
								Labels: map[string]string{
									kargoapi.LabelKeyCredentialType: kargoapi.LabelValueCredentialTypeGit,
								},
							},
							Data: map[string][]byte{
								libCreds.FieldRepoURL:  []byte("https://github.com/example/repo"),
								libCreds.FieldUsername: []byte("user"),
								libCreds.FieldPassword: []byte("pass"),
							},
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

	_, err = s.UpdateRepoCredentials(ctx, connect.NewRequest(&svcv1alpha1.UpdateRepoCredentialsRequest{
		Project:  "kargo-demo",
		Name:     "secret",
		Type:     "helm",
		RepoUrl:  "https://charts.example.com",
		Username: "new-user",
		Password: "new-pass",
	}))
	require.NoError(t, err)

	secret := corev1.Secret{}

	require.NoError(t, s.client.Get(ctx, types.NamespacedName{
		Namespace: "kargo-demo",
		Name:      "secret",
	}, &secret))

	// Verify credential type was updated
	require.Equal(t, kargoapi.LabelValueCredentialTypeHelm, secret.Labels[kargoapi.LabelKeyCredentialType])

	// Verify all data fields were replaced
	require.Equal(t, "https://charts.example.com", string(secret.Data[libCreds.FieldRepoURL]))
	require.Equal(t, "new-user", string(secret.Data[libCreds.FieldUsername]))
	require.Equal(t, "new-pass", string(secret.Data[libCreds.FieldPassword]))
}

func TestApplyUpdateRepoCredentialsRequestToK8sSecret(t *testing.T) {
	baseSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				kargoapi.LabelKeyCredentialType: kargoapi.LabelValueCredentialTypeGit,
			},
			Annotations: map[string]string{
				kargoapi.AnnotationKeyDescription: "old description",
			},
		},
		Data: map[string][]byte{
			libCreds.FieldRepoURL:        []byte("https://github.com/example/repo"),
			libCreds.FieldUsername:       []byte("old-user"),
			libCreds.FieldPassword:       []byte("old-pass"),
			libCreds.FieldRepoURLIsRegex: []byte("true"),
		},
	}

	t.Run("replaces all fields", func(t *testing.T) {
		secret := baseSecret.DeepCopy()
		applyUpdateRepoCredentialsRequestToK8sSecret(
			secret,
			updateRepoCredentialsRequest{
				Description: "new description",
				Type:        kargoapi.LabelValueCredentialTypeHelm,
				RepoURL:     "https://charts.example.com",
				Username:    "new-user",
				Password:    "new-pass",
			},
		)

		require.Equal(t, "new description", secret.Annotations[kargoapi.AnnotationKeyDescription])
		require.Equal(t, kargoapi.LabelValueCredentialTypeHelm, secret.Labels[kargoapi.LabelKeyCredentialType])
		require.Equal(t, "https://charts.example.com", string(secret.Data[libCreds.FieldRepoURL]))
		require.Equal(t, "new-user", string(secret.Data[libCreds.FieldUsername]))
		require.Equal(t, "new-pass", string(secret.Data[libCreds.FieldPassword]))
		// RepoURLIsRegex should not be present since it wasn't set in request
		_, hasRegex := secret.Data[libCreds.FieldRepoURLIsRegex]
		require.False(t, hasRegex)
	})

	t.Run("sets repoUrlIsRegex when true", func(t *testing.T) {
		secret := baseSecret.DeepCopy()
		applyUpdateRepoCredentialsRequestToK8sSecret(
			secret,
			updateRepoCredentialsRequest{
				Type:           kargoapi.LabelValueCredentialTypeGit,
				RepoURL:        "https://github.com/example/.*",
				RepoURLIsRegex: true,
				Username:       "user",
				Password:       "pass",
			},
		)

		require.Equal(t, "true", string(secret.Data[libCreds.FieldRepoURLIsRegex]))
	})

	t.Run("clears description when empty", func(t *testing.T) {
		secret := baseSecret.DeepCopy()
		applyUpdateRepoCredentialsRequestToK8sSecret(
			secret,
			updateRepoCredentialsRequest{
				Description: "",
				Type:        kargoapi.LabelValueCredentialTypeGit,
				RepoURL:     "https://github.com/example/repo",
				Username:    "user",
				Password:    "pass",
			},
		)

		_, hasDescription := secret.Annotations[kargoapi.AnnotationKeyDescription]
		require.False(t, hasDescription)
	})
}

func TestValidateUpdateRepoCredentialsRequest(t *testing.T) {
	tests := []struct {
		name    string
		req     updateRepoCredentialsRequest
		wantErr string
	}{
		{
			name: "valid request",
			req: updateRepoCredentialsRequest{
				Type:     kargoapi.LabelValueCredentialTypeGit,
				RepoURL:  "https://github.com/example/repo",
				Username: "user",
				Password: "pass",
			},
			wantErr: "",
		},
		{
			name: "missing type",
			req: updateRepoCredentialsRequest{
				RepoURL:  "https://github.com/example/repo",
				Username: "user",
				Password: "pass",
			},
			wantErr: "type should not be empty",
		},
		{
			name: "invalid type",
			req: updateRepoCredentialsRequest{
				Type:     "invalid",
				RepoURL:  "https://github.com/example/repo",
				Username: "user",
				Password: "pass",
			},
			wantErr: "type should be one of git, helm, or image",
		},
		{
			name: "missing repoUrl",
			req: updateRepoCredentialsRequest{
				Type:     kargoapi.LabelValueCredentialTypeGit,
				Username: "user",
				Password: "pass",
			},
			wantErr: "repoUrl should not be empty",
		},
		{
			name: "missing username",
			req: updateRepoCredentialsRequest{
				Type:     kargoapi.LabelValueCredentialTypeGit,
				RepoURL:  "https://github.com/example/repo",
				Password: "pass",
			},
			wantErr: "username should not be empty",
		},
		{
			name: "missing password",
			req: updateRepoCredentialsRequest{
				Type:     kargoapi.LabelValueCredentialTypeGit,
				RepoURL:  "https://github.com/example/repo",
				Username: "user",
			},
			wantErr: "password should not be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateUpdateRepoCredentialsRequest(tt.req)
			if tt.wantErr == "" {
				require.NoError(t, err)
			} else {
				require.EqualError(t, err, tt.wantErr)
			}
		})
	}
}

func Test_server_updateProjectRepoCredentials(t *testing.T) {
	testProject := &kargoapi.Project{
		ObjectMeta: metav1.ObjectMeta{Name: "fake-project"},
	}
	const (
		testRepoURL  = "https://github.com/example/repo"
		testUsername = "username"
		testPassword = "password"
	)
	testCreds := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fake-creds",
			Namespace: testProject.Name,
			Labels: map[string]string{
				kargoapi.LabelKeyCredentialType: kargoapi.LabelValueCredentialTypeGit,
			},
		},
		Data: map[string][]byte{
			libCreds.FieldRepoURL:        []byte(testRepoURL),
			libCreds.FieldUsername:       []byte(testUsername),
			libCreds.FieldPassword:       []byte(testPassword),
			libCreds.FieldRepoURLIsRegex: []byte("true"),
		},
	}
	testRESTEndpoint(
		t, &config.ServerConfig{SecretManagementEnabled: true},
		http.MethodPut, "/v1beta1/projects/"+testProject.Name+"/repo-credentials/"+testCreds.Name,
		[]restTestCase{
			{
				name:          "Secret management disabled",
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject),
				serverConfig:  &config.ServerConfig{SecretManagementEnabled: false},
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotImplemented, w.Code)
				},
			},
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
				name: "missing required field type",
				body: func() *bytes.Buffer {
					b, _ := json.Marshal(updateRepoCredentialsRequest{
						RepoURL:  "https://github.com/example/new-repo",
						Username: "new-user",
						Password: "new-pass",
					})
					return bytes.NewBuffer(b)
				}(),
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusBadRequest, w.Code)
				},
			},
			{
				name: "invalid type value",
				body: func() *bytes.Buffer {
					b, _ := json.Marshal(updateRepoCredentialsRequest{
						Type:     "invalid",
						RepoURL:  "https://github.com/example/new-repo",
						Username: "new-user",
						Password: "new-pass",
					})
					return bytes.NewBuffer(b)
				}(),
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusBadRequest, w.Code)
				},
			},
			{
				name: "credentials do not exist",
				body: func() *bytes.Buffer {
					b, _ := json.Marshal(updateRepoCredentialsRequest{
						Type:     kargoapi.LabelValueCredentialTypeHelm,
						RepoURL:  "https://charts.example.com",
						Username: "new-user",
						Password: "new-pass",
					})
					return bytes.NewBuffer(b)
				}(),
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name: "Secret exists but is not labeled as credentials",
				body: func() *bytes.Buffer {
					b, _ := json.Marshal(updateRepoCredentialsRequest{
						Type:     kargoapi.LabelValueCredentialTypeHelm,
						RepoURL:  "https://charts.example.com",
						Username: "new-user",
						Password: "new-pass",
					})
					return bytes.NewBuffer(b)
				}(),
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testProject,
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Name:      testCreds.Name,
							Namespace: testProject.Name,
						},
					},
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusConflict, w.Code)
				},
			},
			{
				name: "updates credentials with replace semantics",
				body: func() *bytes.Buffer {
					b, _ := json.Marshal(updateRepoCredentialsRequest{
						Description: "updated credentials",
						Type:        kargoapi.LabelValueCredentialTypeHelm,
						RepoURL:     "https://charts.example.com",
						Username:    "new-user",
						Password:    "new-pass",
					})
					return bytes.NewBuffer(b)
				}(),
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testProject,
					testCreds,
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, c client.Client) {
					require.Equal(t, http.StatusOK, w.Code)

					resSecret := &corev1.Secret{}
					err := json.Unmarshal(w.Body.Bytes(), resSecret)
					require.NoError(t, err)
					require.Equal(t, testProject.Name, resSecret.Namespace)
					require.Equal(t, testCreds.Name, resSecret.Name)
					require.Equal(
						t,
						map[string]string{
							kargoapi.LabelKeyCredentialType: kargoapi.LabelValueCredentialTypeHelm,
						},
						resSecret.Labels,
					)
					// Password should be redacted in response
					require.Equal(t, redacted, resSecret.StringData[libCreds.FieldPassword])

					// Verify the Secret was updated in the cluster
					secret := &corev1.Secret{}
					err = c.Get(
						t.Context(),
						client.ObjectKeyFromObject(resSecret),
						secret,
					)
					require.NoError(t, err)
					require.Equal(t, "updated credentials", secret.Annotations[kargoapi.AnnotationKeyDescription])
					require.Equal(t, kargoapi.LabelValueCredentialTypeHelm, secret.Labels[kargoapi.LabelKeyCredentialType])
					require.Equal(t, "https://charts.example.com", string(secret.Data[libCreds.FieldRepoURL]))
					require.Equal(t, "new-user", string(secret.Data[libCreds.FieldUsername]))
					require.Equal(t, "new-pass", string(secret.Data[libCreds.FieldPassword]))
					// RepoURLIsRegex should be removed (replace semantics)
					_, hasRegex := secret.Data[libCreds.FieldRepoURLIsRegex]
					require.False(t, hasRegex)
				},
			},
		},
	)
}

func Test_server_updateSharedRepoCredentials(t *testing.T) {
	const (
		testRepoURL  = "https://github.com/example/repo"
		testUsername = "username"
		testPassword = "password"
	)
	testCreds := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fake-creds",
			Namespace: testSharedResourcesNamespace,
			Labels: map[string]string{
				kargoapi.LabelKeyCredentialType: kargoapi.LabelValueCredentialTypeGit,
			},
		},
		Data: map[string][]byte{
			libCreds.FieldRepoURL:        []byte(testRepoURL),
			libCreds.FieldUsername:       []byte(testUsername),
			libCreds.FieldPassword:       []byte(testPassword),
			libCreds.FieldRepoURLIsRegex: []byte("true"),
		},
	}
	testRESTEndpoint(
		t, &config.ServerConfig{
			SecretManagementEnabled:  true,
			SharedResourcesNamespace: testSharedResourcesNamespace,
		},
		http.MethodPut, "/v1beta1/shared/repo-credentials/"+testCreds.Name,
		[]restTestCase{
			{
				name:         "Secret management disabled",
				serverConfig: &config.ServerConfig{SecretManagementEnabled: false},
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotImplemented, w.Code)
				},
			},
			{
				name:          "invalid JSON in request body",
				body:          bytes.NewBufferString("{invalid json"),
				clientBuilder: fake.NewClientBuilder(),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusBadRequest, w.Code)
				},
			},
			{
				name: "missing required field type",
				body: func() *bytes.Buffer {
					b, _ := json.Marshal(updateRepoCredentialsRequest{
						RepoURL:  "https://github.com/example/new-repo",
						Username: "new-user",
						Password: "new-pass",
					})
					return bytes.NewBuffer(b)
				}(),
				clientBuilder: fake.NewClientBuilder(),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusBadRequest, w.Code)
				},
			},
			{
				name: "credentials do not exist",
				body: func() *bytes.Buffer {
					b, _ := json.Marshal(updateRepoCredentialsRequest{
						Type:     kargoapi.LabelValueCredentialTypeHelm,
						RepoURL:  "https://charts.example.com",
						Username: "new-user",
						Password: "new-pass",
					})
					return bytes.NewBuffer(b)
				}(),
				clientBuilder: fake.NewClientBuilder(),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name: "Secret exists but is not labeled as credentials",
				body: func() *bytes.Buffer {
					b, _ := json.Marshal(updateRepoCredentialsRequest{
						Type:     kargoapi.LabelValueCredentialTypeHelm,
						RepoURL:  "https://charts.example.com",
						Username: "new-user",
						Password: "new-pass",
					})
					return bytes.NewBuffer(b)
				}(),
				clientBuilder: fake.NewClientBuilder().WithObjects(
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Name:      testCreds.Name,
							Namespace: testSharedResourcesNamespace,
						},
					},
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusConflict, w.Code)
				},
			},
			{
				name: "updates credentials with replace semantics",
				body: func() *bytes.Buffer {
					b, _ := json.Marshal(updateRepoCredentialsRequest{
						Description: "updated shared credentials",
						Type:        kargoapi.LabelValueCredentialTypeHelm,
						RepoURL:     "https://charts.example.com",
						Username:    "new-user",
						Password:    "new-pass",
					})
					return bytes.NewBuffer(b)
				}(),
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testCreds,
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, c client.Client) {
					require.Equal(t, http.StatusOK, w.Code)

					resSecret := &corev1.Secret{}
					err := json.Unmarshal(w.Body.Bytes(), resSecret)
					require.NoError(t, err)
					require.Equal(t, testSharedResourcesNamespace, resSecret.Namespace)
					require.Equal(t, testCreds.Name, resSecret.Name)
					require.Equal(
						t,
						map[string]string{
							kargoapi.LabelKeyCredentialType: kargoapi.LabelValueCredentialTypeHelm,
						},
						resSecret.Labels,
					)
					// Password should be redacted in response
					require.Equal(t, redacted, resSecret.StringData[libCreds.FieldPassword])

					// Verify the Secret was updated in the cluster
					secret := &corev1.Secret{}
					err = c.Get(
						t.Context(),
						client.ObjectKeyFromObject(resSecret),
						secret,
					)
					require.NoError(t, err)
					require.Equal(t, "updated shared credentials", secret.Annotations[kargoapi.AnnotationKeyDescription])
					require.Equal(t, kargoapi.LabelValueCredentialTypeHelm, secret.Labels[kargoapi.LabelKeyCredentialType])
					require.Equal(t, "https://charts.example.com", string(secret.Data[libCreds.FieldRepoURL]))
					require.Equal(t, "new-user", string(secret.Data[libCreds.FieldUsername]))
					require.Equal(t, "new-pass", string(secret.Data[libCreds.FieldPassword]))
					// RepoURLIsRegex should be removed (replace semantics)
					_, hasRegex := secret.Data[libCreds.FieldRepoURLIsRegex]
					require.False(t, hasRegex)
				},
			},
		},
	)
}
