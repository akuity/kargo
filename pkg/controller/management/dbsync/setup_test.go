package dbsync

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/reconciler"
	"github.com/akuity/kargo/pkg/reconciler/list"
)

// TestControllerResync runs the reconciler on the engine with only the diff
// source, as a resync does, and checks that keys the diff returns are
// resolved against live Kubernetes: a present object is synced and an
// orphaned row's key is deleted.
func TestControllerResync(t *testing.T) {
	t.Parallel()
	scheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(scheme))
	kube := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
		&kargoapi.Project{ObjectMeta: metav1.ObjectMeta{Name: "live", UID: "live-uid"}},
	).Build()
	s := &fakeSyncer{project: true, requests: []reconcile.Request{
		{NamespacedName: client.ObjectKey{Name: "live"}},
		{NamespacedName: client.ObjectKey{Name: "gone"}},
	}}
	c, err := reconciler.New[reconcile.Request]("db-sync-test").
		Watch(list.New(s.Diff)).
		Func(newReconciler(kube, s))
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- c.Start(ctx) }()
	require.Eventually(t, func() bool {
		synced, deleted := s.snapshot()
		return len(synced) == 1 && len(deleted) == 1
	}, 5*time.Second, 10*time.Millisecond)
	cancel()
	select {
	case err = <-done:
		require.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("controller did not stop")
	}
	synced, deleted := s.snapshot()
	require.Equal(t, "live-uid", string(synced[0].GetUID()))
	require.Equal(t, []client.ObjectKey{{Name: "gone"}}, deleted)
}

// TestControllerFailedDiff checks that a diff which fails at startup fails
// the controller, so it never runs against a store it cannot read.
func TestControllerFailedDiff(t *testing.T) {
	t.Parallel()
	s := &fakeSyncer{project: true, diffErr: context.DeadlineExceeded}
	c, err := reconciler.New[reconcile.Request]("db-sync-test").
		Watch(list.New(s.Diff)).
		Func(newReconciler(nil, s))
	require.NoError(t, err)
	require.ErrorIs(t, c.Start(context.Background()), context.DeadlineExceeded)
	synced, deleted := s.snapshot()
	require.Empty(t, synced)
	require.Empty(t, deleted)
}
