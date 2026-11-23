package projectconfigs

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/database"
	"github.com/akuity/kargo/pkg/database/projectconfigstore"
	"github.com/akuity/kargo/pkg/logging"
)

const (
	// importName is the name the import of ProjectConfig resources is
	// recorded under once it has completed.
	importName = "projectconfigs"
	// advisoryLockKey is the PostgreSQL advisory lock the import is run
	// under, so that one replica imports while the others wait. It is ASCII
	// "kargopc1"; any fixed value works, but it must never change, or
	// replicas of two releases could import at once.
	advisoryLockKey int64 = 0x6b6172676f706331
	// policyBindingName is the ValidatingAdmissionPolicyBinding, installed
	// by the chart, that makes ProjectConfig resources read-only. The import
	// waits for it, so no write to a resource after the import is lost.
	policyBindingName = "kargo-projectconfigs-read-only"
	// policyBindingPollInterval is how often the import checks for the
	// policy binding while it waits for it.
	policyBindingPollInterval = 2 * time.Second
	// unlockTimeout bounds the release of the advisory lock, which runs
	// even after the import's context has ended.
	unlockTimeout = 10 * time.Second
)

// projectConfigGVK is the kind of the resources imported. They are read as
// unstructured objects, so the kind need not be registered in any scheme.
var projectConfigGVK = kargoapi.GroupVersion.WithKind("ProjectConfig")

// importLedger records that the import has completed and serializes the
// replicas that would run it.
type importLedger interface {
	// completed reports whether the import has been recorded.
	completed(ctx context.Context) (bool, error)
	// lock blocks until the caller holds the import lock and returns a
	// function that releases it.
	lock(ctx context.Context) (release func(), err error)
	// record records that the import has completed.
	record(ctx context.Context) error
}

// configImporter stores a ProjectConfig read from its resource unless its
// Project already has one. projectconfigstore.Store is one.
type configImporter interface {
	Import(ctx context.Context, cfg *kargoapi.ProjectConfig, projectUID string) (bool, error)
}

// Importer copies the ProjectConfig resources of a Kargo installation that
// kept them in Kubernetes into the database, once.
type Importer struct {
	ledger       importLedger
	store        configImporter
	kube         client.Reader
	mapper       meta.RESTMapper
	pollInterval time.Duration
}

// NewImporter returns an Importer that records its completion in, and locks
// through, pool; stores ProjectConfigs through store; and reads resources
// through kube, which must not be served from a cache, and mapper.
func NewImporter(
	pool *pgxpool.Pool,
	store projectconfigstore.Store,
	kube client.Reader,
	mapper meta.RESTMapper,
) *Importer {
	return newImporter(&pgLedger{pool: pool}, store, kube, mapper)
}

func newImporter(
	ledger importLedger,
	store configImporter,
	kube client.Reader,
	mapper meta.RESTMapper,
) *Importer {
	return &Importer{
		ledger:       ledger,
		store:        store,
		kube:         kube,
		mapper:       mapper,
		pollInterval: policyBindingPollInterval,
	}
}

// Run blocks until the import is recorded, by this replica or another, and
// returns nil; or returns why it is not. The import is recorded only once
// every ProjectConfig resource that can be imported has been, or when there
// is no ProjectConfig kind to import from. It never overwrites a
// ProjectConfig in the database, so an import interrupted part way can
// safely be run again.
func (i *Importer) Run(ctx context.Context) error {
	logger := logging.LoggerFromContext(ctx).WithValues("import", importName)
	done, err := i.ledger.completed(ctx)
	if err != nil {
		return fmt.Errorf("error checking for an earlier ProjectConfig import: %w", err)
	}
	if done {
		logger.Debug("ProjectConfig resources were imported already")
		return nil
	}
	logger.Info("waiting for the ProjectConfig import lock")
	release, err := i.ledger.lock(ctx)
	if err != nil {
		return fmt.Errorf("error acquiring the ProjectConfig import lock: %w", err)
	}
	defer release()
	// Another replica may have imported while this one waited for the lock.
	if done, err = i.ledger.completed(ctx); err != nil {
		return fmt.Errorf("error checking for an earlier ProjectConfig import: %w", err)
	}
	if done {
		logger.Info("ProjectConfig resources were imported by another replica")
		return nil
	}
	if err = i.importAll(ctx, logger); err != nil {
		return err
	}
	if err = i.ledger.record(ctx); err != nil {
		return fmt.Errorf("error recording the ProjectConfig import: %w", err)
	}
	logger.Info("recorded the ProjectConfig import")
	return nil
}

