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

func Test_server_adminLogin_logsAttempt(t *testing.T) {
	const testPassword = "admin-password"
	hashedPassword, err := bcrypt.GenerateFromPassword(
		[]byte(testPassword),
		bcrypt.DefaultCost,
	)
	require.NoError(t, err)
	adminCfg := &config.AdminConfig{
		HashedPassword:  string(hashedPassword),
		TokenIssuer:     "test-issuer",
		TokenAudience:   "test-audience",
		TokenTTL:        time.Hour,
		TokenSigningKey: []byte("test-key"),
	}

	testCases := []struct {
		name            string
		cfg             config.ServerConfig
		authHeader      string
		expectedMessage string
		expectedError   string
	}{
		{
			name:            "success",
			cfg:             config.ServerConfig{AdminConfig: adminCfg},
			authHeader:      "Bearer " + testPassword,
			expectedMessage: "login successful",
		},
		{
			name:            "invalid password",
			cfg:             config.ServerConfig{AdminConfig: adminCfg},
			authHeader:      "Bearer wrong-password",
			expectedMessage: "login failed",
			expectedError:   "invalid password",
		},
		{
			name:            "missing authorization header",
			cfg:             config.ServerConfig{AdminConfig: adminCfg},
			expectedMessage: "login failed",
			expectedError:   "Authorization header is required",
		},
		{
			name:            "admin user not enabled",
			authHeader:      "Bearer " + testPassword,
			expectedMessage: "login failed",
			expectedError:   "admin user is not enabled",
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
			if testCase.authHeader != "" {
				req.Header.Set("Authorization", testCase.authHeader)
			}
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = req

			(&server{cfg: testCase.cfg}).adminLogin(c)

			entries := recorded.All()
			require.Len(t, entries, 1)
			require.Equal(t, zapcore.InfoLevel, entries[0].Level)
			require.Equal(t, testCase.expectedMessage, entries[0].Message)
			fields := entries[0].ContextMap()
			require.Equal(t, "admin", fields["loginType"])
			if testCase.expectedError == "" {
				require.NotContains(t, fields, "error")
			} else {
				require.Equal(t, testCase.expectedError, fields["error"])
			}
			for _, v := range fields {
				require.NotContains(t, v, testPassword)
			}
		})
	}
}
