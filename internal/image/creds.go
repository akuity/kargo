package image

import "net/url"

// Credentials represents the credentials for connecting to a private image
// repository. It implements the
// distribution/V3/registry/client/auth.CredentialStore interface.
type Credentials struct {
	// Username identifies a principal, which combined with the value of the
	// Password field, can be used for reading from some image repository.
	Username string
	// Password, when combined with the principal identified by the Username
	// field, can be used for reading from some image repository.
	Password      string
	refreshTokens map[string]string
}

// Basic implements distribution/V3/registry/client/auth.CredentialStore.
func (c Credentials) Basic(*url.URL) (string, string) {
	return c.Username, c.Password
}

// RefreshToken implements distribution/V3/registry/client/auth.CredentialStore.
func (c Credentials) RefreshToken(_ *url.URL, service string) string {
	return c.refreshTokens[service]
}

// SetRefreshToken implements
// distribution/V3/registry/client/auth.CredentialStore.
func (c Credentials) SetRefreshToken(_ *url.URL, service, token string) {
	if c.refreshTokens != nil {
		c.refreshTokens[service] = token
	}
}
