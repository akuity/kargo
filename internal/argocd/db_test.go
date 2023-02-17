package argocd

import (
	"context"

	argocd "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
)

type fakeDB struct {
	getRepositoryFn func(
		ctx context.Context,
		url string,
	) (*argocd.Repository, error)

	getRepositoryCredentialsFn func(
		ctx context.Context,
		name string,
	) (*argocd.RepoCreds, error)
}

func (f *fakeDB) GetRepository(
	ctx context.Context,
	url string,
) (*argocd.Repository, error) {
	if f.getRepositoryFn == nil {
		return nil, nil
	}
	return f.getRepositoryFn(ctx, url)
}

func (f *fakeDB) GetRepositoryCredentials(
	ctx context.Context,
	name string,
) (*argocd.RepoCreds, error) {
	if f.getRepositoryCredentialsFn == nil {
		return nil, nil
	}
	return f.getRepositoryCredentialsFn(ctx, name)
}
