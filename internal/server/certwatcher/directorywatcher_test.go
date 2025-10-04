package certwatcher

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/stretchr/testify/require"
)

func TestDirectoryWatcher(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "file.txt")
	require.NoError(t, os.WriteFile(filePath, []byte("hello"), 0600))

	watcher, err := fsnotify.NewWatcher()
	require.NoError(t, err)
	require.NoError(t, watcher.Add(tempDir))

	dw := &directoryWatcher{
		watcher:      watcher,
		watchedFiles: make(map[string]string),
		done:         make(chan struct{}),
	}
	absPath, err := filepath.Abs(filePath)
	require.NoError(t, err)
	dw.watchedFiles[absPath], _ = filepath.EvalSymlinks(absPath)

	events := dw.watch()

	// Test file modification
	require.NoError(t, os.WriteFile(filePath, []byte("world"), 0600))
	select {
	case <-events:
		// Expected event
	case <-time.After(1 * time.Second):
		t.Fatal("timed out waiting for event")
	}

	// Test file removal and recreation
	require.NoError(t, os.Remove(filePath))
	// Without a short pause, the test can be flaky
	time.Sleep(10 * time.Millisecond)
	require.NoError(t, os.WriteFile(filePath, []byte("new"), 0600))
	select {
	case <-events:
		// Expected event
	case <-time.After(1 * time.Second):
		t.Fatal("timed out waiting for event")
	}

	close(dw.done)

	timeout := time.After(1 * time.Second)
CLOSED:
	for {
		select {
		case _, ok := <-events:
			if !ok {
				break CLOSED
			}
			continue CLOSED
		case <-timeout:
			t.Fatal("timed out waiting for events to be closed")
		}
	}
}
