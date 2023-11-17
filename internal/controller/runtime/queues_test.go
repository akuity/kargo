package runtime

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func TestPriorityQueue(t *testing.T) {
	_, err := NewPriorityQueue(nil)
	require.Error(t, err)
	require.Equal(
		t,
		"the priority queue was initialized with a nil client.Object "+
			"comparison function",
		err.Error(),
	)

	objects := make([]client.Object, 50)
	for i := range objects {
		objects[i] = &kargoapi.Promotion{
			ObjectMeta: metav1.ObjectMeta{
				// UUIDs contain enough randomness that we know we're not accidentally
				// creating a list that's already ordered by name.
				Name: uuid.New().String(),
			},
		}
	}
	objects[0] = nil // This will be ignored

	pq, err := NewPriorityQueue(
		func(client.Object, client.Object) bool {
			return true // Implementation doesn't matter for this test
		},
		objects...,
	)
	require.NoError(t, err)
	require.Equal(t, 49, pq.Depth()) // make sure the nil object was not added

	objects = objects[1:] // Remove the problematic 0 element

	pq, err = NewPriorityQueue(
		func(lhs client.Object, rhs client.Object) bool {
			// lhs has higher priority than rhs if lexically less than rhs
			return lhs.GetName() < rhs.GetName()
		},
		objects...,
	)
	require.NoError(t, err)

	// Now push a bunch...
	for i := 0; i < 50; i++ {
		added := pq.Push(
			&kargoapi.Promotion{
				ObjectMeta: metav1.ObjectMeta{
					Name: uuid.New().String(),
				},
			},
		)
		require.True(t, added)
	}

	// Now pop until we get a nil
	objects = nil
	for {
		object := pq.Pop()
		if object == nil {
			break
		}
		objects = append(objects, object)
	}

	// Verify objects are ordered lexically by object name
	var lastName string
	for _, object := range objects {
		if lastName != "" {
			require.GreaterOrEqual(t, object.GetName(), lastName)
		}
		lastName = object.GetName()
	}
}

func TestPeek(t *testing.T) {
	objects := []client.Object{
		&kargoapi.Promotion{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "bbb",
				Namespace: "default",
			},
		},
		&kargoapi.Promotion{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "aaa",
				Namespace: "default",
			},
		},
		&kargoapi.Promotion{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "ccc",
				Namespace: "default",
			},
		},
	}
	pq, err := NewPriorityQueue(
		func(lhs client.Object, rhs client.Object) bool {
			return lhs.GetName() < rhs.GetName()
		},
		objects...,
	)
	require.NoError(t, err)
	require.Equal(t, 3, pq.Depth())

	require.Equal(t, "aaa", pq.Peek().GetName())
	require.Equal(t, "aaa", pq.Peek().GetName())
	require.Equal(t, "aaa", pq.Pop().GetName())
	require.Equal(t, 2, pq.Depth())

	require.Equal(t, "bbb", pq.Peek().GetName())
	require.Equal(t, "bbb", pq.Pop().GetName())
	require.Equal(t, 1, pq.Depth())

	require.Equal(t, "ccc", pq.Peek().GetName())
	require.Equal(t, "ccc", pq.Pop().GetName())
	require.Equal(t, 0, pq.Depth())
	require.Nil(t, pq.Pop())
}

// TestDuplicatePush verifies when we push the same object, second one is a no-op
func TestDuplicatePush(t *testing.T) {
	obj1 := &kargoapi.Promotion{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo",
			Namespace: "default",
		},
	}
	obj2 := obj1.DeepCopy()
	pq, err := NewPriorityQueue(
		func(lhs client.Object, rhs client.Object) bool {
			return lhs.GetName() < rhs.GetName()
		},
	)
	require.NoError(t, err)

	require.Equal(t, 0, pq.Depth())
	require.True(t, pq.Push(obj1))
	require.Equal(t, 1, pq.Depth())
	require.False(t, pq.Push(obj2))
	require.Equal(t, 1, pq.Depth())

	require.Equal(t, "foo", pq.Pop().GetName())
	require.Equal(t, 0, pq.Depth())
	require.Nil(t, pq.Pop())
}
