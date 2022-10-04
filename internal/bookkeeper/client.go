package bookkeeper

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/pkg/errors"
)

// ClientOptions encapsulates connection options for the Bookkeeper client.
type ClientOptions struct {
	// TODO: Document this
	AllowInsecureConnections bool
}

// client is an implementation of the Service interface that handles bookkeeping
// requests by delegating to a remote server.
type client struct {
	address    string
	httpClient *http.Client
}

// NewClient returns an implementation of the Service interface for
// handling bookkeeping requests by delegating to a remote server.
func NewClient(address string, opts *ClientOptions) Service {
	if opts == nil {
		opts = &ClientOptions{}
	}
	return &client{
		address: address,
		httpClient: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: opts.AllowInsecureConnections, // nolint: gosec
				},
			},
		},
	}
}

func (c *client) Handle(ctx context.Context, req Request) (Response, error) {
	res := Response{}
	reqBodyBytes, err := json.Marshal(req)
	if err != nil {
		return res, errors.Wrap(err, "error marshaling HTTP(S) request body")
	}
	reqBodyReader := bytes.NewBuffer(reqBodyBytes)
	httpReq, err := http.NewRequest(
		http.MethodPost,
		fmt.Sprintf("%s/v1alpha1", c.address),
		reqBodyReader,
	)
	if err != nil {
		return res, errors.Wrap(err, "error preparing HTTP(S) POST request")
	}
	httpReq = httpReq.WithContext(ctx)
	httpReq.Header.Add("Content-Type", "application/json")
	httpReq.Header.Add("Accept", "application/json")
	httpRes, err := c.httpClient.Do(httpReq)
	if err != nil {
		return res, errors.Wrap(err, "error making HTTP(S) POST request")
	}
	if httpRes.StatusCode != http.StatusOK {
		return res, errors.Wrapf(
			err,
			"HTTP(S) POST request received unexpected error code %d",
			httpRes.StatusCode,
		)
	}
	resBodyBytes, err := io.ReadAll(httpRes.Body)
	if err != nil {
		return res, errors.Wrap(err, "error reading HTTP(S) response body")
	}
	if err = json.Unmarshal(resBodyBytes, &res); err != nil {
		return res, errors.Wrap(err, "error unmarshaling HTTP(S) response body")
	}
	return res, nil
}
