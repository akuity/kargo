package github

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

func TestNewApplicationTokenSource(t *testing.T) {
	privateKey, err := generatePrivateKey()
	require.NoError(t, err)

	testCases := []struct {
		name       string
		issuer     string
		privateKey []byte
		assertions func(*testing.T, oauth2.TokenSource, error)
	}{
		{
			name: "issuer is not provided",
			assertions: func(t *testing.T, _ oauth2.TokenSource, err error) {
				require.ErrorContains(t, err, "issuer is required")
			},
		},
		{
			name:   "private key is not provided",
			issuer: "abc",
			assertions: func(t *testing.T, _ oauth2.TokenSource, err error) {
				require.ErrorContains(t, err, "private key is required")
			},
		},
		{
			name:       "valid application token source",
			issuer:     "abc",
			privateKey: privateKey,
			assertions: func(t *testing.T, ts oauth2.TokenSource, err error) {
				require.NoError(t, err)
				require.NotNil(t, ts)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			tokenSource, err := newApplicationTokenSource(
				testCase.issuer,
				testCase.privateKey,
			)
			testCase.assertions(t, tokenSource, err)
		})
	}
}

func generatePrivateKey() ([]byte, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}
	// Encode the private key in the PEM format
	privateKeyPEM := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	}
	return pem.EncodeToMemory(privateKeyPEM), nil
}
