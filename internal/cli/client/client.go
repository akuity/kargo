package client

import (
	"context"
	"crypto/tls"
	"net/http"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	"github.com/spf13/pflag"

	"github.com/akuity/kargo/internal/cli/config"
	"github.com/akuity/kargo/internal/cli/option"
	"github.com/akuity/kargo/pkg/api/service/v1alpha1/svcv1alpha1connect"
)

type Options struct {
	InsecureTLS bool
	LocalServer bool
}

// AddFlags adds the flags for the client options to the provided flag set.
func (o *Options) AddFlags(flags *pflag.FlagSet) {
	option.InsecureTLS(flags, &o.InsecureTLS)
	option.LocalServer(flags, &o.LocalServer)
}

// GetClientFromConfig returns a new client for the Kargo API server located at
// the address specified in local configuration, using credentials also
// specified in local configuration UNLESS the specified options indicates that
// the local server should be used instead.
func GetClientFromConfig(
	ctx context.Context,
	cfg config.CLIConfig,
	opts Options,
) (
	svcv1alpha1connect.KargoServiceClient,
	error,
) {
	if opts.LocalServer {
		return GetLocalServerClient(ctx, opts)
	}
	if cfg.APIAddress == "" || cfg.BearerToken == "" {
		return nil, errors.New(
			"seems like you are not logged in; please use `kargo login` to authenticate",
		)
	}
	skipTLSVerify := opts.InsecureTLS || cfg.InsecureSkipTLSVerify
	cfg, err := newTokenRefresher().refreshToken(ctx, cfg, skipTLSVerify)
	if err != nil {
		return nil, errors.Wrap(err, "error refreshing token")
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
	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: insecureTLS, // nolint: gosec
			},
		},
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
