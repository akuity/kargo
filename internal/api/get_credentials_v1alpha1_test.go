package api

import (
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestSanitizeCredentialSecret(t *testing.T) {
	creds := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				"last-applied-configuration": "fake-configuration",
			},
		},
		Data: map[string][]byte{
			"repoURL":        []byte("fake-url"),
			"repoURLPattern": []byte("fake-pattern"),
			"username":       []byte("fake-username"),
			"password":       []byte("fake-password"),
			"random-key":     []byte("random-value"),
		},
	}
	sanitizedCreds := sanitizeCredentialSecret(creds)
	require.Equal(
		t,
		map[string]string{
			"last-applied-configuration": redacted,
		},
		sanitizedCreds.Annotations,
	)
	require.Equal(
		t,
		map[string]string{
			"repoURL":        "fake-url",
			"repoURLPattern": "fake-pattern",
			"username":       "fake-username",
			"password":       redacted,
			"random-key":     redacted,
		},
		sanitizedCreds.StringData,
	)
}
