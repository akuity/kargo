package credentials

import "context"

// Database is an interface for a Credentials store.
type Database interface {
	Get(
		ctx context.Context,
		namespace string,
		credType Type,
		repo string,
	) (*Credentials, error)
}

// FakeDB is a mock implementation of the Database interface that is used to
// facilitate unit testing.
type FakeDB struct {
	GetFn func(
		ctx context.Context,
		namespace string,
		credType Type,
		repo string,
	) (*Credentials, error)
}

func (f *FakeDB) Get(
	ctx context.Context,
	namespace string,
	credType Type,
	repo string,
) (*Credentials, error) {
	if f.GetFn == nil {
		return nil, nil
	}
	return f.GetFn(ctx, namespace, credType, repo)
}
