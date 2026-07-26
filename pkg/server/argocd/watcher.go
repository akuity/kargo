package argocd

import (
	"context"
	"time"

	"github.com/kelseyhightower/envconfig"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/logging"
)

const (
	// SecretFieldName is the key in secret data for shard name.
	SecretFieldName = "name"
	// SecretFieldURL is the key in secret data for ArgoCD URL.
	SecretFieldURL = "url"
)

// WatcherConfig holds configuration for the secret watcher.
type WatcherConfig struct {
	KargoNamespace string `envconfig:"KARGO_NAMESPACE" default:"kargo"`
}

// WatcherConfigFromEnv returns a WatcherConfig populated from environment variables.
func WatcherConfigFromEnv() WatcherConfig {
	cfg := WatcherConfig{}
	envconfig.MustProcess("", &cfg)
	return cfg
}

// Watcher watches for ArgoCD shard secrets and updates the URLStore.
type Watcher struct {
	cfg       WatcherConfig
	clientset kubernetes.Interface
	store     URLStore
}

// NewWatcher creates a new secret watcher.
func NewWatcher(restConfig *rest.Config, store URLStore) (*Watcher, error) {
	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}
	return &Watcher{
		cfg:       WatcherConfigFromEnv(),
		clientset: clientset,
		store:     store,
	}, nil
}

// NewWatcherWithClient creates a new secret watcher with a provided clientset.
// This is useful for testing.
func NewWatcherWithClient(cfg WatcherConfig, clientset kubernetes.Interface, store URLStore) *Watcher {
	return &Watcher{
		cfg:       cfg,
		clientset: clientset,
		store:     store,
	}
}

// Start begins watching for secrets. It performs an initial sync and then
// watches for changes. This is a blocking call that runs until the context
// is canceled.
func (w *Watcher) Start(ctx context.Context) error {
	logger := logging.LoggerFromContext(ctx)
	logger.Info(
		"Starting ArgoCD shard secret watcher",
		"namespace", w.cfg.KargoNamespace,
	)

	if err := w.syncSecrets(ctx); err != nil {
		logger.Error(err, "Initial sync failed")
		return err
	}

	for {
		select {
		case <-ctx.Done():
			logger.Info("ArgoCD shard secret watcher stopped")
			return ctx.Err()
		default:
			if err := w.watchSecrets(ctx); err != nil {
				if ctx.Err() != nil {
					return ctx.Err()
				}
				logger.Error(err, "Watch error, retrying...")
				time.Sleep(5 * time.Second)
			}
		}
	}
}

// syncSecrets performs initial load of all ArgoCD shard secrets.
func (w *Watcher) syncSecrets(ctx context.Context) error {
	logger := logging.LoggerFromContext(ctx)

	labelSelector := labels.Set{
		kargoapi.LabelKeyArgoCDShard: kargoapi.LabelValueTrue,
	}.AsSelector().String()

	secrets, err := w.clientset.CoreV1().Secrets(w.cfg.KargoNamespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return err
	}

	for i := range secrets.Items {
		w.processSecret(ctx, &secrets.Items[i])
	}

	logger.Info(
		"Initial sync complete",
		"secretsProcessed", len(secrets.Items),
	)
	return nil
}

// watchSecrets sets up a watch for ArgoCD shard secrets.
func (w *Watcher) watchSecrets(ctx context.Context) error {
	logger := logging.LoggerFromContext(ctx)

	labelSelector := labels.Set{
		kargoapi.LabelKeyArgoCDShard: kargoapi.LabelValueTrue,
	}.AsSelector().String()

	watcher, err := w.clientset.CoreV1().Secrets(w.cfg.KargoNamespace).Watch(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return err
	}
	defer watcher.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case event, ok := <-watcher.ResultChan():
			if !ok {
				return nil
			}

			secret, ok := event.Object.(*corev1.Secret)
			if !ok {
				logger.Debug("Received non-secret object, skipping")
				continue
			}

			switch event.Type {
			case watch.Added, watch.Modified:
				w.processSecret(ctx, secret)
			case watch.Deleted:
				w.deleteSecret(ctx, secret)
			case watch.Error:
				logger.Error(nil, "Watch error event received")
			}
		}
	}
}

// processSecret extracts shard data from a secret and updates the store.
func (w *Watcher) processSecret(ctx context.Context, secret *corev1.Secret) {
	logger := logging.LoggerFromContext(ctx)

	if secret.Data == nil {
		logger.Debug(
			"Secret has no data, skipping",
			"secret", secret.Name,
		)
		return
	}

	nameBytes, hasName := secret.Data[SecretFieldName]
	urlBytes, hasURL := secret.Data[SecretFieldURL]

	if !hasURL {
		logger.Info(
			"Secret missing required 'url' field, skipping",
			"secret", secret.Name,
		)
		return
	}

	url := string(urlBytes)
	if url == "" {
		logger.Info(
			"Secret has empty 'url' field, skipping",
			"secret", secret.Name,
		)
		return
	}

	name := ""
	if hasName {
		name = string(nameBytes)
	}

	w.store.UpdateDynamicShard(name, url)
	logger.Info(
		"Updated ArgoCD shard from secret",
		"secret", secret.Name,
		"shardName", name,
		"url", url,
	)
}

// deleteSecret removes a shard from the store when its secret is deleted.
func (w *Watcher) deleteSecret(ctx context.Context, secret *corev1.Secret) {
	logger := logging.LoggerFromContext(ctx)

	name := ""
	if secret.Data != nil {
		if nameBytes, ok := secret.Data[SecretFieldName]; ok {
			name = string(nameBytes)
		}
	}

	w.store.DeleteDynamicShard(name)
	logger.Info(
		"Deleted ArgoCD shard from secret",
		"secret", secret.Name,
		"shardName", name,
	)
}
