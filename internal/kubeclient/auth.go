package kubeclient

import (
	"context"
	"net/http"
	"strings"

	"k8s.io/client-go/rest"
)

// GetCredential implements a hacky method of gleaning the bearer token that is
// used for authentication to a given Kubernetes API server. It works by
// building an HTTP client using the provided *rest.Config as well as a custom
// http.RoundTripper. The custom http.RoundTripper copies the Authorization
// header from an outbound request to the X-Kargo-User-Credential header in the
// corresponding inbound response. This client is used to make a request to the
// Kubernetes API server and the value of the X-Kargo-User-Credential header is
// read from the response and returned.
//
// Note: The reason the token isn't simply read directly from the *rest.Config
// is because that strategy would not account for cases where the bearer token
// is actually supplied by a credential plugin.
//
// This method will not work when authentication to the Kubernetes API server
// is achieved using a client certificate, but that methods of authentication
// does not seem to be widely used beyond kind and k3d.
func GetCredential(ctx context.Context, cfg *rest.Config) (string, error) {
	// This sets a custom round tripper that will copy the Authorization header
	// from the request to the X-Kargo-User-Credential header in the response.
	cfg.Wrap(newAuthorizationHeaderHook)
	rc, err := rest.HTTPClientFor(cfg)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, cfg.Host, nil)
	if err != nil {
		return "", err
	}
	res, err := rc.Do(req)
	defer func() {
		_ = res.Body.Close()
	}()
	if err != nil {
		return "", err
	}
	return strings.TrimPrefix(
		res.Header.Get(xKargoUserCredentialHeader),
		"Bearer ",
	), nil
}
