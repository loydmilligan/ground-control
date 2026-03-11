// Package watch provides file watching for Flight Deck state updates
package watch

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/mmariani/ground-control/internal/sidecar"
)

// StateUpdate represents a state change notification
type StateUpdate struct {
	ProjectPath string
	State       *sidecar.ProjectState
	Error       error
}

// Watcher watches project state files for changes
type Watcher struct {
	watcher     *fsnotify.Watcher
	updates     chan StateUpdate
	done        chan struct{}
	mu          sync.Mutex
	watchedDirs map[string]bool // track watched directories
}

// New creates a new state file watcher
func New() (*Watcher, error) {
	fw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	w := &Watcher{
		watcher:     fw,
		updates:     make(chan StateUpdate, 10),
		done:        make(chan struct{}),
		watchedDirs: make(map[string]bool),
	}

	go w.run()
	return w, nil
}

// WatchProject starts watching a project's state.json
func (w *Watcher) WatchProject(projectPath string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	gcPath := filepath.Join(projectPath, ".gc")
	if w.watchedDirs[gcPath] {
		return nil // Already watching
	}

	// Ensure .gc directory exists
	if _, err := os.Stat(gcPath); os.IsNotExist(err) {
		return err
	}

	if err := w.watcher.Add(gcPath); err != nil {
		return err
	}

	w.watchedDirs[gcPath] = true
	return nil
}

// UnwatchProject stops watching a project
func (w *Watcher) UnwatchProject(projectPath string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	gcPath := filepath.Join(projectPath, ".gc")
	if !w.watchedDirs[gcPath] {
		return nil // Not watching
	}

	if err := w.watcher.Remove(gcPath); err != nil {
		return err
	}

	delete(w.watchedDirs, gcPath)
	return nil
}

// Updates returns the channel for state updates
func (w *Watcher) Updates() <-chan StateUpdate {
	return w.updates
}

// Close stops the watcher
func (w *Watcher) Close() error {
	close(w.done)
	return w.watcher.Close()
}

func (w *Watcher) run() {
	// Debounce timer to avoid rapid-fire updates
	var debounceTimer *time.Timer
	var pendingPath string

	for {
		select {
		case <-w.done:
			if debounceTimer != nil {
				debounceTimer.Stop()
			}
			return

		case event, ok := <-w.watcher.Events:
			if !ok {
				return
			}

			// Only care about state.json changes
			if filepath.Base(event.Name) != "state.json" {
				continue
			}

			// Only care about write/create events
			if event.Op&(fsnotify.Write|fsnotify.Create) == 0 {
				continue
			}

			// Debounce: wait 100ms before sending update
			projectPath := filepath.Dir(filepath.Dir(event.Name))
			pendingPath = projectPath

			if debounceTimer != nil {
				debounceTimer.Stop()
			}

			debounceTimer = time.AfterFunc(100*time.Millisecond, func() {
				w.sendUpdate(pendingPath)
			})

		case err, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
			w.updates <- StateUpdate{Error: err}
		}
	}
}

func (w *Watcher) sendUpdate(projectPath string) {
	statePath := filepath.Join(projectPath, ".gc", "state.json")

	data, err := os.ReadFile(statePath)
	if err != nil {
		w.updates <- StateUpdate{
			ProjectPath: projectPath,
			Error:       err,
		}
		return
	}

	var state sidecar.ProjectState
	if err := json.Unmarshal(data, &state); err != nil {
		w.updates <- StateUpdate{
			ProjectPath: projectPath,
			Error:       err,
		}
		return
	}

	w.updates <- StateUpdate{
		ProjectPath: projectPath,
		State:       &state,
	}
}
