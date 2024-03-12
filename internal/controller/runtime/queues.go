package runtime

import (
	"container/heap"
	"errors"
	"sync"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// PriorityQueue is an interface for any priority queue containing
// runtime.Objects.
type PriorityQueue interface {
	// Push adds a copy of the provided client.Object to the priority queue.
	// Returns true if the item was added to the queue, false if it already existed
	// Pushing of nil objects have no effect
	Push(client.Object) bool
	// Pop removes the highest priority client.Object from the priority queue and
	// returns it. Implementations MUST return nil if the priority queue is empty.
	Pop() client.Object
	// Peek returns the highest priority client.Object from the priority queue
	// without removing it.
	Peek() client.Object
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

	// objectsByNamespaceName is used to deduplicate pushes of the same object
	// to the queue, allowing Push() to be idempotent
	objectsByNamespaceName map[types.NamespacedName]bool

	// mu is a mutex used to ensure only a single goroutine is executing critical
	// sections of code at any time.
	mu sync.RWMutex
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
	filteredObjs := []client.Object{}
	objectsByNamespaceName := make(map[types.NamespacedName]bool)
	// filter out duplicates and nils
	for i, object := range objects {
		if object == nil {
			continue
		}
		key := types.NamespacedName{
			Namespace: object.GetNamespace(),
			Name:      object.GetName(),
		}
		if objectsByNamespaceName[key] {
			continue
		}
		objectsByNamespaceName[key] = true
		filteredObjs = append(filteredObjs, objects[i])
	}
	internalQueue := &internalPriorityQueue{
		objects:  filteredObjs,
		higherFn: higherFn,
	}
	heap.Init(internalQueue)
	return &priorityQueue{
		objectsByNamespaceName: objectsByNamespaceName,
		internalQueue:          internalQueue,
	}, nil
}

func (p *priorityQueue) Push(item client.Object) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	if item == nil {
		return false
	}
	key := types.NamespacedName{
		Namespace: item.GetNamespace(),
		Name:      item.GetName(),
	}
	if p.objectsByNamespaceName[key] {
		return false
	}
	heap.Push(p.internalQueue, item.DeepCopyObject())
	p.objectsByNamespaceName[key] = true
	return true
}

func (p *priorityQueue) Pop() client.Object {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.internalQueue.Len() == 0 {
		return nil
	}
	obj := heap.Pop(p.internalQueue).(client.Object) // nolint: forcetypeassert
	key := types.NamespacedName{
		Namespace: obj.GetNamespace(),
		Name:      obj.GetName(),
	}
	delete(p.objectsByNamespaceName, key)
	return obj
}

func (p *priorityQueue) Peek() client.Object {
	p.mu.RLock()
	defer p.mu.RUnlock()
	n := len(p.internalQueue.objects)
	if n == 0 {
		return nil
	}
	return p.internalQueue.objects[0]
}

func (p *priorityQueue) Depth() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
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
	i.objects = append(i.objects, item.(client.Object)) // nolint: forcetypeassert
}

func (i *internalPriorityQueue) Pop() any {
	n := len(i.objects)
	item := i.objects[n-1]
	i.objects[n-1] = nil // avoid memory leak
	i.objects = i.objects[:n-1]
	return item
}
