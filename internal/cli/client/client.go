package client

import (
	"context"
	"crypto/tls"
	"net/http"

	"connectrpc.com/connect"
	"github.com/pkg/errors"

	"github.com/akuity/kargo/internal/cli/config"
	"github.com/akuity/kargo/internal/cli/option"
	"github.com/akuity/kargo/pkg/api/service/v1alpha1/svcv1alpha1connect"
)

// GetClientFromConfig returns a new client for the Kargo API server located at
// the address specified in local configuration, using credentials also
// specified in local configuration UNLESS the specified options indicates that
// the local server should be used instead.
func GetClientFromConfig(ctx context.Context, opt *option.Option) (
	svcv1alpha1connect.KargoServiceClient,
	error,
) {
	if opt.UseLocalServer {
		return GetClient(opt.LocalServerAddress, "", opt.InsecureTLS), nil
	}
	cfg, err := config.LoadCLIConfig()
	if err != nil {
		return nil, err
	}
	skipTLSVerify := opt.InsecureTLS || cfg.InsecureSkipTLSVerify
	if cfg, err =
		newTokenRefresher().refreshToken(ctx, cfg, skipTLSVerify); err != nil {
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
