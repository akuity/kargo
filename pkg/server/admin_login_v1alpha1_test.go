package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
	"sigs.k8s.io/controller-runtime/pkg/client"

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
