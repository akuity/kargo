package basic

import (
	"context"

	corev1 "k8s.io/api/core/v1"

	"github.com/akuity/kargo/internal/credentials"
)

// SecretToCreds is an implementation of credentials.Helper that simply extracts
// a username, password, and SSH private key from a secret.
func SecretToCreds(
	_ context.Context,
	_ string,
	_ credentials.Type,
	_ string,
	secret *corev1.Secret,
) (*credentials.Credentials, error) {
	if secret == nil {
		// This helper can't handle this
		return nil, nil
	}

	creds := &credentials.Credentials{
		Username:      string(secret.Data["username"]),
		Password:      string(secret.Data["password"]),
		SSHPrivateKey: string(secret.Data["sshPrivateKey"]),
	}
	if (creds.Username != "" && creds.Password != "") ||
		creds.SSHPrivateKey != "" {
		return creds, nil
	}
	return nil, nil
}
