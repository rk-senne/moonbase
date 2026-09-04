package discovery

import (
	"go/parser"
	"go/token"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

// discoverDepGraph builds a file-level import graph for Go projects. It returns
// nil for non-Go projects, when go.mod's module directive cannot be read, or
// when the graph build fails — keeping Discover infallible and leaving all
// existing behaviour unchanged when the graph is unavailable.
func discoverDepGraph(projectDir string, stack StackInfo) *DepGraph {
	if stack.Language != "go" {
		return nil
	}

	modulePath := readModulePath(projectDir)
	if modulePath == "" {
		return nil
	}

	g, err := BuildGoDepGraph(projectDir, modulePath)
	if err != nil {
		slog.Debug("depgraph: build failed, skipping", "err", err)
		return nil
	}
	return g
}

// readModulePath extracts the module path from go.mod's `module` directive.
// Returns "" if go.mod is missing or has no module line.
func readModulePath(projectDir string) string {
	data, err := os.ReadFile(filepath.Join(projectDir, "go.mod"))
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if rest, ok := strings.CutPrefix(trimmed, "module"); ok {
			// Require a space/tab after "module" to avoid matching "modulefoo".
			if len(rest) > 0 && (rest[0] == ' ' || rest[0] == '\t') {
				return strings.TrimSpace(rest)
			}
		}
	}
	return ""
}

// BuildGoDepGraph builds a file-level import graph for the Go module rooted at
// rootDir with import prefix modulePath. It walks .go files (skipping vendor,
// .git, and hidden directories), parses each in ImportsOnly mode, and records
// file→file edges for imports whose path is under modulePath. Files that fail
// to parse are skipped (logged at debug); stdlib and third-party imports are
// ignored. Returns ErrDepGraphRoot only when rootDir itself cannot be walked.
func BuildGoDepGraph(rootDir, modulePath string) (*DepGraph, error) {
	if _, err := os.Stat(rootDir); err != nil {
		return nil, ErrDepGraphRoot
	}

	fset := token.NewFileSet()

	// Pass 1: collect, per parseable file, its slash-relative path, the
	// directory it lives in, and its intra-project import package paths.
	type fileInfo struct {
		rel     string
		dir     string   // slash-relative dir (the file's package dir)
		imports []string // intra-project import paths (module-prefixed)
	}
	var files []fileInfo
	// dirFiles maps a slash-relative dir to the files it contains.
	dirFiles := make(map[string][]string)
	prefix := modulePath + "/"

	walkErr := filepath.WalkDir(rootDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			// Unreadable entry: skip it, keep walking.
			return nil //nolint:nilerr // tolerate unreadable dirs per infallibility contract
		}
		if d.IsDir() {
			name := d.Name()
			if path == rootDir {
				return nil
			}
			if name == "vendor" || name == ".git" || (strings.HasPrefix(name, ".") && name != ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(d.Name(), ".go") {
			return nil
		}

		rel, relErr := filepath.Rel(rootDir, path)
		if relErr != nil {
			return nil
		}
		relSlash := filepath.ToSlash(rel)

		src, readErr := os.ReadFile(path)
		if readErr != nil {
			slog.Debug("depgraph: skipping unreadable file", "path", relSlash, "err", readErr)
			return nil
		}

		astFile, parseErr := parser.ParseFile(fset, path, src, parser.ImportsOnly)
		if parseErr != nil {
			slog.Debug("depgraph: skipping unparseable file", "path", relSlash, "err", parseErr)
			return nil
		}

		dir := filepath.ToSlash(filepath.Dir(rel))
		var imps []string
		for _, imp := range astFile.Imports {
			if imp.Path == nil {
				continue
			}
			p := strings.Trim(imp.Path.Value, `"`)
			if p == modulePath || strings.HasPrefix(p, prefix) {
				imps = append(imps, p)
			}
		}

		files = append(files, fileInfo{rel: relSlash, dir: dir, imports: imps})
		dirFiles[dir] = append(dirFiles[dir], relSlash)
		return nil
	})
	if walkErr != nil {
		return nil, ErrDepGraphRoot
	}

	g := NewDepGraph()

	// Every parseable file is a node, even with no edges.
	for _, fi := range files {
		g.addNode(fi.rel)
	}

	// Pass 2: resolve each file's intra-project imports to the files in the
	// target package directory and record forward + reverse edges.
	for _, fi := range files {
		for _, imp := range fi.imports {
			targetDir := importPathToDir(imp, modulePath)
			for _, targetFile := range dirFiles[targetDir] {
				g.addEdge(fi.rel, targetFile)
			}
		}
	}

	return g, nil
}

// importPathToDir converts an intra-project import path to its slash-relative
// directory. "modulePath/pkg/foo" → "pkg/foo"; "modulePath" (the root package)
// → ".".
func importPathToDir(importPath, modulePath string) string {
	if importPath == modulePath {
		return "."
	}
	return strings.TrimPrefix(importPath, modulePath+"/")
}
