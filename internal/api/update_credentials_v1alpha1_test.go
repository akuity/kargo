package api

import (
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func TestApplyCredentialsUpdateToSecret(t *testing.T) {
	baseSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				kargoapi.CredentialTypeLabelKey: kargoapi.CredentialTypeLabelValueGit,
			},
		},
		Data: map[string][]byte{
			"repoURL":        []byte("fake-url"),
			"repoURLPattern": []byte("fake-pattern"),
			"username":       []byte("fake-username"),
			"password":       []byte("fake-password"),
		},
	}

	t.Run("update repoURL", func(t *testing.T) {
		expectedSecret := baseSecret.DeepCopy()
		expectedSecret.Data["repoURL"] = []byte("new-fake-url")
		delete(expectedSecret.Data, "repoURLPattern")
		secret := baseSecret.DeepCopy()
		applyCredentialsUpdateToSecret(
			secret,
			credentialsUpdate{
				repoURL: "new-fake-url",
			},
		)
		require.Equal(t, expectedSecret, secret)
	})

	t.Run("update repoURLPattern", func(t *testing.T) {
		expectedSecret := baseSecret.DeepCopy()
		expectedSecret.Data["repoURLPattern"] = []byte("new-fake-pattern")
		delete(expectedSecret.Data, "repoURL")
		secret := baseSecret.DeepCopy()
		applyCredentialsUpdateToSecret(
			secret,
			credentialsUpdate{
				repoURLPattern: "new-fake-pattern",
			},
		)
		require.Equal(t, expectedSecret, secret)
	})

	t.Run("update username", func(t *testing.T) {
		expectedSecret := baseSecret.DeepCopy()
		expectedSecret.Data["username"] = []byte("new-fake-username")
		secret := baseSecret.DeepCopy()
		applyCredentialsUpdateToSecret(
			secret,
			credentialsUpdate{
				username: "new-fake-username",
			},
		)
		require.Equal(t, expectedSecret, secret)
	})

	t.Run("update password", func(t *testing.T) {
		expectedSecret := baseSecret.DeepCopy()
		expectedSecret.Data["password"] = []byte("new-fake-password")
		secret := baseSecret.DeepCopy()
		applyCredentialsUpdateToSecret(
			secret,
			credentialsUpdate{
				password: "new-fake-password",
			},
		)
		require.Equal(t, expectedSecret, secret)
	})
}
