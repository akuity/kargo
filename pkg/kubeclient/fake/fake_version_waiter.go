package fake

import (
	"context"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/types"
)

// VersionWait records a single call to WaitForVersion.
type VersionWait struct {
	Key             types.NamespacedName
	ResourceVersion string
	Timeout         time.Duration
}

// VersionWaiter is a test double for kubeclient.VersionWaiter that records
// calls without blocking. Use Waits() to inspect what was awaited.
type VersionWaiter struct {
	mu    sync.Mutex
	waits []VersionWait
}

// WaitForVersion records the call and returns nil immediately.
func (f *VersionWaiter) WaitForVersion(
	_ context.Context,
	key types.NamespacedName,
	rv string,
	timeout time.Duration,
) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.waits = append(f.waits, VersionWait{
		Key:             key,
		ResourceVersion: rv,
		Timeout:         timeout,
	})
	return nil
}

// Waits returns a copy of all recorded WaitForVersion calls.
func (f *VersionWaiter) Waits() []VersionWait {
	f.mu.Lock()
	defer f.mu.Unlock()
	result := make([]VersionWait, len(f.waits))
	copy(result, f.waits)
	return result
}
