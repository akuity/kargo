package argocd

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
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

	store.UpdateDynamicShard("to-delete", "https://delete-me.example.com")
	require.Len(t, store.GetShards(), 1)

	secret := newArgoCDShardSecret("test-secret", "to-delete", "https://delete-me.example.com")
	watcher.deleteSecret(context.Background(), secret)

	assert.Empty(t, store.GetShards())
}

func TestWatcher_WatchEvents(t *testing.T) {
	clientset := fake.NewSimpleClientset()
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
	errCh := make(chan error, 1)
	go func() {
		errCh <- watcher.watchSecrets(ctx)
	}()
	time.Sleep(100 * time.Millisecond)

	secret := newArgoCDShardSecret("new-secret", "new-shard", "https://new.example.com")
	fakeWatcher.Add(secret)
	time.Sleep(100 * time.Millisecond)

	shards := store.GetShards()
	require.Len(t, shards, 1)
	assert.Equal(t, "https://new.example.com", shards["new-shard"].Url)

	secret.Data[SecretFieldURL] = []byte("https://updated.example.com")
	fakeWatcher.Modify(secret)
	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, "https://updated.example.com", store.GetShards()["new-shard"].Url)

	fakeWatcher.Delete(secret)
	time.Sleep(100 * time.Millisecond)
	assert.Empty(t, store.GetShards())

	cancel()
	fakeWatcher.Stop()
	select {
	case <-errCh:
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
		},
	}
	watcher.processSecret(context.Background(), secret)

	shards := store.GetShards()
	require.Len(t, shards, 1)
	assert.Equal(t, "https://default.example.com", shards[""].Url)
}

func TestWatcher_ProcessSecret_NilData(t *testing.T) {
	store := NewURLStore()
	store.SetStaticShards(nil, "argocd")
	watcher := NewWatcherWithClient(
		WatcherConfig{KargoNamespace: "kargo"},
		fake.NewSimpleClientset(),
		store,
	)

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "nil-data-secret",
			Namespace: "kargo",
			Labels:    map[string]string{kargoapi.LabelKeyArgoCDShard: kargoapi.LabelValueTrue},
		},
		Data: nil,
	}
	watcher.processSecret(context.Background(), secret)
	assert.Empty(t, store.GetShards())
}

func TestWatcher_DeleteSecret_NilData(t *testing.T) {
	store := NewURLStore()
	store.SetStaticShards(nil, "argocd")
	watcher := NewWatcherWithClient(
		WatcherConfig{KargoNamespace: "kargo"},
		fake.NewSimpleClientset(),
		store,
	)

	store.UpdateDynamicShard("", "https://default.example.com")
	require.Len(t, store.GetShards(), 1)

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "nil-data-secret",
			Namespace: "kargo",
			Labels:    map[string]string{kargoapi.LabelKeyArgoCDShard: kargoapi.LabelValueTrue},
		},
		Data: nil,
	}
	watcher.deleteSecret(context.Background(), secret)
	assert.Empty(t, store.GetShards())
}

func TestWatcher_WatchSecrets_ErrorEvent(t *testing.T) {
	clientset := fake.NewSimpleClientset()
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
	errCh := make(chan error, 1)
	go func() {
		errCh <- watcher.watchSecrets(ctx)
	}()
	time.Sleep(100 * time.Millisecond)

	fakeWatcher.Error(&metav1.Status{Status: metav1.StatusFailure, Message: "test error"})
	time.Sleep(100 * time.Millisecond)
	assert.Empty(t, store.GetShards())

	cancel()
	fakeWatcher.Stop()
	select {
	case <-errCh:
	case <-time.After(time.Second):
		t.Fatal("Watcher did not stop")
	}
}

func TestWatcher_WatchSecrets_NonSecretObject(t *testing.T) {
	clientset := fake.NewSimpleClientset()
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
	errCh := make(chan error, 1)
	go func() {
		errCh <- watcher.watchSecrets(ctx)
	}()
	time.Sleep(100 * time.Millisecond)

	fakeWatcher.Add(&corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "not-a-secret", Namespace: "kargo"},
	})
	time.Sleep(100 * time.Millisecond)
	assert.Empty(t, store.GetShards())

	cancel()
	fakeWatcher.Stop()
	select {
	case <-errCh:
	case <-time.After(time.Second):
		t.Fatal("Watcher did not stop")
	}
}

func TestWatcher_WatchSecrets_ChannelClose(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	fakeWatcher := watch.NewFake()
	clientset.PrependWatchReactor("secrets", k8stest.DefaultWatchReactor(fakeWatcher, nil))

	store := NewURLStore()
	store.SetStaticShards(nil, "argocd")
	watcher := NewWatcherWithClient(
		WatcherConfig{KargoNamespace: "kargo"},
		clientset,
		store,
	)

	errCh := make(chan error, 1)
	go func() {
		errCh <- watcher.watchSecrets(context.Background())
	}()
	time.Sleep(100 * time.Millisecond)

	fakeWatcher.Stop()
	select {
	case err := <-errCh:
		assert.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("Watcher did not stop")
	}
}

func TestWatcher_Start_ContextCancellation(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	store := NewURLStore()
	store.SetStaticShards(nil, "argocd")
	watcher := NewWatcherWithClient(
		WatcherConfig{KargoNamespace: "kargo"},
		clientset,
		store,
	)

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		errCh <- watcher.Start(ctx)
	}()
	time.Sleep(100 * time.Millisecond)

	cancel()
	select {
	case err := <-errCh:
		assert.ErrorIs(t, err, context.Canceled)
	case <-time.After(2 * time.Second):
		t.Fatal("Start did not stop after context cancellation")
	}
}

func TestWatcher_Start_InitialSyncFailure(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	clientset.PrependReactor("list", "secrets", func(_ k8stest.Action) (bool, runtime.Object, error) {
		return true, nil, assert.AnError
	})

	store := NewURLStore()
	store.SetStaticShards(nil, "argocd")
	watcher := NewWatcherWithClient(
		WatcherConfig{KargoNamespace: "kargo"},
		clientset,
		store,
	)

	assert.Error(t, watcher.Start(context.Background()))
}

func TestWatcher_SyncSecrets_ListError(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	clientset.PrependReactor("list", "secrets", func(_ k8stest.Action) (bool, runtime.Object, error) {
		return true, nil, assert.AnError
	})

	store := NewURLStore()
	store.SetStaticShards(nil, "argocd")
	watcher := NewWatcherWithClient(
		WatcherConfig{KargoNamespace: "kargo"},
		clientset,
		store,
	)

	assert.Error(t, watcher.syncSecrets(context.Background()))
}

func TestWatcher_WatchSecrets_WatchError(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	clientset.PrependWatchReactor("secrets", func(_ k8stest.Action) (bool, watch.Interface, error) {
		return true, nil, assert.AnError
	})

	store := NewURLStore()
	store.SetStaticShards(nil, "argocd")
	watcher := NewWatcherWithClient(
		WatcherConfig{KargoNamespace: "kargo"},
		clientset,
		store,
	)

	assert.Error(t, watcher.watchSecrets(context.Background()))
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
