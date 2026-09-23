package event

import (
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

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

	evt := NewAPITokenCreated("API token created", "test-actor", tokenSecret)

	require.Equal(t, kargoapi.EventTypeAPITokenCreated, evt.Type())
	require.Equal(t, "test-project", evt.GetProject())
	require.Equal(t, "test-token", evt.GetName())
	require.Equal(t, "Secret", evt.Kind())
	require.Equal(t, "test-role", evt.RoleName)
	require.Equal(t, "API token created", evt.GetMessage())
	require.NotNil(t, evt.Actor)
	require.Equal(t, "test-actor", *evt.Actor)
}

func TestNewAPITokenCreated_NoActor(t *testing.T) {
	evt := NewAPITokenCreated("API token created", "", &corev1.Secret{})
	require.Nil(t, evt.Actor)
}

func TestAPITokenCreated_AnnotationsRoundTrip(t *testing.T) {
	actor := "test-actor"
	original := &APITokenCreated{
		Common: Common{
			Project: "test-project",
			Actor:   &actor,
			Message: "API token created",
		},
		APIToken: APIToken{
			Name:     "test-token",
			RoleName: "test-role",
		},
	}

	annotations := original.MarshalAnnotations()
	require.Equal(t, map[string]string{
		kargoapi.AnnotationKeyEventProject:      "test-project",
		kargoapi.AnnotationKeyEventActor:        "test-actor",
		kargoapi.AnnotationKeyEventAPITokenName: "test-token",
		kargoapi.AnnotationKeyEventRoleName:     "test-role",
	}, annotations)

	decoded, err := UnmarshalAPITokenCreatedAnnotations("event-id", annotations)
	require.NoError(t, err)
	// The message travels on the Kubernetes Event itself, not in annotations.
	original.Message = ""
	original.ID = "event-id"
	require.Equal(t, original, decoded)
}
