package watcher

import (
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

type FileEvent struct {
	Path string
	Time time.Time
}

type Watcher struct {
	w       *fsnotify.Watcher
	Events  chan FileEvent
	recent  []FileEvent
	mu      sync.Mutex
	running bool
}

func New() (*Watcher, error) {
	fw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	return &Watcher{
		w:      fw,
		Events: make(chan FileEvent, 32),
	}, nil
}

func (w *Watcher) Start(dir string) error {
	w.running = true
	// Watch the directory (non-recursive for performance)
	if err := w.w.Add(dir); err != nil {
		return err
	}

	go func() {
		for {
			select {
			case event, ok := <-w.w.Events:
				if !ok {
					return
				}
				if event.Op&(fsnotify.Write|fsnotify.Create) != 0 {
					fe := FileEvent{
						Path: filepath.Base(event.Name),
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
	}()
	return nil
}

func (w *Watcher) Recent() []FileEvent {
	w.mu.Lock()
	defer w.mu.Unlock()
	cp := make([]FileEvent, len(w.recent))
	copy(cp, w.recent)
	return cp
}

func (w *Watcher) Running() bool {
	return w.running
}

func (w *Watcher) Stop() {
	w.running = false
	w.w.Close()
}
