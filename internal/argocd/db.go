package argocd

import (
	"context"

	argocd "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
)

// DB is the subset of the db.ArgoDB interface functions that are actually
// used by this package. Dealing with a more limited set of functions makes
// it a bit easier to mock out the DB.
type DB interface {
	GetRepository(ctx context.Context, url string) (*argocd.Repository, error)
	GetRepositoryCredentials(
		ctx context.Context,
		name string,
	) (*argocd.RepoCreds, error)
}
