package api

import (
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	libCreds "github.com/akuity/kargo/internal/credentials"
)

func TestSanitizeCredentialSecret(t *testing.T) {
	creds := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				"last-applied-configuration": "fake-configuration",
			},
		},
		Data: map[string][]byte{
			libCreds.FieldRepoURL:  []byte("fake-url"),
			libCreds.FieldUsername: []byte("fake-username"),
			libCreds.FieldPassword: []byte("fake-password"),
			"random-key":           []byte("random-value"),
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
			libCreds.FieldRepoURL:  "fake-url",
			libCreds.FieldUsername: "fake-username",
			libCreds.FieldPassword: redacted,
			"random-key":           redacted,
		},
		sanitizedCreds.StringData,
	)
}
