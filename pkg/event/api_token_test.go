package event

import (
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func TestNewAPITokenCreated(t *testing.T) {
	tokenSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test-project",
			Name:      "test-token",
			Annotations: map[string]string{
				corev1.ServiceAccountNameKey: "test-role",
			},
		},
	}
	testCases := []struct {
		name        string
		actor       string
		tokenSecret *corev1.Secret
		systemLevel bool
		expected    *APITokenCreated
	}{
		{
			name:        "project-level token",
			actor:       "test-actor",
			tokenSecret: tokenSecret,
			expected: &APITokenCreated{
				Common: Common{
					Project: "test-project",
					Actor:   ptr.To("test-actor"),
					Message: "API token created",
				},
				APIToken: APIToken{
					Name:     "test-token",
					RoleName: "test-role",
				},
			},
		},
		{
			name:        "system-level token without actor",
			tokenSecret: tokenSecret,
			systemLevel: true,
			expected: &APITokenCreated{
				Common: Common{
					Project: "test-project",
					Message: "API token created",
				},
				APIToken: APIToken{
					Name:        "test-token",
					RoleName:    "test-role",
					SystemLevel: true,
				},
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			evt, err := NewAPITokenCreated(
				"API token created",
				testCase.actor,
				testCase.tokenSecret,
				testCase.systemLevel,
			)
			require.NoError(t, err)
			requireCloudEvent(
				t,
				evt,
				kargoapi.EventTypeAPITokenCreated,
				"Secret",
			)
			require.Equal(t, testCase.expected, dataAs[APITokenCreated](t, evt))
		})
	}
}

func TestNewAPITokenDeleted(t *testing.T) {
	tokenSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test-project",
			Name:      "test-token",
			Annotations: map[string]string{
				corev1.ServiceAccountNameKey: "test-role",
			},
		},
	}

	evt, err := NewAPITokenDeleted("API token deleted", "test-actor", tokenSecret, true)
	require.NoError(t, err)

	requireCloudEvent(
		t,
		evt,
		kargoapi.EventTypeAPITokenDeleted,
		"Secret",
	)
	require.Equal(
		t,
		&APITokenDeleted{
			Common: Common{
				Project: "test-project",
				Actor:   ptr.To("test-actor"),
				Message: "API token deleted",
			},
			APIToken: APIToken{
				Name:        "test-token",
				RoleName:    "test-role",
				SystemLevel: true,
			},
		},
		dataAs[APITokenDeleted](t, evt),
	)
}
