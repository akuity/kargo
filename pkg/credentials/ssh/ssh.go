package ssh

import (
	"context"

	"github.com/akuity/kargo/pkg/credentials"
)

const sshPrivateKey = "sshPrivateKey"

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
	return len(req.Data) > 0 && req.Data[sshPrivateKey] != nil, nil
}

func (p *CredentialProvider) GetCredentials(
	_ context.Context,
	req credentials.Request,
) (*credentials.Credentials, error) {
	creds := &credentials.Credentials{
		SSHPrivateKey: string(req.Data[sshPrivateKey]),
	}
	if creds.SSHPrivateKey != "" {
		return creds, nil
	}
	return nil, nil
}
