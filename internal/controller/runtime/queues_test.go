package runtime

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/technosophos/moniker"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	api "github.com/akuity/kargo/api/v1alpha1"
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

	randomNamer := moniker.New()
	objects := make([]client.Object, 100)
	for i := range objects {
		objects[i] = &api.Promotion{
			ObjectMeta: metav1.ObjectMeta{
				Name: randomNamer.NameSep("-"),
			},
		}
	}
	objects[0] = nil // This should be invalid

	fmt.Println("adding a line to push the offending line down")

	_, err = NewPriorityQueue(
		func(client.Object, client.Object) bool {
			return true // Implementation doesn't matter for this test
		},
		objects...,
	)
	require.Error(t, err)
	require.Equal(
		t,
		"the priority queue was initialized with at least one nil client.Object "+
			"at position 0",
		err.Error(),
	)

	objects = objects[1:] // Remove the problematic 0 element

	pq, err := NewPriorityQueue(
		func(lhs client.Object, rhs client.Object) bool {
			// lhs has higher priority than rhs if lexically less than rhs
			return lhs.GetName() < rhs.GetName()
		},
		objects...,
	)
	require.NoError(t, err)

	// Now push a bunch...
	for i := 0; i < 200; i++ {
		err = pq.Push(
			&api.Promotion{
				ObjectMeta: metav1.ObjectMeta{
					Name: randomNamer.NameSep("-"),
				},
			},
		)
		require.NoError(t, err)
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
