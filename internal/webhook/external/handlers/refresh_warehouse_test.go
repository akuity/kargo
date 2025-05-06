package handlers

import (
	"bytes"
	"net/http"
	"testing"

	"github.com/akuity/kargo/internal/webhook/external/providers"
)

func TestRefreshWarehouse(t *testing.T) {
	// logger := logging.NewLogger(logging.InfoLevel)
	// kubeClient := fake.NewClientBuilder().Build()
	// httpClient := cleanhttp.DefaultClient()

	for _, test := range []struct {
		name         string
		providerName providers.Name
		req          func() *http.Request
	}{
		{
			name: "success",
			req: func() *http.Request {
				return nil
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {

		})
	}
}

var mockRequestPayload = bytes.NewBuffer([]byte(`
{
	"ref": "refs/heads/main",
	"before": "1fe030abc48d0d0ee7b3d650d6e9449775990318",
	"after": "f12cd167152d80c0a2e28cb45e827c6311bba910",
	"repository": {
	  "html_url": "https://github.com/akuityio/ee-server-poc",
	},
	"pusher": {
	  "name": "akuityio",
	  "email": "fhuskovic92@gmail.com"
	},
	"head_commit": {
	  "id": "f12cd167152d80c0a2e28cb45e827c6311bba910",
	}
  }	
`))
