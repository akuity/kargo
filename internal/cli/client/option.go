package client

import "github.com/bufbuild/connect-go"

func NewOption(credential string) connect.ClientOption {
	if credential == "" {
		connect.WithClientOptions()
	}
	return connect.WithClientOptions(
		connect.WithInterceptors(
			&authInterceptor{
				credential: credential,
			},
		),
	)
}
