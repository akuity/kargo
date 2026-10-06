// Package legacysecrets contains a one-time startup cleanup for a hazard
// left behind by the removal (in v1.12.0) of the automatic Secret migration
// that used to bridge the deprecated global.clusterSecretsNamespace and
// controller.globalCredentials.namespaces settings to their replacements,
// global.systemResources.namespace and global.sharedResources.namespace.
package legacysecrets

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/api"
	"github.com/akuity/kargo/pkg/logging"
)

// RemoveOrphanedFinalizers strips kargoapi.FinalizerName from any Secret
// outside of excludedNamespaces.
//
// Context: the now-removed migration reconciler (see git history for
// pkg/controller/management/secrets) stamped kargoapi.FinalizerName onto
// every source Secret it synced from a deprecated namespace, intending to
// remove that finalizer itself once the source Secret or its namespace was
// deleted. Now that the reconciler is gone, nothing ever removes it. This
// reliably bites on upgrade to v1.12.0+: global.clusterSecretsNamespace's
// namespace was chart-managed, so Helm prunes it the moment its templates
// disappear from the new chart version, and namespace deletion cascades
// into deleting the Secrets within it -- which hangs forever on any Secret
// still carrying the orphaned finalizer.
//
// This is a one-time, idempotent cleanup: once it has run successfully
// against a given cluster, no Secret should carry this finalizer for this
// reason again, so later runs are cheap no-ops. It runs once at management
// controller startup rather than as a dedicated reconciler, since there is
// nothing ongoing to watch for.
//
// excludedNamespaces MUST include the current, live systemResources and
// sharedResources namespaces: pkg/controller/management/replication uses
// this exact same finalizer, on the exact same resource kind, for an
// unrelated and still-active purpose (tracking source Secrets/ConfigMaps
// in the shared resources namespace pending replication to Project
// namespaces). This function must never interfere with that.
//
// reader must be an uncached client.Reader (e.g. a Manager's APIReader).
// The management controller's cache restricts Secret access to a fixed
// set of namespaces, but the deprecated namespaces this cleanup targets
// could be named anything an operator chose, so listing must bypass that
// cache entirely. Writes are unaffected by the same restriction, since
// controller-runtime clients always send mutating requests directly to the
// API server rather than through the cache.
//
// TODO(fuskovic): Remove this cleanup once we're confident no cluster is
// upgrading directly from a version <= v1.11.x anymore (tentatively
// v1.14.0 or later).
func RemoveOrphanedFinalizers(
	ctx context.Context,
	reader client.Reader,
	writer client.Client,
	excludedNamespaces ...string,
) error {
	excluded := make(map[string]bool, len(excludedNamespaces))
	for _, ns := range excludedNamespaces {
		excluded[ns] = true
	}

	secrets := &corev1.SecretList{}
	if err := reader.List(ctx, secrets); err != nil {
		return fmt.Errorf("error listing Secrets: %w", err)
	}

	logger := logging.LoggerFromContext(ctx)
	for i := range secrets.Items {
		secret := &secrets.Items[i]
		if excluded[secret.Namespace] ||
			!controllerutil.ContainsFinalizer(secret, kargoapi.FinalizerName) {
			continue
		}
		if err := api.RemoveFinalizer(ctx, writer, secret); err != nil {
			return fmt.Errorf(
				"error removing orphaned finalizer from Secret %q in namespace %q: %w",
				secret.Name, secret.Namespace, err,
			)
		}
		logger.Info(
			"removed finalizer orphaned by the removed secret migration reconciler",
			"namespace", secret.Namespace,
			"name", secret.Name,
		)
	}
	return nil
}
