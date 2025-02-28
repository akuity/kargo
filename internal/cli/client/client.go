package client

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"

	"connectrpc.com/connect"
	"github.com/hashicorp/go-cleanhttp"
	"github.com/spf13/pflag"

	"github.com/akuity/kargo/internal/cli/config"
	"github.com/akuity/kargo/internal/cli/option"
	"github.com/akuity/kargo/pkg/api/service/v1alpha1/svcv1alpha1connect"
)

type Options struct {
	InsecureTLS bool
}

// AddFlags adds the flags for the client options to the provided flag set.
func (o *Options) AddFlags(flags *pflag.FlagSet) {
	option.InsecureTLS(flags, &o.InsecureTLS)
}

// GetClientFromConfig returns a new client for the Kargo API server located at
// the address specified in local configuration, using credentials also
// specified in the local configuration.
func GetClientFromConfig(
	ctx context.Context,
	cfg config.CLIConfig,
	opts Options,
) (
	svcv1alpha1connect.KargoServiceClient,
	error,
) {
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
	return GetClient(cfg.APIAddress, cfg.BearerToken, skipTLSVerify), nil
}

// GetClient returns a new client for the Kargo API server located at the
// specified address. If the provided credential is non-empty, the client will
// be decorated with an interceptor that adds the credential to outbound
// requests.
func GetClient(
	serverAddress string,
	credential string,
	insecureTLS bool,
) svcv1alpha1connect.KargoServiceClient {
	httpClient := cleanhttp.DefaultClient()

	if insecureTLS {
		transport := cleanhttp.DefaultTransport()
		transport.TLSClientConfig = &tls.Config{
			InsecureSkipVerify: true, // nolint: gosec
		}
		httpClient.Transport = transport
	}

	if credential == "" {
		return svcv1alpha1connect.NewKargoServiceClient(httpClient, serverAddress)
	}
	return svcv1alpha1connect.NewKargoServiceClient(
		httpClient,
		serverAddress,
		connect.WithClientOptions(
			connect.WithInterceptors(
				&authInterceptor{
					credential: credential,
				},
			),
		),
	)
}
