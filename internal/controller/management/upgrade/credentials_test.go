package upgrade

import (
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	libCreds "github.com/akuity/kargo/internal/credentials"
)

func TestTransformCredentialsSecret(t *testing.T) {
	const testURL = "https://github.com/starkindustries/jarvis.git"
	const testUsername = "tony@starkindustries.com"
	const testPassword = "ilovepepperpotts"

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				SecretTypeLabelKey: repoLabelValue,
			},
		},
		Data: map[string][]byte{
			"type":                 []byte(kargoapi.CredentialTypeLabelValueGit),
			"url":                  []byte(testURL),
			libCreds.FieldUsername: []byte(testUsername),
			libCreds.FieldPassword: []byte(testPassword),
		},
	}

	t.Run("exact url", func(t *testing.T) {
		s := secret.DeepCopy()

		transformCredentialsSecret(s)

		_, ok := s.Labels[SecretTypeLabelKey]
		require.False(t, ok)
		repoType, ok := s.Labels[kargoapi.CredentialTypeLabelKey]
		require.True(t, ok)
		require.Equal(t, kargoapi.CredentialTypeLabelValueGit, repoType)
		require.Equal(t, testURL, s.StringData[libCreds.FieldRepoURL])
		_, ok = s.StringData[libCreds.FieldRepoURLIsRegex]
		require.False(t, ok)
		require.Equal(t, testUsername, s.StringData[libCreds.FieldUsername])
		require.Equal(t, testPassword, s.StringData[libCreds.FieldPassword])
	})

	t.Run("url prefix", func(t *testing.T) {
		s := secret.DeepCopy()
		s.Labels[SecretTypeLabelKey] = repoCredsLabelValue

		transformCredentialsSecret(s)

		_, ok := s.Labels[SecretTypeLabelKey]
		require.False(t, ok)
		repoType, ok := s.Labels[kargoapi.CredentialTypeLabelKey]
		require.True(t, ok)
		require.Equal(t, kargoapi.CredentialTypeLabelValueGit, repoType)
		require.NotEqual(t, testURL, s.StringData[libCreds.FieldRepoURL])
		require.Contains(t, s.StringData[libCreds.FieldRepoURL], testURL)
		require.Equal(t, "true", s.StringData[libCreds.FieldRepoURLIsRegex])
		require.Equal(t, testUsername, s.StringData[libCreds.FieldUsername])
		require.Equal(t, testPassword, s.StringData[libCreds.FieldPassword])
	})
}
