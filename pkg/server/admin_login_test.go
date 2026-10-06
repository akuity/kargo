package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
	"golang.org/x/crypto/bcrypt"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/akuity/kargo/pkg/logging"
	"github.com/akuity/kargo/pkg/server/config"
)

func Test_server_adminLogin(t *testing.T) {
	const testPassword = "admin-password"
	hashedPassword, err := bcrypt.GenerateFromPassword(
		[]byte(testPassword),
		bcrypt.DefaultCost,
	)
	require.NoError(t, err)
	testRESTEndpoint(
		t, &config.ServerConfig{
			AdminConfig: &config.AdminConfig{
				HashedPassword:  string(hashedPassword),
				TokenIssuer:     "test-issuer",
				TokenAudience:   "test-audience",
				TokenTTL:        time.Hour,
				TokenSigningKey: []byte("test-key"),
			},
		},
		http.MethodPost, "/v1beta1/login",
		[]restTestCase{
			{
				name:         "admin user not enabled",
				serverConfig: &config.ServerConfig{},
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusForbidden, w.Code)
				},
			},
			{
				name: "missing authorization header",
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusBadRequest, w.Code)
				},
			},
			{
				name:    "invalid authorization format",
				headers: map[string]string{"Authorization": "Invalid"}, // Only Bearer is valid
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusBadRequest, w.Code)
				},
			},
			{
				name:    "invalid password",
				headers: map[string]string{"Authorization": "Bearer wrong-password"},
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusForbidden, w.Code)
				},
			},
			{
				name:    "success",
				headers: map[string]string{"Authorization": "Bearer " + testPassword},
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)
				},
			},
		},
	)
}

func Test_server_adminLogin_logsSuccess(t *testing.T) {
	const testPassword = "admin-password"
	hashedPassword, err := bcrypt.GenerateFromPassword(
		[]byte(testPassword),
		bcrypt.DefaultCost,
	)
	require.NoError(t, err)
	s := &server{cfg: config.ServerConfig{
		AdminConfig: &config.AdminConfig{
			HashedPassword:  string(hashedPassword),
			TokenIssuer:     "test-issuer",
			TokenAudience:   "test-audience",
			TokenTTL:        time.Hour,
			TokenSigningKey: []byte("test-key"),
		},
	}}

	testCases := []struct {
		name       string
		password   string
		assertions func(t *testing.T, entries []observer.LoggedEntry)
	}{
		{
			name:     "success is logged",
			password: testPassword,
			assertions: func(t *testing.T, entries []observer.LoggedEntry) {
				require.Len(t, entries, 1)
				require.Equal(t, zapcore.InfoLevel, entries[0].Level)
				require.Equal(t, "admin login successful", entries[0].Message)
			},
		},
		{
			// The request logging middleware records failures.
			name:     "failure is not logged",
			password: "wrong-password",
			assertions: func(t *testing.T, entries []observer.LoggedEntry) {
				require.Empty(t, entries)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			core, recorded := observer.New(zapcore.InfoLevel)
			req := httptest.NewRequest(http.MethodPost, "/v1beta1/login", nil)
			req = req.WithContext(logging.ContextWithLogger(
				req.Context(),
				logging.Wrap(zap.New(core)),
			))
			req.Header.Set("Authorization", "Bearer "+testCase.password)
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = req

			s.adminLogin(c)

			testCase.assertions(t, recorded.All())
		})
	}
}
