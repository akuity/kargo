package api

import (
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	libCreds "github.com/akuity/kargo/internal/credentials"
)

func TestApplyCredentialsUpdateToSecret(t *testing.T) {
	baseSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				kargoapi.CredentialTypeLabelKey: kargoapi.CredentialTypeLabelValueGit,
			},
		},
		Data: map[string][]byte{
			libCreds.FieldRepoURL:  []byte("fake-url"),
			libCreds.FieldUsername: []byte("fake-username"),
			libCreds.FieldPassword: []byte("fake-password"),
		},
	}

	t.Run("update repoURL", func(t *testing.T) {
		expectedSecret := baseSecret.DeepCopy()
		expectedSecret.Data[libCreds.FieldRepoURL] = []byte("new-fake-url")
		secret := baseSecret.DeepCopy()
		applyCredentialsUpdateToSecret(
			secret,
			credentialsUpdate{
				repoURL: "new-fake-url",
			},
		)
		require.Equal(t, expectedSecret, secret)
	})

	t.Run("update repoURL with pattern", func(t *testing.T) {
		expectedSecret := baseSecret.DeepCopy()
		expectedSecret.Data[libCreds.FieldRepoURL] = []byte("new-fake-url")
		expectedSecret.Data[libCreds.FieldRepoURLIsRegex] = []byte("true")
		secret := baseSecret.DeepCopy()
		applyCredentialsUpdateToSecret(
			secret,
			credentialsUpdate{
				repoURL:        "new-fake-url",
				repoURLISRegex: true,
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

	t.Run("update description", func(t *testing.T) {
		expectedSecret := baseSecret.DeepCopy()
		expectedSecret.Annotations = map[string]string{
			kargoapi.AnnotationKeyDescription: "new description",
		}
		secret := baseSecret.DeepCopy()
		applyCredentialsUpdateToSecret(
			secret,
			credentialsUpdate{
				description: "new description",
			},
		)
		require.Equal(t, expectedSecret, secret)
	})
}
