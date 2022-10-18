package bookkeeper

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/akuityio/k8sta/internal/common/version"
	"github.com/pkg/errors"
)

// ClientOptions encapsulates connection options for the Bookkeeper client.
type ClientOptions struct {
	// TODO: Document this
	AllowInsecureConnections bool
}

// Client is an interface for components that can handle bookkeeping requests
// by delegating to a remote server.
type Client interface {
	Service
	// ServerVersion returns version information from the server.
	ServerVersion(context.Context) (version.Version, error)
}

// client is an implementation of the Service interface that handles bookkeeping
// requests by delegating to a remote server.
type client struct {
	address    string
	httpClient *http.Client
}

// NewClient returns an implementation of the Client interface for
// handling bookkeeping requests by delegating to a remote server.
func NewClient(address string, opts *ClientOptions) Client {
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

func (c *client) RenderConfig(
	ctx context.Context,
	req RenderRequest,
) (Response, error) {
	res := Response{}
	return res, c.doRequest(
		ctx,
		http.MethodPost,
		"v1alpha1/render",
		req,
		&res,
	)
}

func (c *client) ServerVersion(ctx context.Context) (version.Version, error) {
	ver := version.Version{}
	return ver, c.doRequest(ctx, http.MethodGet, "version", nil, &ver)
}

func (c *client) doRequest(
	ctx context.Context,
	method string,
	path string,
	body any,
	res any,
) error {
	var reqBodyReader io.Reader
	if body != nil {
		reqBodyBytes, err := json.Marshal(body)
		if err != nil {
			return errors.Wrap(err, "error marshaling HTTP(S) request body")
		}
		reqBodyReader = bytes.NewBuffer(reqBodyBytes)
	}
	httpReq, err := http.NewRequest(
		method,
		fmt.Sprintf("%s/%s", c.address, path),
		reqBodyReader,
	)
	if err != nil {
		return errors.Wrap(err, "error creating HTTP(S) request")
	}
	httpReq = httpReq.WithContext(ctx)
	httpReq.Header.Add("Content-Type", "application/json")
	httpReq.Header.Add("Accept", "application/json")
	httpRes, err := c.httpClient.Do(httpReq)
	if err != nil {
		return errors.Wrap(err, "error making HTTP(S) request")
	}
	if httpRes.StatusCode != http.StatusOK {
		return c.unmarshalToError(httpRes)
	}
	resBodyBytes, err := io.ReadAll(httpRes.Body)
	if err != nil {
		return errors.Wrap(err, "error reading HTTP(S) response body")
	}
	if err = json.Unmarshal(resBodyBytes, res); err != nil {
		return errors.Wrap(err, "error unmarshaling HTTP(S) response body")
	}
	return nil
}

func (c *client) unmarshalToError(res *http.Response) error {
	var resErr error
	switch res.StatusCode {
	case http.StatusBadRequest:
		resErr = &ErrBadRequest{}
	case http.StatusNotFound:
		resErr = &ErrNotFound{}
	case http.StatusConflict:
		resErr = &ErrConflict{}
	case http.StatusNotImplemented:
		resErr = &ErrNotSupported{}
	case http.StatusInternalServerError:
		resErr = &ErrInternalServer{}
	default:
		return errors.Errorf("received %d from Bookkeeper server", res.StatusCode)
	}
	bodyBytes, err := io.ReadAll(res.Body)
	if err != nil {
		return errors.Wrap(err, "error reading error response body")
	}
	if err = json.Unmarshal(bodyBytes, resErr); err != nil {
		return errors.Wrap(err, "error unmarshaling error response body")
	}
	return resErr
}
