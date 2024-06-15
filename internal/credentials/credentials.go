package credentials

import (
	"context"

	corev1 "k8s.io/api/core/v1"
)

const (
	FieldRepoURL        = "repoURL"
	FieldRepoURLIsRegex = "repoURLIsRegex"
	FieldUsername       = "username"
	FieldPassword       = "password"
)

// Type is a string type used to represent a type of Credentials.
type Type string

// String returns the string representation of a Type.
func (t Type) String() string {
	return string(t)
}

const (
	// TypeGit represents credentials for a Git repository.
	TypeGit Type = "git"
	// TypeHelm represents credentials for a Helm chart repository.
	TypeHelm Type = "helm"
	// TypeImage represents credentials for an image repository.
	TypeImage Type = "image"
)

// Credentials generically represents any type of repository credential.
type Credentials struct {
	// Username identifies a principal, which combined with the value of the
	// Password field, can be used for access to some repository.
	Username string
	// Password, when combined with the principal identified by the Username
	// field, can be used for access to some repository.
	Password string
	// SSHPrivateKey is a private key that can be used for access to some remote
	// repository. This is primarily applicable for Git repositories.
	SSHPrivateKey string
}

type Helper func(
	ctx context.Context,
	project string,
	credType Type,
	repoURL string,
	secret *corev1.Secret,
) (*Credentials, error)