// importAll imports every ProjectConfig resource that can be. Only a missing
// ProjectConfig kind means there is nothing to import; any other failure is
// returned, so that the import is not recorded.
func (i *Importer) importAll(ctx context.Context, logger *logging.Logger) error {
	if _, err := i.mapper.RESTMapping(projectConfigGVK.GroupKind(), projectConfigGVK.Version); err != nil {
		if meta.IsNoMatchError(err) {
			logger.Info("the ProjectConfig kind does not exist; there is nothing to import")
			return nil
		}
		return fmt.Errorf("error resolving the ProjectConfig kind: %w", err)
	}
	if err := i.waitForPolicyBinding(ctx, logger); err != nil {
		return err
	}
	list := &unstructured.UnstructuredList{}
	list.SetGroupVersionKind(projectConfigGVK.GroupVersion().WithKind(projectConfigGVK.Kind + "List"))
	if err := i.kube.List(ctx, list); err != nil {
		return fmt.Errorf("error listing ProjectConfig resources: %w", err)
	}
	contested := contestedPaths(list.Items)
	var imported, existing, skipped int
	for idx := range list.Items {
		cr := &list.Items[idx]
		crLogger := logger.WithValues("namespace", cr.GetNamespace(), "name", cr.GetName())
		if cr.GetDeletionTimestamp() != nil {
			crLogger.Info("not importing a ProjectConfig resource that is being deleted")
			skipped++
			continue
		}
		project := &kargoapi.Project{}
		err := i.kube.Get(ctx, client.ObjectKey{Name: cr.GetNamespace()}, project)
		if apierrors.IsNotFound(err) {
			crLogger.Info("not importing a ProjectConfig resource whose namespace has no Project")
			skipped++
			continue
		}
		if err != nil {
			return fmt.Errorf("error getting Project %q: %w", cr.GetNamespace(), err)
		}
		cfg := &kargoapi.ProjectConfig{}
		if err = runtime.DefaultUnstructuredConverter.FromUnstructured(cr.Object, cfg); err != nil {
			return fmt.Errorf(
				"error converting ProjectConfig resource %q in namespace %q: %w",
				cr.GetName(), cr.GetNamespace(), err,
			)
		}
		cfg.Status.WebhookReceivers = withoutContested(cfg.Status.WebhookReceivers, contested, crLogger)
		inserted, err := i.store.Import(ctx, cfg, string(project.UID))
		if err != nil {
			return fmt.Errorf(
				"error importing ProjectConfig resource %q in namespace %q: %w",
				cr.GetName(), cr.GetNamespace(), err,
			)
		}
		if !inserted {
			crLogger.Info("not importing a ProjectConfig resource whose Project has a ProjectConfig already")
			existing++
			continue
		}
		crLogger.Debug("imported ProjectConfig resource")
		imported++
	}
	logger.Info(
		"imported ProjectConfig resources",
		"imported", imported,
		"existing", existing,
		"skipped", skipped,
	)
	return nil
}

// waitForPolicyBinding blocks until the binding that makes ProjectConfig
// resources read-only exists. While pods roll, an API server of the previous
// release can still write the resources, and Helm may create the binding
// after the Deployments. Once it exists, such a write fails visibly instead
// of being lost to an import that has already read the resource.
func (i *Importer) waitForPolicyBinding(ctx context.Context, logger *logging.Logger) error {
	waiting := false
	err := wait.PollUntilContextCancel(ctx, i.pollInterval, true, func(ctx context.Context) (bool, error) {
		err := i.kube.Get(
			ctx,
			client.ObjectKey{Name: policyBindingName},
			&admissionregistrationv1.ValidatingAdmissionPolicyBinding{},
		)
		if apierrors.IsNotFound(err) {
			if !waiting {
				logger.Info(
					"waiting for the ValidatingAdmissionPolicyBinding that makes "+
						"ProjectConfig resources read-only",
					"binding", policyBindingName,
				)
				waiting = true
			}
			return false, nil
		}
		return err == nil, err
	})
	if err != nil {
		return fmt.Errorf(
			"error waiting for ValidatingAdmissionPolicyBinding %q: %w",
			policyBindingName, err,
		)
	}
	return nil
}

