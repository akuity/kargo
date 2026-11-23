package stages

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/client/apiserver/freightclient/testutil"
	"github.com/akuity/kargo/pkg/controller/stages/verification"
)

func TestHandleDelete_clearsFreightStatusThroughAPIServer(t *testing.T) {
	const project = "fake-project"
	const stageName = "fake-stage"

	scheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(scheme))

	stage := &kargoapi.Stage{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:  project,
			Name:       stageName,
			Finalizers: []string{kargoapi.FinalizerName},
		},
	}

	// The Stage lives in the Kubernetes client (its finalizer is removed there);
	// Freight status writes go through the API server client.
	c := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(stage).
		Build()

	// Selecting which Freight reference the Stage is the API server's job, so
	// the test asserts only that the Stage is cleared in a single request.
	var clearCalls []clearStageCall
	mockClient := &testutil.MockFreightClient{
		ClearStageFreightsFun: func(_ context.Context, project, stage string) error {
			clearCalls = append(clearCalls, clearStageCall{project, stage})
			return nil
		},
	}

	ver := verification.NewVerifier("", nil, nil, nil, nil, nil)
	require.NoError(t, HandleDelete(
		t.Context(),
		ver,
		c,
		mockClient,
		stage,
	))

	require.Equal(t, []clearStageCall{{project, stageName}}, clearCalls)

	// The finalizer is removed from the Stage.
	gotStage := &kargoapi.Stage{}
	require.NoError(t, c.Get(t.Context(), client.ObjectKeyFromObject(stage), gotStage))
	require.NotContains(t, gotStage.Finalizers, kargoapi.FinalizerName)
}
