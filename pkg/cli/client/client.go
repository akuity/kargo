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
	"github.com/spf13/pflag"

	"github.com/akuity/kargo/pkg/cli/config"
	"github.com/akuity/kargo/pkg/cli/option"
	"github.com/akuity/kargo/pkg/client/watch"
	"github.com/akuity/kargo/pkg/server"
	kargogen "github.com/akuity/kargo/pkg/x/client/generated"
	"github.com/akuity/kargo/pkg/x/version"
)

// GetClientFromConfig returns a client for the Kargo API server located at
// the address specified in local configuration, using credentials also
// specified in the local configuration.
func GetClientFromConfig(
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
	return GetClient(cfg.APIAddress, cfg.BearerToken, skipTLSVerify)
}

// GetClient returns a client for the Kargo API server located at the
// specified address, authenticating with the specified credential if one is
// provided.
func GetClient(
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

// APIError makes API errors returned by the client presentable. The client's
// GenericOpenAPIError renders only the HTTP status text; the server's
// explanation is in the response body it carries.
func APIError(err error) error {
	genErr := &kargogen.GenericOpenAPIError{}
	if errors.As(err, &genErr) {
		if body := strings.TrimSpace(string(genErr.Body())); body != "" {
			return fmt.Errorf("%s: %s", genErr.Error(), body)
		}
	}
	return err
}

var errNotLoggedIn = errors.New(
	"seems like you are not logged in; please use `kargo login` to " +
		"authenticate or set both the KARGO_API_ADDRESS and KARGO_API_TOKEN " +
		"environment variables",
)

type Options struct {
	InsecureTLS bool
}

// AddFlags adds the flags for the client options to the provided flag set.
func (o *Options) AddFlags(flags *pflag.FlagSet) {
	option.InsecureTLS(flags, &o.InsecureTLS)
}

// versionHeaderTransport wraps an http.RoundTripper and adds the CLI version
// header to every outbound request.
type versionHeaderTransport struct {
	wrapped http.RoundTripper
}

func (t *versionHeaderTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set(server.CLIVersionHeader, version.GetVersion().Version)
	return t.wrapped.RoundTrip(req)
}

// GetWatchClientFromConfig returns a new watch client for the Kargo API server
// located at the address specified in local configuration, using credentials
// also specified in the local configuration.
func GetWatchClientFromConfig(
	ctx context.Context,
	cfg config.CLIConfig,
	opts Options,
) (*watch.Client, error) {
	if cfg.APIAddress == "" || cfg.BearerToken == "" {
		return nil, errNotLoggedIn
	}
	skipTLSVerify := opts.InsecureTLS || cfg.InsecureSkipTLSVerify
	cfg, err := newTokenRefresher().refreshToken(ctx, cfg, skipTLSVerify)
	if err != nil {
		return nil, fmt.Errorf("error refreshing token: %w", err)
	}
	return GetWatchClient(cfg.APIAddress, cfg.BearerToken, skipTLSVerify), nil
}

// GetWatchClient returns a new watch client for the Kargo API server located at
// the specified address.
func GetWatchClient(
	serverAddress string,
	credential string,
	insecureTLS bool,
) *watch.Client {
	httpClient := cleanhttp.DefaultClient()

	// Start with the default transport
	baseTransport := cleanhttp.DefaultTransport()
	if insecureTLS {
		baseTransport.TLSClientConfig = &tls.Config{
			InsecureSkipVerify: true, // nolint: gosec
		}
	}

	// Wrap with version header transport
	httpClient.Transport = &versionHeaderTransport{wrapped: baseTransport}

	return watch.NewClient(serverAddress, httpClient, credential)
}
