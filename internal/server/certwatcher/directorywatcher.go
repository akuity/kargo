package certwatcher

import (
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
)

type directoryWatcher struct {
	watcher      *fsnotify.Watcher
	watchedFiles map[string]string
	done         chan struct{}
}

func (d *directoryWatcher) watch() <-chan struct{} {
	events := make(chan struct{})
	go func() {
		defer close(events)
		for {
			select {
			case <-d.done:
				return
			case event, ok := <-d.watcher.Events:
				if !ok {
					return
				}
				if d.shouldSendEvent(event) {
					select {
					case events <- struct{}{}:
					case <-d.done:
						return
					}
				}
			}
		}
	}()

	return events
}

func (d *directoryWatcher) shouldSendEvent(event fsnotify.Event) bool {
	sleepTime := 10 * time.Millisecond
	eventPath, _ := filepath.Abs(event.Name)
	eventPath, _ = filepath.EvalSymlinks(eventPath)

	for abs, previous := range d.watchedFiles {
		currentWatchedPath, _ := filepath.Abs(abs)
		switch {
		case currentWatchedPath == "":
			// watched file was removed; wait for write event to trigger reload
			d.watchedFiles[abs] = ""
		case currentWatchedPath != previous:
			// File previously didn't exist; send a signal to the caller
			time.Sleep(sleepTime)
			d.watchedFiles[abs] = currentWatchedPath
			return true
		case eventPath == currentWatchedPath && isUpdatedFileEvent(event):
			// File was modified so send a signal to the caller
			time.Sleep(sleepTime)
			d.watchedFiles[abs] = currentWatchedPath
			return true
		}
	}
	return false
}

func isUpdatedFileEvent(event fsnotify.Event) bool {
	return (event.Op&fsnotify.Write) == fsnotify.Write || (event.Op&fsnotify.Create) == fsnotify.Create
}
