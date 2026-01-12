package argocd

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes/fake"
	k8stest "k8s.io/client-go/testing"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func TestWatcher_SyncSecrets(t *testing.T) {
	testCases := map[string]struct {
		secrets        []*corev1.Secret
		expectedShards map[string]string
	}{
		"no secrets": {
			secrets:        nil,
			expectedShards: map[string]string{},
		},
		"single valid secret": {
			secrets: []*corev1.Secret{
				newArgoCDShardSecret("argocd-shard-prod", "production", "https://argocd.example.com"),
			},
			expectedShards: map[string]string{
				"production": "https://argocd.example.com",
			},
		},
		"multiple valid secrets": {
			secrets: []*corev1.Secret{
				newArgoCDShardSecret("argocd-shard-prod", "production", "https://argocd-prod.example.com"),
				newArgoCDShardSecret("argocd-shard-staging", "staging", "https://argocd-staging.example.com"),
				newArgoCDShardSecret("argocd-shard-default", "", "https://argocd.example.com"),
			},
			expectedShards: map[string]string{
				"production": "https://argocd-prod.example.com",
				"staging":    "https://argocd-staging.example.com",
				"":           "https://argocd.example.com",
			},
		},
		"secret missing url is skipped": {
			secrets: []*corev1.Secret{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "bad-secret",
						Namespace: "kargo",
						Labels: map[string]string{
							kargoapi.LabelKeyArgoCDShard: kargoapi.LabelValueTrue,
						},
					},
					Data: map[string][]byte{
						SecretFieldName: []byte("bad-shard"),
						// Missing URL
					},
				},
				newArgoCDShardSecret("good-secret", "good", "https://argocd.example.com"),
			},
			expectedShards: map[string]string{
				"good": "https://argocd.example.com",
			},
		},
		"secret with empty url is skipped": {
			secrets: []*corev1.Secret{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "empty-url-secret",
						Namespace: "kargo",
						Labels: map[string]string{
							kargoapi.LabelKeyArgoCDShard: kargoapi.LabelValueTrue,
						},
					},
					Data: map[string][]byte{
						SecretFieldName: []byte("empty"),
						SecretFieldURL:  []byte(""),
					},
				},
			},
			expectedShards: map[string]string{},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			clientset := fake.NewSimpleClientset()

			// Create secrets
			for _, secret := range tc.secrets {
				_, err := clientset.CoreV1().Secrets(secret.Namespace).Create(
					context.Background(), secret, metav1.CreateOptions{},
				)
				require.NoError(t, err)
			}

			store := NewURLStore()
			store.SetStaticShards(nil, "argocd")

			watcher := NewWatcherWithClient(
				WatcherConfig{KargoNamespace: "kargo"},
				clientset,
				store,
			)

			err := watcher.syncSecrets(context.Background())
			require.NoError(t, err)

			shards := store.GetShards()
			assert.Len(t, shards, len(tc.expectedShards))

			for name, expectedURL := range tc.expectedShards {
				require.Contains(t, shards, name)
				assert.Equal(t, expectedURL, shards[name].Url)
			}
		})
	}
}

func TestWatcher_ProcessSecret(t *testing.T) {
	store := NewURLStore()
	store.SetStaticShards(nil, "argocd")

	watcher := NewWatcherWithClient(
		WatcherConfig{KargoNamespace: "kargo"},
		fake.NewSimpleClientset(),
		store,
	)

	// Process a valid secret
	secret := newArgoCDShardSecret("test-secret", "test-shard", "https://test.example.com")
	watcher.processSecret(context.Background(), secret)

	shards := store.GetShards()
	require.Len(t, shards, 1)
	assert.Equal(t, "https://test.example.com", shards["test-shard"].Url)
}

func TestWatcher_DeleteSecret(t *testing.T) {
	store := NewURLStore()
	store.SetStaticShards(nil, "argocd")

	watcher := NewWatcherWithClient(
		WatcherConfig{KargoNamespace: "kargo"},
		fake.NewSimpleClientset(),
		store,
	)

	// Add a shard first
	store.UpdateDynamicShard("to-delete", "https://delete-me.example.com")

	shards := store.GetShards()
	require.Len(t, shards, 1)

	// Delete via watcher
	secret := newArgoCDShardSecret("test-secret", "to-delete", "https://delete-me.example.com")
	watcher.deleteSecret(context.Background(), secret)

	shards = store.GetShards()
	assert.Empty(t, shards)
}

func TestWatcher_WatchEvents(t *testing.T) {
	clientset := fake.NewSimpleClientset()

	// Create a fake watcher
	fakeWatcher := watch.NewFake()
	clientset.PrependWatchReactor("secrets", k8stest.DefaultWatchReactor(fakeWatcher, nil))

	store := NewURLStore()
	store.SetStaticShards(nil, "argocd")

	watcher := NewWatcherWithClient(
		WatcherConfig{KargoNamespace: "kargo"},
		clientset,
		store,
	)

	ctx, cancel := context.WithCancel(context.Background())

	// Start watching in a goroutine
	errCh := make(chan error, 1)
	go func() {
		errCh <- watcher.watchSecrets(ctx)
	}()

	// Give the watcher time to start
	time.Sleep(100 * time.Millisecond)

	// Send an Add event
	secret := newArgoCDShardSecret("new-secret", "new-shard", "https://new.example.com")
	fakeWatcher.Add(secret)

	// Give time for processing
	time.Sleep(100 * time.Millisecond)

	shards := store.GetShards()
	require.Len(t, shards, 1)
	assert.Equal(t, "https://new.example.com", shards["new-shard"].Url)

	// Send a Modify event
	secret.Data[SecretFieldURL] = []byte("https://updated.example.com")
	fakeWatcher.Modify(secret)

	time.Sleep(100 * time.Millisecond)

	shards = store.GetShards()
	assert.Equal(t, "https://updated.example.com", shards["new-shard"].Url)

	// Send a Delete event
	fakeWatcher.Delete(secret)

	time.Sleep(100 * time.Millisecond)

	shards = store.GetShards()
	assert.Empty(t, shards)

	// Clean up
	cancel()
	fakeWatcher.Stop()

	select {
	case <-errCh:
		// Expected
	case <-time.After(time.Second):
		t.Fatal("Watcher did not stop")
	}
}

func TestWatcher_SecretWithoutNameField(t *testing.T) {
	store := NewURLStore()
	store.SetStaticShards(nil, "argocd")

	watcher := NewWatcherWithClient(
		WatcherConfig{KargoNamespace: "kargo"},
		fake.NewSimpleClientset(),
		store,
	)

	// Secret without name field should default to empty string
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "no-name-field",
			Namespace: "kargo",
			Labels: map[string]string{
				kargoapi.LabelKeyArgoCDShard: kargoapi.LabelValueTrue,
			},
		},
		Data: map[string][]byte{
			SecretFieldURL: []byte("https://default.example.com"),
			// No name field
		},
	}
	watcher.processSecret(context.Background(), secret)

	shards := store.GetShards()
	require.Len(t, shards, 1)
	assert.Equal(t, "https://default.example.com", shards[""].Url)
}

// newArgoCDShardSecret creates a secret with the ArgoCD shard label and data.
func newArgoCDShardSecret(secretName, shardName, url string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: "kargo",
			Labels: map[string]string{
				kargoapi.LabelKeyArgoCDShard: kargoapi.LabelValueTrue,
			},
		},
		Data: map[string][]byte{
			SecretFieldName: []byte(shardName),
			SecretFieldURL:  []byte(url),
		},
	}
}
