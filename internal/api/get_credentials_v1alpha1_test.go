package api

import (
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
)

func TestRedactCredentialSecretValues(t *testing.T) {
	const redacted = "*** REDACTED ***"
	creds := corev1.Secret{
		Data: map[string][]byte{
			"repoURL":        []byte("fake-url"),
			"repoURLPattern": []byte("fake-pattern"),
			"username":       []byte("fake-username"),
			"password":       []byte("fake-password"),
			"random-key":     []byte("random-value"),
		},
	}
	safeCreds := redactCredentialSecretValues(creds)
	require.Equal(
		t,
		map[string]string{
			"repoURL":        "fake-url",
			"repoURLPattern": "fake-pattern",
			"username":       "fake-username",
			"password":       redacted,
			"random-key":     redacted,
		},
		safeCreds.StringData,
	)
}
