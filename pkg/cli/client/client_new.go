package client

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/hashicorp/go-cleanhttp"

	"github.com/akuity/kargo/pkg/cli/config"
	kargogen "github.com/akuity/kargo/pkg/x/client/generated"
)

// GetNewClientFromConfig returns a client for the Kargo API server located at
// the address specified in local configuration, using credentials also
// specified in the local configuration.
//
// TODO: Rename to GetClientFromConfig once the go-swagger-based function of
// that name is deleted along with the old client.
func GetNewClientFromConfig(
	ctx context.Context,
	cfg config.CLIConfig,
	opts Options,
) (*kargogen.APIClient, error) {
	if cfg.APIAddress == "" || cfg.BearerToken == "" {
		return nil, errNotLoggedIn
	}
	skipTLSVerify := opts.InsecureTLS || cfg.InsecureSkipTLSVerify
	cfg, err := newTokenRefresher().refreshToken(ctx, cfg, skipTLSVerify)
	if err != nil {
		return nil, fmt.Errorf("error refreshing token: %w", err)
	}
	return GetNewClient(cfg.APIAddress, cfg.BearerToken, skipTLSVerify)
}

// GetNewClient returns a client for the Kargo API server located at the
// specified address, authenticating with the specified credential if one is
// provided.
//
// TODO: Rename to GetClient once the go-swagger-based function of that name
// is deleted along with the old client. This file should become client.go at
// the same time.
func GetNewClient(
	serverAddress string,
	credential string,
	insecureTLS bool,
) (*kargogen.APIClient, error) {
	if _, err := url.Parse(serverAddress); err != nil {
		return nil, fmt.Errorf("invalid server address: %w", err)
	}

	baseTransport := cleanhttp.DefaultTransport()
	if insecureTLS {
		baseTransport.TLSClientConfig = &tls.Config{
			InsecureSkipVerify: true, // nolint: gosec
		}
	}

	genCfg := kargogen.NewConfiguration()
	genCfg.Servers = kargogen.ServerConfigurations{
		{URL: strings.TrimSuffix(serverAddress, "/")},
	}
	genCfg.HTTPClient = &http.Client{
		Transport: &versionHeaderTransport{wrapped: baseTransport},
	}
	if credential != "" {
		genCfg.AddDefaultHeader("Authorization", "Bearer "+credential)
	}
	return kargogen.NewAPIClient(genCfg), nil
}

// NewClientAPIError makes API errors returned by the new client presentable.
// The client's GenericOpenAPIError renders only the HTTP status text; the
// server's explanation is in the response body it carries.
//
// TODO: Rename to APIError once the old client is deleted.
func NewClientAPIError(err error) error {
	genErr := &kargogen.GenericOpenAPIError{}
	if errors.As(err, &genErr) {
		if body := strings.TrimSpace(string(genErr.Body())); body != "" {
			return fmt.Errorf("%s: %s", genErr.Error(), body)
		}
	}
	return err
}
