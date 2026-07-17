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
	generatedv2 "github.com/akuity/kargo/pkg/client/generatedv2"
)

// GetClientV2FromConfig returns a v2 (openapi-generator-based) client for the
// Kargo API server located at the address specified in local configuration,
// using credentials also specified in the local configuration.
//
// This exists alongside GetClientFromConfig (the go-swagger-based v1 client)
// during the migration to an openapi-generator-based Go client. Only a
// handful of commands use this so far; everything else still uses
// GetClientFromConfig.
func GetClientV2FromConfig(
	ctx context.Context,
	cfg config.CLIConfig,
	opts Options,
) (*generatedv2.APIClient, error) {
	if cfg.APIAddress == "" || cfg.BearerToken == "" {
		return nil, errNotLoggedIn
	}
	skipTLSVerify := opts.InsecureTLS || cfg.InsecureSkipTLSVerify
	cfg, err := newTokenRefresher().refreshToken(ctx, cfg, skipTLSVerify)
	if err != nil {
		return nil, fmt.Errorf("error refreshing token: %w", err)
	}
	return GetClientV2(cfg.APIAddress, cfg.BearerToken, skipTLSVerify)
}

// GetClientV2 returns a v2 (openapi-generator-based) client for the Kargo API
// server located at the specified address, authenticating with the specified
// credential if one is provided.
func GetClientV2(
	serverAddress string,
	credential string,
	insecureTLS bool,
) (*generatedv2.APIClient, error) {
	if _, err := url.Parse(serverAddress); err != nil {
		return nil, fmt.Errorf("invalid server address: %w", err)
	}

	baseTransport := cleanhttp.DefaultTransport()
	if insecureTLS {
		baseTransport.TLSClientConfig = &tls.Config{
			InsecureSkipVerify: true, // nolint: gosec
		}
	}

	genCfg := generatedv2.NewConfiguration()
	genCfg.Servers = generatedv2.ServerConfigurations{
		{URL: strings.TrimSuffix(serverAddress, "/")},
	}
	genCfg.HTTPClient = &http.Client{
		Transport: &versionHeaderTransport{wrapped: baseTransport},
	}
	if credential != "" {
		genCfg.AddDefaultHeader("Authorization", "Bearer "+credential)
	}
	return generatedv2.NewAPIClient(genCfg), nil
}

// V2APIError makes API errors returned by the v2 client presentable. The
// client's GenericOpenAPIError renders only the HTTP status text; the
// server's explanation is in the response body it carries.
func V2APIError(err error) error {
	genErr := &generatedv2.GenericOpenAPIError{}
	if errors.As(err, &genErr) {
		if body := strings.TrimSpace(string(genErr.Body())); body != "" {
			return fmt.Errorf("%s: %s", genErr.Error(), body)
		}
	}
	return err
}
