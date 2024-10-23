package event

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/logging"
)

func Test_newRecorder(t *testing.T) {
	ctx := context.TODO()
	client := fake.NewClientBuilder().Build()
	logger := logging.LoggerFromContext(ctx)
	r := newRecorder(ctx, client, logger)

	require.NotNil(t, r.backoff)
	require.NotNil(t, r.sink)
	require.NotNil(t, r.logger)
	require.NotNil(t, r.newEventHandlerFn)
}

func Test_retryDecider(t *testing.T) {
	eventGR := schema.GroupResource{
		Group:    corev1.GroupName,
		Resource: "Event",
	}
	testCases := map[string]struct {
		input       error
		shouldRetry bool
	}{
		"event already exists": {
			input:       apierrors.NewAlreadyExists(eventGR, "fake-event"),
			shouldRetry: false,
		},
		"namespace is terminating": {
			input: &apierrors.StatusError{
				ErrStatus: metav1.Status{
					Code: http.StatusForbidden,
					Details: &metav1.StatusDetails{
						Causes: []metav1.StatusCause{
							{
								Type: corev1.NamespaceTerminatingCause,
							},
						},
					},
				},
			},
			shouldRetry: false,
		},
		"unknown error": {
			input:       errors.New("unknown error"),
			shouldRetry: true,
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			r := &recorder{
				logger: logging.Wrap(logr.Discard()),
			}
			require.Equal(t, tc.shouldRetry, r.newRetryDecider(&corev1.Event{})(tc.input))
		})
	}
}

func Test_newSink(t *testing.T) {
	s := newSink(
		context.TODO(),
		fake.NewClientBuilder().Build(),
	)
	require.NotNil(t, s.client)
	require.NotNil(t, s.ctx)
}

func TestNewPromotionEventAnnotations(t *testing.T) {
	testCases := map[string]struct {
		actor     string
		promotion *v1alpha1.Promotion
		freight   *v1alpha1.Freight
		expected  map[string]string
	}{
		"promotion with freight and argocd apps": {
			actor: "test-user",
			promotion: &v1alpha1.Promotion{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-promotion",
					Namespace: "test-namespace",
					CreationTimestamp: metav1.Time{
						Time: time.Date(2024, 10, 22, 0, 0, 0, 0, time.UTC),
					},
					Annotations: map[string]string{
						v1alpha1.AnnotationKeyCreateActor: "promotion-creator",
					},
				},
				Spec: v1alpha1.PromotionSpec{
					Freight: "test-freight",
					Stage:   "test-stage",
					Steps: []v1alpha1.PromotionStep{
						{
							Uses: "argocd-update",
							Config: &v1.JSON{Raw: []byte(`{
  "apps": [
    {
      "name": "test-app-1"
    },
    {
      "name": "test-app-2",
      "namespace": "test-namespace"
    }
  ]
}`)},
						},
					},
				},
			},
			freight: &v1alpha1.Freight{
				ObjectMeta: metav1.ObjectMeta{
					CreationTimestamp: metav1.Time{
						Time: time.Date(2024, 10, 22, 0, 0, 0, 0, time.UTC),
					},
				},
				Alias:   "test-alias",
				Commits: []v1alpha1.GitCommit{{Tag: "test-tag"}},
				Images:  []v1alpha1.Image{{Tag: "test-tag"}},
				Charts:  []v1alpha1.Chart{{Name: "test-chart"}},
			},
			expected: map[string]string{
				v1alpha1.AnnotationKeyEventProject:             "test-namespace",
				v1alpha1.AnnotationKeyEventPromotionName:       "test-promotion",
				v1alpha1.AnnotationKeyEventFreightName:         "test-freight",
				v1alpha1.AnnotationKeyEventStageName:           "test-stage",
				v1alpha1.AnnotationKeyEventPromotionCreateTime: "2024-10-22T00:00:00Z",
				v1alpha1.AnnotationKeyEventActor:               "promotion-creator",
				v1alpha1.AnnotationKeyEventFreightCreateTime:   "2024-10-22T00:00:00Z",
				v1alpha1.AnnotationKeyEventFreightAlias:        "test-alias",
				v1alpha1.AnnotationKeyEventFreightCommits:      `[{"tag":"test-tag"}]`,
				v1alpha1.AnnotationKeyEventFreightImages:       `[{"tag":"test-tag"}]`,
				v1alpha1.AnnotationKeyEventFreightCharts:       `[{"name":"test-chart"}]`,
				v1alpha1.AnnotationKeyEventApplications: `[{"name":"test-app-1","namespace":"argocd"},` +
					`{"name":"test-app-2","namespace":"test-namespace"}]`,
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			result := NewPromotionEventAnnotations(context.TODO(), tc.actor, tc.promotion, tc.freight)
			require.Equal(t, len(tc.expected), len(result), "Number of annotations doesn't match")
			for key, expectedValue := range tc.expected {
				if key == v1alpha1.AnnotationKeyEventApplications {
					expectedAppsJSON := tc.expected[v1alpha1.AnnotationKeyEventApplications]
					actualAppsJSON := result[v1alpha1.AnnotationKeyEventApplications]
					var expectedApps, actualApps []types.NamespacedName
					err := json.Unmarshal([]byte(expectedAppsJSON), &expectedApps)
					require.NoError(t, err, "Failed to unmarshal expected applications")
					err = json.Unmarshal([]byte(actualAppsJSON), &actualApps)
					require.NoError(t, err, "Failed to unmarshal actual applications")
					require.Equal(t, expectedApps, actualApps, "Applications mismatch")
					continue
				}
				actualValue, exists := result[key]
				require.True(t, exists, "Expected annotation %s not found", key)
				require.Equal(t, expectedValue, actualValue, "Annotation %s value mismatch", key)
			}
		})
	}
}
