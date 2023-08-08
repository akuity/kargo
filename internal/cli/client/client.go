package client

import (
	"net/http"

	"github.com/bufbuild/connect-go"

	"github.com/akuity/kargo/internal/cli/config"
	"github.com/akuity/kargo/internal/cli/option"
	"github.com/akuity/kargo/pkg/api/service/v1alpha1/svcv1alpha1connect"
)

// GetClientFromConfig returns a new client for the Kargo API server located at
// the address specified in local configuration, using credentials also
// specified in local configuration UNLESS the specified options indicates that
// the local server should be used instead.
func GetClientFromConfig(opt *option.Option) (
	svcv1alpha1connect.KargoServiceClient,
	error,
) {
	if opt.UseLocalServer {
		return GetClient(opt.ServerURL, ""), nil
	}
	cfg, err := config.LoadCLIConfig()
	if err != nil {
		return nil, err
	}
	return GetClient(cfg.APIAddress, cfg.BearerToken), nil
}

// GetClient returns a new client for the Kargo API server located at the
// specified address. If the provided credential is non-empty, the client will
// be decorated with an interceptor that adds the credential to outbound
// requests.
func GetClient(
	serverAddress string,
	credential string,
) svcv1alpha1connect.KargoServiceClient {
	if credential == "" {
		return svcv1alpha1connect.NewKargoServiceClient(
			http.DefaultClient,
			serverAddress,
		)
	}
	return svcv1alpha1connect.NewKargoServiceClient(
		http.DefaultClient,
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
