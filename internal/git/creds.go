package git

// Credentials represents the credentials for connecting to a private Git
// repository.
type Credentials struct {
	// Username identifies a principal, which combined with the value of the
	// Password field, can be used for accessing some Git repository.
	Username string
	// Password, when combined with the principal identified by the Username
	// field, can be used for accessing some Git repository.
	Password string
	// SSHPrivateKey is a private key that can be used for accessing some Git
	// repository.
	SSHPrivateKey string
}
