package client

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/url"

	httptransport "github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"
	"github.com/hashicorp/go-cleanhttp"
	"github.com/spf13/pflag"

	"github.com/akuity/kargo/pkg/cli/config"
	"github.com/akuity/kargo/pkg/cli/option"
	client "github.com/akuity/kargo/pkg/client/generated"
	"github.com/akuity/kargo/pkg/client/watch"
)

type Options struct {
	InsecureTLS bool
}

// AddFlags adds the flags for the client options to the provided flag set.
func (o *Options) AddFlags(flags *pflag.FlagSet) {
	option.InsecureTLS(flags, &o.InsecureTLS)
}

func GetClientFromConfig(
	ctx context.Context,
	cfg config.CLIConfig,
	opts Options,
) (*client.KargoAPI, error) {
	if cfg.APIAddress == "" || cfg.BearerToken == "" {
		return nil, errors.New(
			"seems like you are not logged in; please use `kargo login` to authenticate",
		)
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

	if insecureTLS {
		transport := cleanhttp.DefaultTransport()
		transport.TLSClientConfig = &tls.Config{
			InsecureSkipVerify: true, // nolint: gosec
		}
		apiClient.Transport.(*httptransport.Runtime).Transport = transport // nolint: forcetypeassert
	}

	// Set authentication if credential is provided
	if credential != "" {
		rt := apiClient.Transport.(*httptransport.Runtime) // nolint: forcetypeassert
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
		return nil, errors.New(
			"seems like you are not logged in; please use `kargo login` to authenticate",
		)
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

	if insecureTLS {
		transport := cleanhttp.DefaultTransport()
		transport.TLSClientConfig = &tls.Config{
			InsecureSkipVerify: true, // nolint: gosec
		}
		httpClient.Transport = transport
	}

	return watch.NewClient(serverAddress, httpClient, credential)
}
