package git

// RepoCredentials represents the credentials for connecting to a private git
// repository.
type RepoCredentials struct {
	// SSHPrivateKey is a private key that can be used for both reading from and
	// writing to some remote repository.
	SSHPrivateKey string
	// Username identifies a principal, which combined with the value of the
	// Password field, can be used for both reading from and writing to some
	// remote repository.
	Username string
	// Password, when combined with the principal identified by the Username
	// field, can be used for both reading from and writing to some remote
	// repository.
	//
	// #nosec G117 -- This struct is never marshaled.
	Password string
}
