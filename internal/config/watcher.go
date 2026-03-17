package config

import (
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Watcher watches config files for changes and triggers a reload callback.
type Watcher struct {
	fsw       *fsnotify.Watcher
	reloadFn  func() error
	debounce  time.Duration
	mu        sync.Mutex
	timer     *time.Timer
	done      chan struct{}
	watchDirs map[string]string // dir -> target file (for non-existent file watching)
}

// WatchPaths returns the system and user config file paths to watch.
// Project config (.clincus.toml) and CLINCUS_CONFIG are out of scope.
func WatchPaths() []string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "/tmp"
	}
	return []string{
		"/etc/clincus/config.toml",
		filepath.Join(homeDir, ".config/clincus/config.toml"),
	}
}

// NewWatcher creates a config file watcher. reloadFn is called (debounced)
// when any watched file changes. Paths that don't exist yet are handled by
// watching their parent directory and detecting file creation.
func NewWatcher(paths []string, reloadFn func() error, debounce time.Duration) (*Watcher, error) {
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	w := &Watcher{
		fsw:       fsw,
		reloadFn:  reloadFn,
		debounce:  debounce,
		done:      make(chan struct{}),
		watchDirs: make(map[string]string),
	}

	for _, p := range paths {
		if _, statErr := os.Stat(p); statErr == nil {
			// File exists — watch it directly.
			if addErr := fsw.Add(p); addErr != nil {
				log.Printf("[config-watcher] warning: cannot watch %s: %v", p, addErr)
			}
		} else {
			// File does not exist — watch parent directory so we can detect creation (AC8).
			dir := filepath.Dir(p)
			if _, dirErr := os.Stat(dir); dirErr == nil {
				w.watchDirs[dir] = p
				if addErr := fsw.Add(dir); addErr != nil {
					log.Printf("[config-watcher] warning: cannot watch dir %s: %v", dir, addErr)
				}
			} else {
				log.Printf("[config-watcher] info: %s and parent dir do not exist, skipping", p)
			}
		}
	}

	return w, nil
}

// Start begins the event loop in a goroutine. Call Close to stop.
func (w *Watcher) Start() {
	go w.loop()
}

// Close stops the watcher and releases resources.
func (w *Watcher) Close() error {
	close(w.done)
	w.mu.Lock()
	if w.timer != nil {
		w.timer.Stop()
	}
	w.mu.Unlock()
	return w.fsw.Close()
}

func (w *Watcher) loop() {
	for {
		select {
		case <-w.done:
			return
		case event, ok := <-w.fsw.Events:
			if !ok {
				return
			}
			w.handleEvent(event)
		case err, ok := <-w.fsw.Errors:
			if !ok {
				return
			}
			log.Printf("[config-watcher] error: %v", err)
		}
	}
}

func (w *Watcher) handleEvent(event fsnotify.Event) {
	// Check if this is a directory event for a file that didn't exist at startup.
	if target, ok := w.watchDirs[filepath.Dir(event.Name)]; ok {
		// Only trigger if the event is about our target file.
		absEvent, _ := filepath.Abs(event.Name)
		absTarget, _ := filepath.Abs(target)
		if absEvent != absTarget {
			return
		}
		// If the target file was created, start watching it directly and
		// remove the directory watch.
		if event.Has(fsnotify.Create) {
			if err := w.fsw.Add(target); err == nil {
				dir := filepath.Dir(target)
				_ = w.fsw.Remove(dir)
				delete(w.watchDirs, dir)
			}
		}
	}

	// Trigger debounced reload on Write, Create, or Rename (AC6).
	if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) || event.Has(fsnotify.Rename) {
		w.scheduleReload()
	}
}

// scheduleReload resets the debounce timer. If multiple events arrive
// within the debounce window they collapse into a single reload (AC6).
func (w *Watcher) scheduleReload() {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.timer != nil {
		w.timer.Stop()
	}
	w.timer = time.AfterFunc(w.debounce, func() {
		if err := w.reloadFn(); err != nil {
			log.Printf("[config-watcher] reload failed: %v", err)
		}
	})
}
