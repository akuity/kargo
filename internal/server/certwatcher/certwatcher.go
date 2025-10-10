package certwatcher

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/fsnotify/fsnotify"
)

// Certwatcher watches for any changes to a certificate and key pair.
// It is used to reload the certificate and key pair when they are updated.
// It is also used to reload the certificate and key pair when the file is
// created.
type CertWatcher struct {
	directories map[string]*directoryWatcher
	notify      chan struct{}
}

func NewCertWatcher(certPath, keyPath string) (*CertWatcher, error) {
	certWatcher := &CertWatcher{
		directories: make(map[string]*directoryWatcher),
		notify:      make(chan struct{}),
	}

	err := certWatcher.addPath(certPath)
	if err != nil {
		return nil, err
	}

	err = certWatcher.addPath(keyPath)
	if err != nil {
		return nil, err
	}

	return certWatcher, nil
}

// Events returns a channel that will be notified when the certificate or key
// pair is updated.
func (c *CertWatcher) Events() <-chan struct{} {
	return c.notify
}

// Run starts the certwatcher and watches for changes to the certificate and
// key pair.  Run blocks until the certwatcher is closed.
func (c *CertWatcher) Run() {
	defer close(c.notify)
	wg := sync.WaitGroup{}
	for _, dirWatcher := range c.directories {
		wg.Add(1)
		go func(dirWatcher *directoryWatcher) {
			defer wg.Done()
			events := dirWatcher.watch()
			for range events {
				select {
				case c.notify <- struct{}{}:
				case <-dirWatcher.done:
					return
				}
			}
		}(dirWatcher)
	}
	wg.Wait()
}

// Close closes the certwatcher and stops watching for changes.
func (c *CertWatcher) Close() {
	for _, dirWatcher := range c.directories {
		close(dirWatcher.done)
		dirWatcher.watcher.Close()
	}
}

func (c *CertWatcher) addPath(path string) error {
	_, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("failed to stat %q: %w", path, err)
	}

	absolutePath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to get absolute path for %q: %w", path, err)
	}
	fp, _ := filepath.EvalSymlinks(absolutePath)

	fileDir := filepath.Dir(absolutePath)

	dirWatcher, ok := c.directories[fileDir]
	if ok {
		dirWatcher.watchedFiles[absolutePath] = fp
		return nil
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create fsnotify watcher: %w", err)
	}
	err = watcher.Add(fileDir)
	if err != nil {
		return fmt.Errorf("failed to add %q to fsnotify watcher: %w", fileDir, err)
	}

	dirWatcher = &directoryWatcher{
		watcher:      watcher,
		watchedFiles: make(map[string]string),
		done:         make(chan struct{}),
	}

	dirWatcher.watchedFiles[absolutePath] = fp
	c.directories[fileDir] = dirWatcher
	return nil
}
