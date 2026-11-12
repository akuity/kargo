package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
	authnv1 "k8s.io/api/authentication/v1"
	authzv1 "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/server/config"
	"github.com/akuity/kargo/pkg/server/kubernetes"
)

func TestSetupRESTRouter_authentication(t *testing.T) {
	gin.SetMode(gin.TestMode)
	const username = "system:serviceaccount:demo:ci-bot"
	kubernetesToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Issuer:  "https://kubernetes.default.svc",
		Subject: username,
	}).SignedString([]byte("test-kubernetes-key"))
	require.NoError(t, err)
	adminConfig := &config.AdminConfig{
		TokenIssuer:     "kargo-admin",
		TokenSigningKey: []byte("test-admin-key"),
	}
	adminToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Issuer:    adminConfig.TokenIssuer,
		Subject:   "admin",
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
	}).SignedString(adminConfig.TokenSigningKey)
	require.NoError(t, err)

	testCases := []struct {
		name              string
		cfg               config.ServerConfig
		path              string
		method            string
		token             string
		tokenAccepted     bool
		allowProjects     bool
		wantStatus        int
		wantTokenReviews  int
		wantAccessReviews int
		wantBody          string
	}{
		{
			name:       "admin and OIDC disabled without token",
			wantStatus: http.StatusUnauthorized,
			wantBody:   "no token provided",
		},
		{
			name:              "admin and OIDC disabled with authorized Kubernetes token",
			token:             kubernetesToken,
			tokenAccepted:     true,
			allowProjects:     true,
			wantStatus:        http.StatusOK,
			wantTokenReviews:  1,
			wantAccessReviews: 1,
			wantBody:          `"name":"demo"`,
		},
		{
			name:              "admin and OIDC disabled with forbidden Kubernetes token",
			token:             kubernetesToken,
			tokenAccepted:     true,
			wantStatus:        http.StatusForbidden,
			wantTokenReviews:  1,
			wantAccessReviews: 1,
		},
		{
			name:             "admin and OIDC disabled with rejected Kubernetes token",
			token:            kubernetesToken,
			wantStatus:       http.StatusUnauthorized,
			wantTokenReviews: 1,
			wantBody:         "invalid token",
		},
		{
			name:             "admin disabled rejects admin token",
			token:            adminToken,
			wantStatus:       http.StatusUnauthorized,
			wantTokenReviews: 1,
			wantBody:         "invalid token",
		},
		{
			name:       "admin enabled accepts admin token",
			cfg:        config.ServerConfig{AdminConfig: adminConfig},
			token:      adminToken,
			wantStatus: http.StatusOK,
			wantBody:   `"name":"demo"`,
		},
		{
			name:       "admin enabled without token",
			cfg:        config.ServerConfig{AdminConfig: &config.AdminConfig{TokenIssuer: "kargo-admin"}},
			wantStatus: http.StatusUnauthorized,
			wantBody:   "no token provided",
		},
		{
			name:              "admin enabled with authorized Kubernetes token",
			cfg:               config.ServerConfig{AdminConfig: &config.AdminConfig{TokenIssuer: "kargo-admin"}},
			token:             kubernetesToken,
			tokenAccepted:     true,
			allowProjects:     true,
			wantStatus:        http.StatusOK,
			wantTokenReviews:  1,
			wantAccessReviews: 1,
			wantBody:          `"name":"demo"`,
		},
		{
			name:       "local mode without token",
			cfg:        config.ServerConfig{LocalMode: true},
			wantStatus: http.StatusUnauthorized,
			wantBody:   "no token provided",
		},
		{
			name:       "public config without token",
			path:       "/v1beta1/system/public-server-config",
			wantStatus: http.StatusOK,
			wantBody:   `"skipAuth":false`,
		},
		{
			name:       "disabled admin login is public",
			method:     http.MethodPost,
			path:       "/v1beta1/login",
			wantStatus: http.StatusForbidden,
			wantBody:   "admin user is not enabled",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			scheme := newTestScheme(t)
			require.NoError(t, authnv1.AddToScheme(scheme))
			require.NoError(t, authzv1.AddToScheme(scheme))
			tokenReviews, accessReviews := 0, 0
			internalClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(&kargoapi.Project{ObjectMeta: metav1.ObjectMeta{Name: "demo"}}).
				WithInterceptorFuncs(interceptor.Funcs{
					Create: func(
						ctx context.Context,
						cl client.WithWatch,
						obj client.Object,
						opts ...client.CreateOption,
					) error {
						switch review := obj.(type) {
						case *authnv1.TokenReview:
							tokenReviews++
							require.Equal(t, testCase.token, review.Spec.Token)
							review.Status.Authenticated = testCase.tokenAccepted
							review.Status.User = authnv1.UserInfo{Username: username}
							return nil
						case *authzv1.SubjectAccessReview:
							accessReviews++
							require.Equal(t, username, review.Spec.User)
							require.NotNil(t, review.Spec.ResourceAttributes)
							require.Equal(t, "list", review.Spec.ResourceAttributes.Verb)
							require.Equal(t, "projects", review.Spec.ResourceAttributes.Resource)
							review.Status.Allowed = testCase.allowProjects
							return nil
						default:
							return cl.Create(ctx, obj, opts...)
						}
					},
				}).Build()
			kubeClient, err := kubernetes.NewClient(
				t.Context(),
				&rest.Config{},
				kubernetes.ClientOptions{
					Scheme: scheme,
					NewInternalClient: func(
						context.Context,
						*rest.Config,
						*runtime.Scheme,
						string,
					) (client.WithWatch, error) {
						return internalClient, nil
					},
				},
			)
			require.NoError(t, err)
			s, ok := NewServer(testCase.cfg, kubeClient, nil, nil, nil).(*server)
			require.True(t, ok)
			router, err := s.setupRESTRouter(t.Context())
			require.NoError(t, err)
			path := testCase.path
			if path == "" {
				path = "/v1beta1/projects"
			}
			method := testCase.method
			if method == "" {
				method = http.MethodGet
			}
			req := httptest.NewRequest(method, path, nil)
			if testCase.token != "" {
				req.Header.Set("Authorization", "Bearer "+testCase.token)
			}
			response := httptest.NewRecorder()
			router.ServeHTTP(response, req)
			require.Equal(t, testCase.wantStatus, response.Code)
			require.Equal(t, testCase.wantTokenReviews, tokenReviews)
			require.Equal(t, testCase.wantAccessReviews, accessReviews)
			if testCase.wantBody != "" {
				require.Contains(t, response.Body.String(), testCase.wantBody)
			}
		})
	}
}
