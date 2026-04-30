package basic

import (
	"context"

	"github.com/akuity/kargo/pkg/credentials"
)

const (
	usernameKey = "username"
	passwordKey = "password"
)

func init() {
	provider := &CredentialProvider{}
	credentials.DefaultProviderRegistry.MustRegister(
		credentials.ProviderRegistration{
			Predicate: provider.Supports,
			Value:     provider,
		},
	)
}

type CredentialProvider struct{}

func (p *CredentialProvider) Supports(
	_ context.Context,
	req credentials.Request,
) (bool, error) {
	return len(req.Data) > 0 &&
		req.Data[usernameKey] != nil &&
		req.Data[passwordKey] != nil, nil
}

func (p *CredentialProvider) GetCredentials(
	_ context.Context,
	req credentials.Request,
) (*credentials.Credentials, error) {
	creds := &credentials.Credentials{
		Username: string(req.Data[usernameKey]),
		Password: string(req.Data[passwordKey]),
	}
	if creds.Username != "" && creds.Password != "" {
		return creds, nil
	}
	return nil, nil
}
