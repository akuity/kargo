package controller

import (
	"context"

	"github.com/akuityio/kargo/internal/credentials"
)

// fakeCredentialsDB is a mock implementation of the credentials.Database
// interface that is used to facilitate unit testing.
type fakeCredentialsDB struct {
	getFn func(
		ctx context.Context,
		namespace string,
		credType credentials.Type,
		repo string,
	) (credentials.Credentials, bool, error)
}

func (f *fakeCredentialsDB) Get(
	ctx context.Context,
	namespace string,
	credType credentials.Type,
	repo string,
) (credentials.Credentials, bool, error) {
	if f.getFn == nil {
		return credentials.Credentials{}, false, nil
	}
	return f.getFn(ctx, namespace, credType, repo)
}
