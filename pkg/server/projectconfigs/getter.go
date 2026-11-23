package projectconfigs

import (
	"context"
	"errors"
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/api"
	"github.com/akuity/kargo/pkg/database"
	"github.com/akuity/kargo/pkg/database/projectconfigstore"
)

type storeGetter struct {
	store projectconfigstore.Store
	kube  client.Reader
}

// StoreGetter serves api.ProjectConfigGetter inside the API server: it reads
// the Project (for its UID) with kube and the config from the store. A
// Project that does not exist, or whose ProjectConfig does not exist (a row
// left by a deleted Project of the same name counts as absent), yields nil.
func StoreGetter(
	store projectconfigstore.Store,
	kube client.Reader,
) api.ProjectConfigGetter {
	return &storeGetter{store: store, kube: kube}
}

func (g *storeGetter) Get(
	ctx context.Context,
	project string,
) (*kargoapi.ProjectConfig, error) {
	p := &kargoapi.Project{}
	if err := g.kube.Get(ctx, client.ObjectKey{Name: project}, p); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("error getting Project %q: %w", project, err)
	}
	cfg, err := g.store.Get(ctx, project, string(p.UID))
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("error getting ProjectConfig %q: %w", project, err)
	}
	return projectconfigstore.ToAPI(cfg), nil
}
