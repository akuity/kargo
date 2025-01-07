package api

import (
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func TestApplyGenericCredentialsUpdateToSecret(t *testing.T) {
	genericSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				kargoapi.CredentialTypeLabelKey: kargoapi.CredentialTypeLabelValueGeneric,
			},
		},
		Data: map[string][]byte{
			"TOKEN_1": []byte("foo"),
			"TOKEN_2": []byte("bar"),
		},
	}

	t.Run("remove key from generic secret", func(t *testing.T) {
		expectedSecret := genericSecret.DeepCopy()
		delete(expectedSecret.Data, "TOKEN_1")
		secret := genericSecret.DeepCopy()

		applyGenericCredentialsUpdateToSecret(secret, genericCredentials{
			data: map[string]string{
				"TOKEN_2": "bar",
			},
		})

		require.Equal(t, expectedSecret, secret)
	})

	t.Run("add key in generic secret", func(t *testing.T) {
		expectedSecret := genericSecret.DeepCopy()
		expectedSecret.Data["TOKEN_3"] = []byte("baz")
		secret := genericSecret.DeepCopy()

		redacted := ""

		applyGenericCredentialsUpdateToSecret(secret, genericCredentials{
			data: map[string]string{
				"TOKEN_1": redacted,
				"TOKEN_2": redacted,
				"TOKEN_3": "baz",
			},
		})

		require.Equal(t, expectedSecret, secret)
	})

	t.Run("edit key in generic secret", func(t *testing.T) {
		expectedSecret := genericSecret.DeepCopy()
		expectedSecret.Data["TOKEN_2"] = []byte("ba")
		secret := genericSecret.DeepCopy()

		redacted := ""

		applyGenericCredentialsUpdateToSecret(secret, genericCredentials{
			data: map[string]string{
				"TOKEN_1": redacted,
				"TOKEN_2": "ba",
			},
		})

		require.Equal(t, expectedSecret, secret)
	})
}
