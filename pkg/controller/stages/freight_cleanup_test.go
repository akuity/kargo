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

func TestHandleDelete_clearsFreightStatusThroughUpdater(t *testing.T) {
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

	// The Freight that the API server reports as referencing the Stage. Selecting
	// which Freight reference the Stage is the API server's job, so the mock
	// returns these as-is and the test asserts only that each is cleared.
	referencing := []kargoapi.Freight{
		{ObjectMeta: metav1.ObjectMeta{Namespace: project, Name: "verified"}},
		{ObjectMeta: metav1.ObjectMeta{Namespace: project, Name: "both"}},
		{ObjectMeta: metav1.ObjectMeta{Namespace: project, Name: "approved"}},
	}

	// The Stage lives in the Kubernetes client (its finalizer is removed there);
	// Freight reads and status writes go through the API server client.
	c := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(stage).
		Build()

	var clearCalls []clearCall
	mockClient := &testutil.MockFreightClient{
		ListFreightReferencingStageFun: func(context.Context, string, string) ([]kargoapi.Freight, error) {
			return referencing, nil
		},
		ClearFreightStageStatusFun: func(_ context.Context, project, freight, stage string) error {
			clearCalls = append(clearCalls, clearCall{project, freight, stage})
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

	// Every referencing Freight is cleared of the Stage.
	require.Equal(t, []clearCall{
		{project, "verified", stageName},
		{project, "both", stageName},
		{project, "approved", stageName},
	}, clearCalls)

	// The finalizer is removed from the Stage.
	gotStage := &kargoapi.Stage{}
	require.NoError(t, c.Get(t.Context(), client.ObjectKeyFromObject(stage), gotStage))
	require.NotContains(t, gotStage.Finalizers, kargoapi.FinalizerName)
}

// clearCall records the arguments of a single ClearFreightStageStatus call.
type clearCall struct {
	project string
	freight string
	stage   string
}
