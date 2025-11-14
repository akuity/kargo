package credentials

import (
	"context"
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

type Request struct {
	Project  string
	Type     Type
	RepoURL  string
	Data     map[string][]byte
	Metadata map[string]string
}

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

// Provider is an interface for providing credentials for a given type,
// repository URL and data values.
type Provider interface {
	// Supports returns true if the Provider can potentially provide credentials
	// for the credential request.
	Supports(context.Context, Request) (bool, error)
	// GetCredentials returns the credentials for the given CredentialRequest. If
	// the Provider cannot provide credentials for the given CredentialRequest, it
	// should return nil.
	GetCredentials(context.Context, Request) (*Credentials, error)
}
