//go:build integration && db

package projectconfigs

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/database"
	"github.com/akuity/kargo/pkg/database/databasetest"
	"github.com/akuity/kargo/pkg/database/projectconfigstore"
)

// discardPublisher is a nats.Publisher that drops what is published.
type discardPublisher struct{}

func (discardPublisher) Publish(string, []byte) error { return nil }

// cancelAfter is a configImporter that cancels the import's context once n
// ProjectConfigs have been imported, as if the replica had been killed.
type cancelAfter struct {
	store  configImporter
	n      int
	cancel context.CancelFunc
	mu     sync.Mutex
	count  int
}

func (c *cancelAfter) Import(
	ctx context.Context,
	cfg *kargoapi.ProjectConfig,
	projectUID string,
) (bool, error) {
	inserted, err := c.store.Import(ctx, cfg, projectUID)
	c.mu.Lock()
	defer c.mu.Unlock()
	c.count++
	if c.count == c.n {
		c.cancel()
	}
	return inserted, err
}

func TestImporterIntegration(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	// newCluster returns a cluster holding the policy binding and, for each
	// of n Projects, the Project and its ProjectConfig resource with one
	// webhook receiver.
	newCluster := func(t *testing.T, n int) client.Reader {
		t.Helper()
		objects := []client.Object{testBinding()}
		for i := range n {
			project := fmt.Sprintf("project-%02d", i)
			objects = append(
				objects,
				testProjectObj(project),
				testProjectConfigCR(t, project, "/"+project),
			)
		}
		return fake.NewClientBuilder().
			WithScheme(importerTestScheme(t)).
			WithRESTMapper(projectConfigMapper()).
			WithObjects(objects...).
			Build()
	}

	t.Run("replicas in parallel import once", func(t *testing.T) {
		t.Parallel()
		pool, _ := databasetest.IsolatedDatabase(t)
		kube := newCluster(t, 5)
		var mu sync.Mutex
		var calls []string
		recording := func(store projectconfigstore.Store) configImporter {
			return importFunc(func(
				ctx context.Context,
				cfg *kargoapi.ProjectConfig,
				uid string,
			) (bool, error) {
				mu.Lock()
				calls = append(calls, cfg.Name)
				mu.Unlock()
				return store.Import(ctx, cfg, uid)
			})
		}
		store := projectconfigstore.New(pool, discardPublisher{})
		var wg sync.WaitGroup
		errs := make([]error, 2)
		for r := range errs {
			wg.Go(func() {
				importer := newImporter(
					&pgLedger{pool: pool},
					recording(store),
					kube,
					projectConfigMapper(),
				)
				importer.pollInterval = time.Millisecond
				errs[r] = importer.Run(ctx)
			})
		}
		wg.Wait()
		require.NoError(t, errs[0])
		require.NoError(t, errs[1])
		// Only the replica that took the lock first imported anything.
		require.Len(t, calls, 5)
		configs, err := store.List(ctx)
		require.NoError(t, err)
		require.Len(t, configs, 5)
		_, err = database.New(pool).GetDataImport(ctx, importName)
		require.NoError(t, err)
	})

	t.Run("an interrupted import finishes when run again", func(t *testing.T) {
		t.Parallel()
		pool, _ := databasetest.IsolatedDatabase(t)
		kube := newCluster(t, 6)
		store := projectconfigstore.New(pool, discardPublisher{})

		killCtx, cancel := context.WithCancel(ctx)
		defer cancel()
		killed := newImporter(
			&pgLedger{pool: pool},
			&cancelAfter{store: store, n: 3, cancel: cancel},
			kube,
			projectConfigMapper(),
		)
		require.Error(t, killed.Run(killCtx))
		done, err := (&pgLedger{pool: pool}).completed(ctx)
		require.NoError(t, err)
		require.False(t, done)
		configs, err := store.List(ctx)
		require.NoError(t, err)
		require.Len(t, configs, 3)

		// The killed run released the lock, so the rerun does not block.
		rerunCtx, rerunCancel := context.WithTimeout(ctx, 30*time.Second)
		defer rerunCancel()
		require.NoError(t, NewImporter(pool, store, kube, projectConfigMapper()).Run(rerunCtx))
		configs, err = store.List(ctx)
		require.NoError(t, err)
		require.Len(t, configs, 6)
		var receivers int
		require.NoError(t, pool.QueryRow(
			ctx,
			"SELECT count(*) FROM project_config_webhook_receivers",
		).Scan(&receivers))
		require.Equal(t, 6, receivers)
		done, err = (&pgLedger{pool: pool}).completed(ctx)
		require.NoError(t, err)
		require.True(t, done)
	})

	t.Run("copies receivers and conditions, not the observed generation", func(t *testing.T) {
		t.Parallel()
		pool, _ := databasetest.IsolatedDatabase(t)
		kube := newCluster(t, 1)
		store := projectconfigstore.New(pool, discardPublisher{})
		require.NoError(t, NewImporter(pool, store, kube, projectConfigMapper()).Run(ctx))

		got, err := store.Get(ctx, "project-00", "project-00-uid")
		require.NoError(t, err)
		require.Nil(t, got.StatusObservedGeneration)
		require.Len(t, got.Receivers, 1)
		require.Equal(t, "project-00-a", got.Receivers[0].Name)
		require.Equal(t, "/project-00", got.Receivers[0].Path)
		require.Equal(t, "https://webhooks.example.com/project-00", got.Receivers[0].URL)
		var conditions []metav1.Condition
		require.NoError(t, json.Unmarshal(got.StatusConditions, &conditions))
		require.Len(t, conditions, 1)
		require.Equal(t, kargoapi.ConditionTypeReady, conditions[0].Type)
		require.Equal(t, "Synced", conditions[0].Reason)
		byPath, err := store.GetByWebhookReceiverPath(ctx, "/project-00")
		require.NoError(t, err)
		require.Equal(t, "project-00", byPath.ProjectName)
	})

	t.Run("a recorded import never runs again", func(t *testing.T) {
		t.Parallel()
		pool, _ := databasetest.IsolatedDatabase(t)
		kube := newCluster(t, 1)
		store := projectconfigstore.New(pool, discardPublisher{})
		require.NoError(t, NewImporter(pool, store, kube, projectConfigMapper()).Run(ctx))
		require.NoError(t, store.Delete(ctx, "project-00", "project-00-uid"))
		require.NoError(t, NewImporter(pool, store, kube, projectConfigMapper()).Run(ctx))
		configs, err := store.List(ctx)
		require.NoError(t, err)
		require.Empty(t, configs)
	})
}

// importFunc adapts a function to configImporter.
type importFunc func(context.Context, *kargoapi.ProjectConfig, string) (bool, error)

func (f importFunc) Import(
	ctx context.Context,
	cfg *kargoapi.ProjectConfig,
	projectUID string,
) (bool, error) {
	return f(ctx, cfg, projectUID)
}
