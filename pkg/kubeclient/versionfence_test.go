package kubeclient

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func TestVersionFence_WaitForVersion_AlreadyObserved(t *testing.T) {
	f := NewVersionFence()
	key := types.NamespacedName{Namespace: "ns", Name: "obj"}

	f.OnAdd(&kargoapi.Promotion{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:       "ns",
			Name:            "obj",
			ResourceVersion: "10",
		},
	}, false)

	err := f.WaitForVersion(t.Context(), key, "10", time.Second)
	require.NoError(t, err)
}

func TestVersionFence_WaitForVersion_ObservedLater(t *testing.T) {
	f := NewVersionFence()
	key := types.NamespacedName{Namespace: "ns", Name: "obj"}

	f.OnAdd(&kargoapi.Promotion{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:       "ns",
			Name:            "obj",
			ResourceVersion: "5",
		},
	}, false)

	done := make(chan error, 1)
	go func() {
		done <- f.WaitForVersion(t.Context(), key, "10", 5*time.Second)
	}()

	time.Sleep(10 * time.Millisecond)
	f.OnUpdate(nil, &kargoapi.Promotion{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:       "ns",
			Name:            "obj",
			ResourceVersion: "10",
		},
	})

	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("WaitForVersion did not return in time")
	}
}

func TestVersionFence_WaitForVersion_Timeout(t *testing.T) {
	f := NewVersionFence()
	key := types.NamespacedName{Namespace: "ns", Name: "obj"}

	err := f.WaitForVersion(t.Context(), key, "10", 50*time.Millisecond)
	require.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestVersionFence_WaitForVersion_ContextCancelled(t *testing.T) {
	f := NewVersionFence()
	key := types.NamespacedName{Namespace: "ns", Name: "obj"}

	ctx, cancel := context.WithCancel(t.Context())
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	err := f.WaitForVersion(ctx, key, "10", 5*time.Second)
	require.Error(t, err)
}

func TestVersionFence_WaitForVersion_NonIntegerRV(t *testing.T) {
	f := NewVersionFence()
	key := types.NamespacedName{Namespace: "ns", Name: "obj"}

	err := f.WaitForVersion(t.Context(), key, "not-a-number", time.Second)
	require.NoError(t, err)
	_ = key
}

func TestVersionFence_WaitForVersion_HigherVersionSatisfies(t *testing.T) {
	f := NewVersionFence()
	key := types.NamespacedName{Namespace: "ns", Name: "obj"}

	f.OnAdd(&kargoapi.Promotion{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:       "ns",
			Name:            "obj",
			ResourceVersion: "15",
		},
	}, false)

	err := f.WaitForVersion(t.Context(), key, "10", time.Second)
	require.NoError(t, err)
}

func TestVersionFence_OnDelete_CleansUp(t *testing.T) {
	f := NewVersionFence()

	f.OnAdd(&kargoapi.Promotion{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:       "ns",
			Name:            "obj",
			ResourceVersion: "10",
		},
	}, false)

	f.OnDelete(&kargoapi.Promotion{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:       "ns",
			Name:            "obj",
			ResourceVersion: "10",
		},
	})

	key := types.NamespacedName{Namespace: "ns", Name: "obj"}
	f.mu.Lock()
	_, exists := f.observed[key]
	f.mu.Unlock()
	require.False(t, exists, "expected key to be removed after OnDelete")
}

func TestVersionFence_ConcurrentUpdates(t *testing.T) {
	f := NewVersionFence()
	key := types.NamespacedName{Namespace: "ns", Name: "obj"}

	for i := range 20 {
		rv := i + 1
		go func() {
			f.OnUpdate(nil, &kargoapi.Promotion{
				ObjectMeta: metav1.ObjectMeta{
					Namespace:       "ns",
					Name:            "obj",
					ResourceVersion: fmt.Sprintf("%d", rv),
				},
			})
		}()
	}

	err := f.WaitForVersion(t.Context(), key, "20", 5*time.Second)
	require.NoError(t, err)

	f.mu.Lock()
	require.GreaterOrEqual(t, f.observed[key], int64(20))
	f.mu.Unlock()
}
