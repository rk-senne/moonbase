package discovery

import (
	"errors"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// ErrDepGraphRoot is returned by BuildGoDepGraph when the root directory cannot
// be walked (an unrecoverable setup error, distinct from a graph that is simply
// empty because no intra-project edges exist).
var ErrDepGraphRoot = errors.New("dependency graph root unreadable")

// DepGraph is a file-level import dependency graph for a single Go module.
//
// Nodes are file paths relative to the module root, normalized to forward
// slashes (e.g. "internal/discovery/discovery.go") so they are stable and
// comparable regardless of the absolute install location. Edges point from a
// file to every file in each intra-project package it imports; because Go
// imports resolve at package granularity, an edge to a package fans out to all
// of that package's files.
//
// Two directions are tracked: forward (file → its dependencies) and reverse
// (file → its dependents). Query methods use visited-set BFS, so cycles and
// self-imports terminate safely.
//
// Known v1 limitations (documented, accepted):
//   - File→package fan-out is coarser than symbol-level: an import of a package
//     links to all files in it, not just the file defining the used symbol.
//   - Build-tag constraints are not evaluated; imports are collected regardless
//     of build tags.
//   - Only a single, non-vendored module is graphed.
//
// Safe for concurrent read.
type DepGraph struct {
	mu sync.RWMutex
	// forward[a] holds the set of files a depends on.
	forward map[string]map[string]struct{}
	// reverse[b] holds the set of files that depend on b.
	reverse map[string]map[string]struct{}
	// nodes is the set of all known file nodes.
	nodes map[string]struct{}
}

// NewDepGraph creates an empty dependency graph.
func NewDepGraph() *DepGraph {
	return &DepGraph{
		forward: make(map[string]map[string]struct{}),
		reverse: make(map[string]map[string]struct{}),
		nodes:   make(map[string]struct{}),
	}
}

// HasDepGraph returns true if a non-empty dependency graph was built.
func (pc *ProjectContext) HasDepGraph() bool {
	return pc.DepGraph != nil && pc.DepGraph.Len() > 0
}

// addNode records a file as a node without adding any edge.
func (g *DepGraph) addNode(file string) {
	g.nodes[file] = struct{}{}
}

// addEdge records a forward edge from → to and its reverse.
// Self-edges are ignored (a file does not depend on itself).
func (g *DepGraph) addEdge(from, to string) {
	if from == to {
		return
	}
	g.addNode(from)
	g.addNode(to)

	if g.forward[from] == nil {
		g.forward[from] = make(map[string]struct{})
	}
	g.forward[from][to] = struct{}{}

	if g.reverse[to] == nil {
		g.reverse[to] = make(map[string]struct{})
	}
	g.reverse[to][from] = struct{}{}
}

// normalizeNode converts a caller-supplied path to slash form. Resolution of
// absolute or prefixed paths to a known relative node key is handled by
// resolveLocked; this only fixes the separator.
func (g *DepGraph) normalizeNode(path string) string {
	return filepath.ToSlash(path)
}

// Nodes returns all file nodes in sorted order.
func (g *DepGraph) Nodes() []string {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return sortedKeys(g.nodes)
}

// Len returns the number of file nodes in the graph.
func (g *DepGraph) Len() int {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return len(g.nodes)
}

// DirectDependencies returns the immediate forward neighbours of path (the
// files it directly imports), sorted and deduplicated. Returns an empty slice
// if the node is unknown. Never panics.
func (g *DepGraph) DirectDependencies(path string) []string {
	g.mu.RLock()
	defer g.mu.RUnlock()

	node := g.resolveLocked(path)
	if node == "" {
		return nil
	}
	return sortedKeys(g.forward[node])
}

// RelatedFiles returns the transitive closure of path in both directions
// (everything it depends on, plus everything that depends on it), excluding
// path itself. Cycle-safe, sorted, deduplicated. Empty if the node is unknown.
func (g *DepGraph) RelatedFiles(path string) []string {
	g.mu.RLock()
	defer g.mu.RUnlock()

	node := g.resolveLocked(path)
	if node == "" {
		return nil
	}

	visited := make(map[string]struct{})
	g.bfsLocked(node, g.forward, visited)
	g.bfsLocked(node, g.reverse, visited)

	delete(visited, node)
	return sortedKeys(visited)
}

// AffectedFiles returns the reverse transitive closure of path (every file that
// transitively depends on it), excluding path itself. This answers "if I change
// this file, what could be affected?". Cycle-safe, sorted, deduplicated. Empty
// if the node is unknown.
func (g *DepGraph) AffectedFiles(path string) []string {
	g.mu.RLock()
	defer g.mu.RUnlock()

	node := g.resolveLocked(path)
	if node == "" {
		return nil
	}

	visited := make(map[string]struct{})
	g.bfsLocked(node, g.reverse, visited)

	delete(visited, node)
	return sortedKeys(visited)
}

// resolveLocked maps a caller-supplied path to a known node key. It tries the
// normalized path directly, then a suffix match against known nodes (so an
// absolute path whose tail matches a relative node key resolves). Returns "" if
// no node matches. Caller must hold at least the read lock.
func (g *DepGraph) resolveLocked(path string) string {
	norm := g.normalizeNode(path)
	if _, ok := g.nodes[norm]; ok {
		return norm
	}
	// Absolute or otherwise-prefixed path: match the longest known node that is
	// a path suffix of the input (bounded by a path separator).
	best := ""
	for node := range g.nodes {
		if strings.HasSuffix(norm, "/"+node) && len(node) > len(best) {
			best = node
		}
	}
	return best
}

// bfsLocked walks adj from start, marking every reachable node in visited.
// Iterative with a visited set, so cycles terminate. Caller holds the lock.
func (g *DepGraph) bfsLocked(start string, adj map[string]map[string]struct{}, visited map[string]struct{}) {
	queue := []string{start}
	visited[start] = struct{}{}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		for next := range adj[cur] {
			if _, seen := visited[next]; seen {
				continue
			}
			visited[next] = struct{}{}
			queue = append(queue, next)
		}
	}
}

// sortedKeys returns the keys of a set as a sorted slice. Nil-safe.
func sortedKeys(set map[string]struct{}) []string {
	if len(set) == 0 {
		return nil
	}
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
