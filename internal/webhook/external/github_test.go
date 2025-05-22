package external

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/indexer"
	"github.com/akuity/kargo/internal/logging"
)

const testSecret = "testsecret" // nolint: gosec

func TestGithubHandler(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(scheme))
	kubeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithIndex(
			&kargoapi.Warehouse{},
			indexer.WarehousesBySubscribedURLsField,
			indexer.WarehousesBySubscribedURLs,
		).Build()
	url := "http://doesntmatter.com"

	for _, test := range []struct {
		name  string
		setup func() *http.Request
		code  int
		msg   string
	}{
		{
			name: "bad request - unsupported event type",
			setup: func() *http.Request {
				b := newBody()
				req := httptest.NewRequest(http.MethodPost, url, b)
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("X-Hub-Signature-256", sign(t, testSecret, b.Bytes()))
				req.Header.Set("X-GitHub-Event", "ping")
				return req
			},
			msg:  "{\"error\":\"only push events are supported\"}\n",
			code: http.StatusNotImplemented,
		},
		{
			name: "request too large",
			setup: func() *http.Request {
				const maxBytes = 2 << 20 // 2MB
				body := make([]byte, maxBytes+1)
				b := io.NopCloser(bytes.NewBuffer(body))
				req := httptest.NewRequest(http.MethodPost, url, b)
				req.Header.Set("X-Hub-Signature-256", sign(t, testSecret, body))
				req.Header.Set("X-GitHub-Event", "push")
				return req
			},
			msg:  "{\"error\":\"response body exceeds limit of 2097152 bytes\"}\n",
			code: http.StatusRequestEntityTooLarge,
		},
		{
			name: "unauthorized - missing signature",
			setup: func() *http.Request {
				b := newBody()
				req := httptest.NewRequest(http.MethodPost, url, b)
				req.Header.Set("X-GitHub-Event", "push")
				return req
			},
			msg:  "{\"error\":\"missing signature\"}\n",
			code: http.StatusUnauthorized,
		},
		{
			name: "unauthorized - invalid signature",
			setup: func() *http.Request {
				b := newBody()
				req := httptest.NewRequest(http.MethodPost, url, b)
				req.Header.Set("X-Hub-Signature-256", sign(t, "invalid-sig", b.Bytes()))
				req.Header.Set("X-GitHub-Event", "push")
				return req
			},
			msg:  "{\"error\":\"unauthorized\"}\n",
			code: http.StatusUnauthorized,
		},
		{
			name: "malformed request",
			setup: func() *http.Request {
				b := bytes.NewBuffer([]byte("invalid json"))
				req := httptest.NewRequest(http.MethodPost, url, b)
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("X-Hub-Signature-256", sign(t, testSecret, b.Bytes()))
				req.Header.Set("X-GitHub-Event", "push")
				return req
			},
			msg:  "{\"error\":\"failed to parse webhook event: invalid character 'i' looking for beginning of value\"}\n",
			code: http.StatusBadRequest,
		},
		{
			name: "OK",
			setup: func() *http.Request {
				b := newBody()
				req := httptest.NewRequest(http.MethodPost, url, b)
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("X-Hub-Signature-256", sign(t, testSecret, b.Bytes()))
				req.Header.Set("X-GitHub-Event", "push")
				return req
			},
			msg:  "{\"msg\":\"refreshed 0 warehouses\"}\n",
			code: http.StatusOK,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			req := test.setup()
			l := logging.NewLogger(logging.DebugLevel)
			ctx := logging.ContextWithLogger(req.Context(), l)
			req = req.WithContext(ctx)
			w := httptest.NewRecorder()
			namespace := "fakenamespace"
			h := githubHandler(kubeClient, namespace, testSecret)
			h(w, req)
			require.Equal(t, test.code, w.Code)
			require.Contains(t, test.msg, w.Body.String())
		})
	}
}

func sign(t *testing.T, s string, b []byte) string {
	t.Helper()

	mac := hmac.New(sha256.New, []byte(s))
	mac.Write(b)
	return fmt.Sprintf("sha256=%s",
		hex.EncodeToString(mac.Sum(nil)),
	)
}

func newBody() *bytes.Buffer {
	return bytes.NewBuffer([]byte(`
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
}
