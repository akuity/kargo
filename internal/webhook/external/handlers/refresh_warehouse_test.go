package handlers

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/logging"
	"github.com/akuity/kargo/internal/webhook/external/providers"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestRefreshWarehouseWebhook(t *testing.T) {
	kubeClient := fake.NewClientBuilder().Build()
	require.NoError(t,
		kargoapi.AddToScheme(kubeClient.Scheme()),
	)
	handler := NewRefreshWarehouseWebhook(
		providers.Github,
		logging.NewLogger(logging.InfoLevel),
		kubeClient,
	)
	serverURL := "http://doesntmatter.com"

	for _, test := range []struct {
		name         string
		setup        func() *http.Request
		expectedCode int
	}{
		{
			name: "OK",
			setup: func() *http.Request {
				secret := uuid.New().String()
				os.Setenv("GH_WEBHOOK_SECRET", secret)
				req := httptest.NewRequest(
					http.MethodPost,
					serverURL,
					mockRequestPayload,
				)
				req.Header.Set("X-Hub-Signature-256", sign(t, secret))
				req.Header.Set("X-GitHub-Event", "push")
				return req
			},
			expectedCode: http.StatusOK,
		},
		{
			name: "internal server error",
			setup: func() *http.Request {
				os.Clearenv()
				return httptest.NewRequest(
					http.MethodPost,
					serverURL,
					mockRequestPayload,
				)
			},
			expectedCode: http.StatusInternalServerError,
		},
		{
			name: "unauthorized",
			setup: func() *http.Request {
				os.Setenv("GH_WEBHOOK_SECRET", uuid.New().String())
				return httptest.NewRequest(
					http.MethodPost,
					serverURL,
					mockRequestPayload,
				)
			},
			expectedCode: http.StatusUnauthorized,
		},
		{
			name: "bad request",
			setup: func() *http.Request {
				secret := uuid.New().String()
				os.Setenv("GH_WEBHOOK_SECRET", secret)
				req := httptest.NewRequest(
					http.MethodPost,
					serverURL,
					mockRequestPayload,
				)
				req.Header.Set("X-Hub-Signature-256", sign(t, secret))
				req.Header.Set("X-GitHub-Event", "ping")
				return req
			},
			expectedCode: http.StatusBadRequest,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			req := test.setup()
			w := httptest.NewRecorder()
			handler(w, req)
			require.Equal(t,
				test.expectedCode,
				w.Code,
			)
		})
	}
}

func sign(t *testing.T, secret string) string {
	t.Helper()

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(mockRequestPayload.Bytes())
	return fmt.Sprintf("sha256=%s",
		hex.EncodeToString(mac.Sum(nil)),
	)
}

var mockRequestPayload = bytes.NewBuffer([]byte(`
{
	"ref": "refs/heads/main",
	"before": "1fe030abc48d0d0ee7b3d650d6e9449775990318",
	"after": "f12cd167152d80c0a2e28cb45e827c6311bba910",
	"repository": {
	  "html_url": "https://github.com/username/repo"
	},
	"pusher": {
	  "name": "username",
	  "email": "email@inbox.com"
	},
	"head_commit": {
	  "id": "f12cd167152d80c0a2e28cb45e827c6311bba910"
	}
  }	
`))
