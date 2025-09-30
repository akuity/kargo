package helm

// Credentials represents the credentials for connecting to a private Helm chart
// repository.
type Credentials struct {
	// Username identifies a principal, which combined with the value of the
	// Password field, can be used for reading from some remote repository.
	Username string
	// Password, when combined with the principal identified by the Username
	// field, can be used for both reading from some remote repository.
	Password string
}
