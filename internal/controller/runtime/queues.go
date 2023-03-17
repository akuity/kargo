package runtime

import (
	"container/heap"
	"sync"

	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// PriorityQueue is an interface for any priority queue containing
// runtime.Objects.
type PriorityQueue interface {
	// Push adds a copy of the provided client.Object to the priority queue.
	// Implementations MUST disallow pushing nil objects.
	Push(client.Object) error
	// Pop removes the highest priority client.Object from the priority queue and
	// returns it. Implementations MUST return nil if the priority queue is empty.
	Pop() client.Object
	// Depth returns the depth of the PriorityQueue.
	Depth() int
}

// ObjectCompareFn is the signature for any function that can compare the
// relative priorities of two client.Objects. Implementations MUST return true
// when the first argument is of higher priority than the second and MUST return
// false otherwise. Implementors of such functions may safely assume that
// neither argument can ever be nil.
type ObjectPriorityFn func(client.Object, client.Object) bool

// priorityQueue is an implementation of the PriorityQueue interface. This
// encapsulates the low-level details of the priority queue implementation found
// at https://pkg.go.dev/container/heap and is also safe for concurrent use by
// multiple goroutines.
type priorityQueue struct {
	// internalQueue is priorityQueue's underlying data structure. It implements
	// heap.Interface.
	internalQueue *internalPriorityQueue
	// mu is a mutex used to ensure only a single goroutine is executing critical
	// sections of code at any time.
	mu sync.Mutex
}

// NewPriorityQueue takes a function for comparing the relative priority of two
// client.Objects (which MUST return true when the first argument is of higher
// priority than the second and MUST return false otherwise) and, optionally,
// any number of client.Objects (which do NOT needs to be pre-ordered) and
// returns an implementation of the PriorityQueue interface that is safe for
// concurrent use by multiple goroutines. This function will also return an
// error if initialized with a nil comparison function or any nil
// runtime.Objects.
func NewPriorityQueue(
	higherFn ObjectPriorityFn,
	objects ...client.Object,
) (PriorityQueue, error) {
	if higherFn == nil {
		return nil, errors.New(
			"the priority queue was initialized with a nil client.Object " +
				"comparison function",
		)
	}
	if objects == nil {
		objects = []client.Object{}
	}
	for i, object := range objects {
		if object == nil {
			return nil, errors.Errorf(
				"the priority queue was initialized with at least one nil "+
					"client.Object at position %d",
				i,
			)
		}
	}
	internalQueue := &internalPriorityQueue{
		objects:  objects,
		higherFn: higherFn,
	}
	heap.Init(internalQueue)
	return &priorityQueue{
		internalQueue: internalQueue,
	}, nil
}

func (p *priorityQueue) Push(item client.Object) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if item == nil {
		return errors.New("a nil client.Object was pushed onto the priority queue")
	}
	heap.Push(p.internalQueue, item.DeepCopyObject())
	return nil
}

func (p *priorityQueue) Pop() client.Object {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.internalQueue.Len() == 0 {
		return nil
	}
	return heap.Pop(p.internalQueue).(client.Object)
}

func (p *priorityQueue) Depth() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.internalQueue.Len()
}

// internalPriorityQueue is the underlying data structure for priorityQueue. It
// implements heap.Interface, which allows priorityQueue to offload ordering of
// its client.Objects by priority to the heap package.
type internalPriorityQueue struct {
	objects  []client.Object
	higherFn ObjectPriorityFn
}

func (i *internalPriorityQueue) Len() int { return len(i.objects) }

func (i *internalPriorityQueue) Less(n, m int) bool {
	return i.higherFn(i.objects[n], i.objects[m])
}

func (i *internalPriorityQueue) Swap(n, m int) {
	i.objects[n], i.objects[m] = i.objects[m], i.objects[n]
}

func (i *internalPriorityQueue) Push(item any) {
	i.objects = append(i.objects, item.(client.Object))
}

func (i *internalPriorityQueue) Pop() any {
	n := len(i.objects)
	item := i.objects[n-1]
	i.objects[n-1] = nil // avoid memory leak
	i.objects = i.objects[:n-1]
	return item
}
