package basic

import (
	"context"

	"github.com/akuity/kargo/internal/credentials"
)

const (
	usernameKey   = "username"
	passwordKey   = "password"
	sshPrivateKey = "sshPrivateKey"
)

type CredentialProvider struct{}

func (*CredentialProvider) Supports(
	_ credentials.Type, _ string, data map[string][]byte,
) bool {
	return len(data) > 0 &&
		(data[usernameKey] != nil && data[passwordKey] != nil) ||
		data[sshPrivateKey] != nil
}

func (p *CredentialProvider) GetCredentials(
	_ context.Context,
	_ string,
	credType credentials.Type,
	repoURL string,
	data map[string][]byte,
) (*credentials.Credentials, error) {
	if !p.Supports(credType, repoURL, data) {
		return nil, nil
	}

	creds := &credentials.Credentials{
		Username:      string(data[usernameKey]),
		Password:      string(data[passwordKey]),
		SSHPrivateKey: string(data[sshPrivateKey]),
	}
	if (creds.Username != "" && creds.Password != "") ||
		creds.SSHPrivateKey != "" {
		return creds, nil
	}
	return nil, nil
}
