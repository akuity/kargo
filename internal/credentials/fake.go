package credentials

import "context"

// FakeDB is a mock implementation of the Database interface that is used to
// facilitate unit testing.
type FakeDB struct {
	GetFn func(
		ctx context.Context,
		namespace string,
		credType Type,
		repo string,
	) (Credentials, bool, error)
}

func (f *FakeDB) Get(
	ctx context.Context,
	namespace string,
	credType Type,
	repo string,
) (Credentials, bool, error) {
	if f.GetFn == nil {
		return Credentials{}, false, nil
	}
	return f.GetFn(ctx, namespace, credType, repo)
}
