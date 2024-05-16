package image

// Credentials represents the credentials for connecting to a private image
// repository.
type Credentials struct {
	// Username identifies a principal, which combined with the value of the
	// Password field, can be used for reading from some image repository.
	Username string
	// Password, when combined with the principal identified by the Username
	// field, can be used for reading from some image repository.
	Password string
}
