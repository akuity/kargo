package client

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	httptransport "github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"
	"github.com/hashicorp/go-cleanhttp"
	"github.com/spf13/pflag"

	"github.com/akuity/kargo/pkg/cli/config"
	"github.com/akuity/kargo/pkg/cli/option"
	client "github.com/akuity/kargo/pkg/client/generated"
	"github.com/akuity/kargo/pkg/client/watch"
	"github.com/akuity/kargo/pkg/server"
	"github.com/akuity/kargo/pkg/x/version"
)

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

func GetClientFromConfig(
	ctx context.Context,
	cfg config.CLIConfig,
	opts Options,
) (*client.KargoAPI, error) {
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

func GetClient(
	serverAddress string,
	credential string,
	insecureTLS bool,
) (*client.KargoAPI, error) {
	u, err := url.Parse(serverAddress)
	if err != nil {
		return nil, fmt.Errorf("invalid server address: %w", err)
	}

	transportCfg := client.DefaultTransportConfig().
		WithSchemes([]string{u.Scheme}).
		WithHost(u.Host)
	apiClient := client.NewHTTPClientWithConfig(strfmt.Default, transportCfg)

	// Get the runtime to configure transport
	rt, ok := apiClient.Transport.(*httptransport.Runtime)
	if !ok {
		return nil, errors.New("unexpected transport type")
	}

	// Start with the default transport
	baseTransport := cleanhttp.DefaultTransport()
	if insecureTLS {
		baseTransport.TLSClientConfig = &tls.Config{
			InsecureSkipVerify: true, // nolint: gosec
		}
	}

	// Wrap with version header transport
	rt.Transport = &versionHeaderTransport{wrapped: baseTransport}

	// Set authentication if credential is provided
	if credential != "" {
		rt.DefaultAuthentication = httptransport.BearerToken(credential)
	}

	return apiClient, nil
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