// contestedPaths returns the webhook receiver paths that more than one
// ProjectConfig resource claims. None of their claimants' receivers for them
// is imported: before the import, such a path resolved to no ProjectConfig.
func contestedPaths(crs []unstructured.Unstructured) map[string]struct{} {
	owners := map[string]string{}
	contested := map[string]struct{}{}
	for idx := range crs {
		cr := &crs[idx]
		owner := cr.GetNamespace() + "/" + cr.GetName()
		receivers, _, _ := unstructured.NestedSlice(cr.Object, "status", "webhookReceivers")
		for _, receiver := range receivers {
			r, ok := receiver.(map[string]any)
			if !ok {
				continue
			}
			path, ok := r["path"].(string)
			if !ok || path == "" {
				continue
			}
			if prev, seen := owners[path]; seen && prev != owner {
				contested[path] = struct{}{}
				continue
			}
			owners[path] = owner
		}
	}
	return contested
}

// withoutContested returns receivers without those whose paths are
// contested.
func withoutContested(
	receivers []kargoapi.WebhookReceiverDetails,
	contested map[string]struct{},
	logger *logging.Logger,
) []kargoapi.WebhookReceiverDetails {
	kept := receivers[:0:0]
	for _, receiver := range receivers {
		if _, ok := contested[receiver.Path]; ok {
			logger.Info(
				"not importing a webhook receiver whose path another ProjectConfig "+
					"resource claims; it is restored when the ProjectConfig is reconciled "+
					"if its path is then unique",
				"receiver", receiver.Name,
				"path", receiver.Path,
			)
			continue
		}
		kept = append(kept, receiver)
	}
	return kept
}

// pgLedger is the importLedger kept in PostgreSQL: the import is recorded in
// the data_imports table, and its lock is a session-level advisory lock.
type pgLedger struct {
	pool *pgxpool.Pool
}

func (l *pgLedger) completed(ctx context.Context) (bool, error) {
	_, err := database.New(l.pool).GetDataImport(ctx, importName)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// lock takes the advisory lock on a connection of its own, which it keeps
// until the lock is released. The lock belongs to that connection's session,
// so the connection is closed, which releases the lock, if unlocking fails,
// rather than returned to the pool still holding it.
func (l *pgLedger) lock(ctx context.Context) (func(), error) {
	conn, err := l.pool.Acquire(ctx)
	if err != nil {
		return nil, fmt.Errorf("error acquiring a database connection: %w", err)
	}
	if _, err = conn.Exec(ctx, "SELECT pg_advisory_lock($1)", advisoryLockKey); err != nil {
		// An interrupted wait leaves it unknown whether the lock was taken.
		closeConn(conn)
		return nil, err
	}
	return func() {
		unlockCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), unlockTimeout)
		defer cancel()
		var unlocked bool
		err := conn.QueryRow(unlockCtx, "SELECT pg_advisory_unlock($1)", advisoryLockKey).Scan(&unlocked)
		if err != nil || !unlocked {
			logging.LoggerFromContext(ctx).Error(
				err, "error releasing the ProjectConfig import lock; closing its connection",
			)
			closeConn(conn)
			return
		}
		conn.Release()
	}, nil
}

func (l *pgLedger) record(ctx context.Context) error {
	return database.New(l.pool).RecordDataImport(ctx, importName)
}

// closeConn closes a pooled connection instead of returning it to the pool.
func closeConn(conn *pgxpool.Conn) {
	closeCtx, cancel := context.WithTimeout(context.Background(), unlockTimeout)
	defer cancel()
	_ = conn.Hijack().Close(closeCtx)
}
