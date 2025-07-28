package certwatcher

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNewCertWatcher(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		tempDir := t.TempDir()
		certPath := filepath.Join(tempDir, "tls.crt")
		keyPath := filepath.Join(tempDir, "tls.key")
		require.NoError(t, os.WriteFile(certPath, []byte("cert"), 0600))
		require.NoError(t, os.WriteFile(keyPath, []byte("key"), 0600))

		cw, err := NewCertWatcher(certPath, keyPath)
		require.NoError(t, err)
		require.NotNil(t, cw)
		require.Len(t, cw.directories, 1)
	})

	t.Run("cert path does not exist", func(t *testing.T) {
		tempDir := t.TempDir()
		certPath := filepath.Join(tempDir, "tls.crt")
		keyPath := filepath.Join(tempDir, "tls.key")
		require.NoError(t, os.WriteFile(keyPath, []byte("key"), 0600))

		_, err := NewCertWatcher(certPath, keyPath)
		require.Error(t, err)
	})

	t.Run("key path does not exist", func(t *testing.T) {
		tempDir := t.TempDir()
		certPath := filepath.Join(tempDir, "tls.crt")
		keyPath := filepath.Join(tempDir, "tls.key")
		require.NoError(t, os.WriteFile(certPath, []byte("cert"), 0600))

		_, err := NewCertWatcher(certPath, keyPath)
		require.Error(t, err)
	})
}

func TestCertWatcher(t *testing.T) {
	tempDir := t.TempDir()
	certPath := filepath.Join(tempDir, "tls.crt")
	keyPath := filepath.Join(tempDir, "tls.key")
	require.NoError(t, os.WriteFile(certPath, []byte("cert"), 0600))
	require.NoError(t, os.WriteFile(keyPath, []byte("key"), 0600))

	cw, err := NewCertWatcher(certPath, keyPath)
	require.NoError(t, err)
	require.NotNil(t, cw)

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		cw.Run()
	}()

	// Wait a bit for the watcher to start
	time.Sleep(100 * time.Millisecond)

	// Update the cert file
	require.NoError(t, os.WriteFile(certPath, []byte("new cert"), 0600))

	select {
	case <-cw.Events():
		// All good
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for event")
	}

	// Update the key file
	require.NoError(t, os.WriteFile(keyPath, []byte("new key"), 0600))

	select {
	case <-cw.Events():
		// All good
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for event")
	}

	cw.Close()
	wg.Wait()
}
