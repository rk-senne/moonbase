package tui

import (
	"strconv"
	"sync"

	"charm.land/glamour/v2"
)

// Markdown rendering caches. glamour.NewTermRenderer is expensive, and the
// pipeline view previously created a fresh renderer AND re-rendered every agent
// message on every frame (each spinner/anim/clock tick and stream chunk). That
// was the dominant source of TUI lag. We now:
//   1. reuse one glamour renderer per wrap width, and
//   2. memoize rendered output keyed by (width, content).
//
// Completed chat messages are immutable, so their render is computed once and
// reused for the life of the process. Only the single in-flight streaming
// message misses the cache as it grows — one render per frame instead of N.
var (
	mdMu        sync.Mutex
	mdRenderers = map[int]*glamour.TermRenderer{}
	mdCache     = map[string]string{}
	mdCacheKeys []string // insertion order for bounded FIFO eviction
)

// mdCacheMax bounds memoized entries so a long-running session with lots of
// streaming partials cannot grow the cache without limit.
const mdCacheMax = 512

// renderMarkdown renders markdown text using glamour with dark styling and word
// wrap. Results are memoized per (width, content). Falls back to raw text if
// rendering fails. Safe for concurrent use.
func renderMarkdown(md string, width int) string {
	if width <= 0 {
		width = 80
	}

	mdMu.Lock()
	defer mdMu.Unlock()

	key := strconv.Itoa(width) + "\x00" + md
	if cached, ok := mdCache[key]; ok {
		return cached
	}

	r := mdRenderers[width]
	if r == nil {
		nr, err := glamour.NewTermRenderer(
			glamour.WithStandardStyle("dark"),
			glamour.WithWordWrap(width),
		)
		if err != nil {
			return md
		}
		mdRenderers[width] = nr
		r = nr
	}

	rendered, err := r.Render(md)
	if err != nil {
		return md
	}

	// Bounded FIFO eviction keeps the working set (visible messages + recent
	// streaming partials) hot without unbounded growth.
	if len(mdCacheKeys) >= mdCacheMax {
		oldest := mdCacheKeys[0]
		mdCacheKeys = mdCacheKeys[1:]
		delete(mdCache, oldest)
	}
	mdCache[key] = rendered
	mdCacheKeys = append(mdCacheKeys, key)

	return rendered
}
