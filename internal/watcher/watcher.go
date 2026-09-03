package watcher

import (
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// DefaultMaxDepth is the maximum directory depth to recurse when watching.
const DefaultMaxDepth = 3

// MaxWatchedDirs is the cap on total watched directories to prevent fd exhaustion.
const MaxWatchedDirs = 500

// excludedDirs are directory names that should never be watched.
var excludedDirs = map[string]bool{
	".git":         true,
	"node_modules": true,
	"vendor":       true,
	"dist":         true,
	"build":        true,
	"__pycache__":  true,
	".next":        true,
	"target":       true,
}

type FileEvent struct {
	Path string
	Time time.Time
}

type Watcher struct {
	w        *fsnotify.Watcher
	Events   chan FileEvent
	recent   []FileEvent
	mu       sync.Mutex
	running  bool
	rootDir  string
	maxDepth int
	dirCount int
}

func New() (*Watcher, error) {
	fw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	return &Watcher{
		w:        fw,
		Events:   make(chan FileEvent, 32),
		maxDepth: DefaultMaxDepth,
	}, nil
}

func (w *Watcher) Start(dir string) error {
	w.running = true
	w.rootDir = dir

	// Recursively walk and add subdirectories up to maxDepth
	if err := w.addRecursive(dir, 0); err != nil {
		return err
	}

	go w.loop()
	return nil
}

// addRecursive walks the directory tree and adds watches up to the depth limit.
func (w *Watcher) addRecursive(dir string, depth int) error {
	if w.dirCount >= MaxWatchedDirs {
		slog.Warn("max watched directories reached, skipping further additions",
			"limit", MaxWatchedDirs, "dir", dir)
		return nil
	}

	if err := w.w.Add(dir); err != nil {
		return fmt.Errorf("watching directory %s: %w", dir, err)
	}
	w.dirCount++

	// Walk subdirectories with depth control
	return filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // skip directories we can't read
		}
		if path == dir {
			return nil // already added above
		}
		if !d.IsDir() {
			return nil // only interested in directories
		}

		// Check exclusion list
		name := d.Name()
		if excludedDirs[name] {
			return fs.SkipDir
		}

		// Check depth relative to root
		rel, relErr := filepath.Rel(w.rootDir, path)
		if relErr != nil {
			return fs.SkipDir
		}
		currentDepth := len(strings.Split(rel, string(filepath.Separator)))
		if currentDepth > w.maxDepth {
			return fs.SkipDir
		}

		// Check directory count cap
		if w.dirCount >= MaxWatchedDirs {
			slog.Warn("max watched directories reached",
				"limit", MaxWatchedDirs, "skipped", path)
			return fs.SkipDir
		}

		if err := w.w.Add(path); err != nil {
			slog.Warn("failed to watch directory", "path", path, "error", err)
			return nil // don't fail the whole walk
		}
		w.dirCount++
		return nil
	})
}

// loop processes fsnotify events and emits FileEvents.
func (w *Watcher) loop() {
	for {
		select {
		case event, ok := <-w.w.Events:
			if !ok {
				return
			}

			// Handle new directory creation — add to watch if within depth
			if event.Op&fsnotify.Create != 0 {
				w.handleCreate(event.Name)
			}

			// Handle directory removal — remove from watch
			if event.Op&fsnotify.Remove != 0 {
				w.handleRemove(event.Name)
			}

			if event.Op&(fsnotify.Write|fsnotify.Create) != 0 {
				// Use relative path from root dir
				relPath := w.relativePath(event.Name)
				fe := FileEvent{
					Path: relPath,
					Time: time.Now(),
				}
				w.mu.Lock()
				w.recent = append(w.recent, fe)
				if len(w.recent) > 10 {
					w.recent = w.recent[len(w.recent)-10:]
				}
				w.mu.Unlock()
				select {
				case w.Events <- fe:
				default:
				}
			}
		case _, ok := <-w.w.Errors:
			if !ok {
				return
			}
		}
	}
}

// handleCreate checks if a new path is a directory and adds it to the watcher.
func (w *Watcher) handleCreate(path string) {
	// Check if it's a directory (best-effort, may race with removal)
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		return
	}

	// Check if excluded
	name := filepath.Base(path)
	if excludedDirs[name] {
		return
	}

	// Check depth
	rel, err := filepath.Rel(w.rootDir, path)
	if err != nil {
		return
	}
	depth := len(strings.Split(rel, string(filepath.Separator)))
	if depth > w.maxDepth {
		return
	}

	// Check cap
	w.mu.Lock()
	if w.dirCount >= MaxWatchedDirs {
		w.mu.Unlock()
		slog.Warn("max watched directories reached, cannot add new directory",
			"limit", MaxWatchedDirs, "dir", path)
		return
	}
	w.mu.Unlock()

	if err := w.w.Add(path); err != nil {
		slog.Warn("failed to watch new directory", "path", path, "error", err)
		return
	}
	w.mu.Lock()
	w.dirCount++
	w.mu.Unlock()
}

// handleRemove attempts to remove a path from the watcher.
func (w *Watcher) handleRemove(path string) {
	// fsnotify auto-removes watches for deleted paths on most systems,
	// but we track the count for correctness.
	_ = w.w.Remove(path)
	w.mu.Lock()
	if w.dirCount > 0 {
		w.dirCount--
	}
	w.mu.Unlock()
}

// relativePath returns the path relative to rootDir, or the basename as fallback.
func (w *Watcher) relativePath(path string) string {
	if w.rootDir == "" {
		return filepath.Base(path)
	}
	rel, err := filepath.Rel(w.rootDir, path)
	if err != nil {
		return filepath.Base(path)
	}
	return rel
}

// Watch adds an additional directory to the watcher (explicit root addition).
func (w *Watcher) Watch(dir string) error {
	if err := w.w.Add(dir); err != nil {
		return fmt.Errorf("watching directory %s: %w", dir, err)
	}
	return nil
}

func (w *Watcher) Recent() []FileEvent {
	w.mu.Lock()
	defer w.mu.Unlock()
	cp := make([]FileEvent, len(w.recent))
	copy(cp, w.recent)
	return cp
}

// DirCount returns the current number of watched directories (thread-safe).
func (w *Watcher) DirCount() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.dirCount
}

// SetDirCount sets the dirCount (for testing only).
func (w *Watcher) SetDirCount(n int) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.dirCount = n
}

func (w *Watcher) Running() bool {
	return w.running
}

func (w *Watcher) Stop() {
	w.running = false
	w.w.Close()
}
