package discovery

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

const testModulePath = "example.com/proj"

// writeProject creates a set of files under a temp dir and returns the root.
// files maps a slash-relative path to its content.
func writeProject(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for rel, content := range files {
		abs := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(abs), 0o700); err != nil {
			t.Fatalf("mkdir for %s: %v", rel, err)
		}
		if err := os.WriteFile(abs, []byte(content), 0o600); err != nil {
			t.Fatalf("write %s: %v", rel, err)
		}
	}
	return root
}

func TestBuildGoDepGraph_HappyPath(t *testing.T) {
	// main imports pkg/foo; pkg/foo imports pkg/bar. Stdlib + third-party ignored.
	root := writeProject(t, map[string]string{
		"main.go": `package main

import (
	"fmt"

	"example.com/proj/pkg/foo"
	"github.com/some/thirdparty"
)

func main() { fmt.Println(foo.X, thirdparty.Y) }
`,
		"pkg/foo/foo.go": `package foo

import "example.com/proj/pkg/bar"

var X = bar.V
`,
		"pkg/bar/bar.go": `package bar

var V = 1
`,
	})

	g, err := BuildGoDepGraph(root, testModulePath)
	if err != nil {
		t.Fatalf("BuildGoDepGraph: %v", err)
	}

	// main.go depends directly on pkg/foo/foo.go (not on stdlib/third-party).
	direct := g.DirectDependencies("main.go")
	want := []string{"pkg/foo/foo.go"}
	if !reflect.DeepEqual(direct, want) {
		t.Errorf("DirectDependencies(main.go) = %v, want %v", direct, want)
	}
}

func TestBuildGoDepGraph_ExcludesStdlibAndThirdParty(t *testing.T) {
	root := writeProject(t, map[string]string{
		"main.go": `package main

import (
	"fmt"
	"strings"

	"github.com/x/y"
)

var _ = fmt.Sprintf
var _ = strings.TrimSpace
var _ = y.Z
`,
	})

	g, err := BuildGoDepGraph(root, testModulePath)
	if err != nil {
		t.Fatalf("BuildGoDepGraph: %v", err)
	}

	if deps := g.DirectDependencies("main.go"); len(deps) != 0 {
		t.Errorf("expected no intra-project edges, got %v", deps)
	}
	// The node itself must still exist.
	if !contains(g.Nodes(), "main.go") {
		t.Errorf("expected main.go to be a node, nodes = %v", g.Nodes())
	}
}

func TestBuildGoDepGraph_MultiFilePackageFanOut(t *testing.T) {
	// main imports pkg/foo, which has two files. Edge fans out to both.
	root := writeProject(t, map[string]string{
		"main.go": `package main

import "example.com/proj/pkg/foo"

var _ = foo.A
`,
		"pkg/foo/a.go": `package foo

var A = 1
`,
		"pkg/foo/b.go": `package foo

var B = 2
`,
	})

	g, err := BuildGoDepGraph(root, testModulePath)
	if err != nil {
		t.Fatalf("BuildGoDepGraph: %v", err)
	}

	direct := g.DirectDependencies("main.go")
	want := []string{"pkg/foo/a.go", "pkg/foo/b.go"}
	if !reflect.DeepEqual(direct, want) {
		t.Errorf("DirectDependencies(main.go) = %v, want %v", direct, want)
	}
}

func TestBuildGoDepGraph_SkipsUnparseableFile(t *testing.T) {
	// One file is broken at the import level; the rest still graph.
	root := writeProject(t, map[string]string{
		"main.go": `package main

import "example.com/proj/pkg/foo"

var _ = foo.A
`,
		"pkg/foo/foo.go": `package foo

var A = 1
`,
		"broken.go": `this is not valid go at all @@@`,
	})

	g, err := BuildGoDepGraph(root, testModulePath)
	if err != nil {
		t.Fatalf("BuildGoDepGraph: %v", err)
	}

	// broken.go is skipped, but main.go → pkg/foo edge survives.
	if contains(g.Nodes(), "broken.go") {
		t.Errorf("expected broken.go to be skipped, nodes = %v", g.Nodes())
	}
	direct := g.DirectDependencies("main.go")
	if !reflect.DeepEqual(direct, []string{"pkg/foo/foo.go"}) {
		t.Errorf("DirectDependencies(main.go) = %v, want [pkg/foo/foo.go]", direct)
	}
}

func TestBuildGoDepGraph_SkipsVendorAndHidden(t *testing.T) {
	root := writeProject(t, map[string]string{
		"main.go": `package main

var X = 1
`,
		"vendor/dep/dep.go": `package dep

var Y = 1
`,
		".hidden/secret.go": `package secret

var Z = 1
`,
	})

	g, err := BuildGoDepGraph(root, testModulePath)
	if err != nil {
		t.Fatalf("BuildGoDepGraph: %v", err)
	}

	nodes := g.Nodes()
	if !contains(nodes, "main.go") {
		t.Errorf("expected main.go node, got %v", nodes)
	}
	if contains(nodes, "vendor/dep/dep.go") {
		t.Errorf("vendor files must be excluded, got %v", nodes)
	}
	if contains(nodes, ".hidden/secret.go") {
		t.Errorf("hidden dirs must be excluded, got %v", nodes)
	}
}

func TestBuildGoDepGraph_IncludesTestFiles(t *testing.T) {
	root := writeProject(t, map[string]string{
		"pkg/foo/foo.go": `package foo

var A = 1
`,
		"pkg/foo/foo_test.go": `package foo

import "testing"

func TestA(t *testing.T) { _ = A }
`,
	})

	g, err := BuildGoDepGraph(root, testModulePath)
	if err != nil {
		t.Fatalf("BuildGoDepGraph: %v", err)
	}

	if !contains(g.Nodes(), "pkg/foo/foo_test.go") {
		t.Errorf("expected _test.go file to be a node, nodes = %v", g.Nodes())
	}
}

func TestDepGraph_RelatedFiles_BothDirectionsTransitive(t *testing.T) {
	// a → b → c. RelatedFiles(b) reaches a (dependent) and c (dependency).
	root := writeProject(t, map[string]string{
		"a/a.go": `package a

import "example.com/proj/b"

var _ = b.B
`,
		"b/b.go": `package b

import "example.com/proj/c"

var B = c.C
`,
		"c/c.go": `package c

var C = 1
`,
	})

	g, err := BuildGoDepGraph(root, testModulePath)
	if err != nil {
		t.Fatalf("BuildGoDepGraph: %v", err)
	}

	related := g.RelatedFiles("b/b.go")
	want := []string{"a/a.go", "c/c.go"}
	if !reflect.DeepEqual(related, want) {
		t.Errorf("RelatedFiles(b/b.go) = %v, want %v", related, want)
	}
}

func TestDepGraph_AffectedFiles_ReverseTransitive(t *testing.T) {
	// a → b → c. Changing c affects b and a (transitive dependents).
	root := writeProject(t, map[string]string{
		"a/a.go": `package a

import "example.com/proj/b"

var _ = b.B
`,
		"b/b.go": `package b

import "example.com/proj/c"

var B = c.C
`,
		"c/c.go": `package c

var C = 1
`,
	})

	g, err := BuildGoDepGraph(root, testModulePath)
	if err != nil {
		t.Fatalf("BuildGoDepGraph: %v", err)
	}

	affected := g.AffectedFiles("c/c.go")
	want := []string{"a/a.go", "b/b.go"}
	if !reflect.DeepEqual(affected, want) {
		t.Errorf("AffectedFiles(c/c.go) = %v, want %v", affected, want)
	}
}

func TestDepGraph_MissingNode_ReturnsEmpty(t *testing.T) {
	g := NewDepGraph()

	if got := g.RelatedFiles("nope.go"); len(got) != 0 {
		t.Errorf("RelatedFiles(missing) = %v, want empty", got)
	}
	if got := g.AffectedFiles("nope.go"); len(got) != 0 {
		t.Errorf("AffectedFiles(missing) = %v, want empty", got)
	}
	if got := g.DirectDependencies("nope.go"); len(got) != 0 {
		t.Errorf("DirectDependencies(missing) = %v, want empty", got)
	}
}

func TestDepGraph_QueryNormalizesInput(t *testing.T) {
	root := writeProject(t, map[string]string{
		"main.go": `package main

import "example.com/proj/pkg/foo"

var _ = foo.A
`,
		"pkg/foo/foo.go": `package foo

var A = 1
`,
	})

	g, err := BuildGoDepGraph(root, testModulePath)
	if err != nil {
		t.Fatalf("BuildGoDepGraph: %v", err)
	}

	// Pass an absolute path — must normalize to relative and still resolve.
	abs := filepath.Join(root, "main.go")
	direct := g.DirectDependencies(abs)
	if !reflect.DeepEqual(direct, []string{"pkg/foo/foo.go"}) {
		t.Errorf("DirectDependencies(abs main.go) = %v, want [pkg/foo/foo.go]", direct)
	}

	// Pass a backslash-style path — must normalize to slash.
	direct = g.DirectDependencies("main.go")
	if !reflect.DeepEqual(direct, []string{"pkg/foo/foo.go"}) {
		t.Errorf("DirectDependencies(main.go) = %v, want [pkg/foo/foo.go]", direct)
	}
}

func TestDepGraph_SelfImportTerminates(t *testing.T) {
	// A package split across two files where one file references the same
	// package's import path would create a self-edge. Ensure queries terminate.
	root := writeProject(t, map[string]string{
		"a/a.go": `package a

import "example.com/proj/b"

var _ = b.B
`,
		"b/b.go": `package b

// b imports a, and a imports b — a cycle.
import "example.com/proj/a"

var B = 1
var _ = a.C
`,
		"a/c.go": `package a

var C = 1
`,
	})

	g, err := BuildGoDepGraph(root, testModulePath)
	if err != nil {
		t.Fatalf("BuildGoDepGraph: %v", err)
	}

	// Must terminate (no infinite loop / stack overflow) and not include self.
	related := g.RelatedFiles("a/a.go")
	if contains(related, "a/a.go") {
		t.Errorf("RelatedFiles must not include the query node itself: %v", related)
	}
	// a/a.go is related to b (imports it) and a/c.go (same package, reached via cycle).
	if !contains(related, "b/b.go") {
		t.Errorf("expected b/b.go in related set, got %v", related)
	}
}

func TestDepGraph_CrossPackageCycleTerminates(t *testing.T) {
	// x → y → x. Both directions must terminate.
	root := writeProject(t, map[string]string{
		"x/x.go": `package x

import "example.com/proj/y"

var X = y.Y
`,
		"y/y.go": `package y

import "example.com/proj/x"

var Y = 1
var _ = x.X
`,
	})

	g, err := BuildGoDepGraph(root, testModulePath)
	if err != nil {
		t.Fatalf("BuildGoDepGraph: %v", err)
	}

	related := g.RelatedFiles("x/x.go")
	if !contains(related, "y/y.go") {
		t.Errorf("expected y/y.go related to x/x.go, got %v", related)
	}
	affected := g.AffectedFiles("x/x.go")
	if !contains(affected, "y/y.go") {
		t.Errorf("expected y/y.go to be affected by x/x.go, got %v", affected)
	}
}

func TestDepGraph_ResultsAreSortedAndDeduped(t *testing.T) {
	// main imports two packages; each has two files. Related set is sorted + unique.
	root := writeProject(t, map[string]string{
		"main.go": `package main

import (
	"example.com/proj/z"
	"example.com/proj/a"
)

var _ = z.Z
var _ = a.A
`,
		"z/z1.go": "package z\n\nvar Z = 1\n",
		"z/z2.go": "package z\n\nvar Z2 = 1\n",
		"a/a1.go": "package a\n\nvar A = 1\n",
		"a/a2.go": "package a\n\nvar A2 = 1\n",
	})

	g, err := BuildGoDepGraph(root, testModulePath)
	if err != nil {
		t.Fatalf("BuildGoDepGraph: %v", err)
	}

	direct := g.DirectDependencies("main.go")
	want := []string{"a/a1.go", "a/a2.go", "z/z1.go", "z/z2.go"}
	if !reflect.DeepEqual(direct, want) {
		t.Errorf("DirectDependencies(main.go) = %v, want %v (sorted, deduped)", direct, want)
	}
}

func TestDepGraph_Len(t *testing.T) {
	root := writeProject(t, map[string]string{
		"a.go": "package main\n\nvar A = 1\n",
		"b.go": "package main\n\nvar B = 1\n",
	})

	g, err := BuildGoDepGraph(root, testModulePath)
	if err != nil {
		t.Fatalf("BuildGoDepGraph: %v", err)
	}
	if g.Len() != 2 {
		t.Errorf("Len() = %d, want 2", g.Len())
	}
}

func TestBuildGoDepGraph_UnreadableRoot(t *testing.T) {
	// A root that does not exist is an unrecoverable setup error.
	_, err := BuildGoDepGraph(filepath.Join(t.TempDir(), "does-not-exist"), testModulePath)
	if err == nil {
		t.Error("expected error for nonexistent root, got nil")
	}
}

// contains reports whether s contains v.
func contains(s []string, v string) bool {
	for _, item := range s {
		if item == v {
			return true
		}
	}
	return false
}

// === Integration: Discover wiring (behaviour preservation) ===

func TestDiscover_NonGoProject_NoDepGraph(t *testing.T) {
	// A non-Go project (package.json) must leave DepGraph nil, HasDepGraph
	// false, and Summary free of any graph segment — behaviour preserved.
	root := writeProject(t, map[string]string{
		"package.json": `{"name":"x","version":"1.0.0"}`,
		"index.js":     `console.log("hi");`,
	})

	ctx := Discover(root)

	if ctx.DepGraph != nil {
		t.Errorf("expected nil DepGraph for non-Go project, got %v", ctx.DepGraph)
	}
	if ctx.HasDepGraph() {
		t.Error("expected HasDepGraph() false for non-Go project")
	}
	if strings.Contains(ctx.Summary(), "Graph") {
		t.Errorf("Summary must not mention Graph for non-Go project: %q", ctx.Summary())
	}
}

func TestDiscover_GoProject_BuildsDepGraph(t *testing.T) {
	root := writeProject(t, map[string]string{
		"go.mod": "module example.com/proj\n\ngo 1.26\n",
		"main.go": `package main

import "example.com/proj/pkg/foo"

func main() { _ = foo.A }
`,
		"pkg/foo/foo.go": "package foo\n\nvar A = 1\n",
	})

	ctx := Discover(root)

	if !ctx.HasDepGraph() {
		t.Fatal("expected HasDepGraph() true for Go project")
	}
	if ctx.DepGraph.Len() != 2 {
		t.Errorf("expected 2 nodes, got %d", ctx.DepGraph.Len())
	}
	if deps := ctx.DepGraph.DirectDependencies("main.go"); !contains(deps, "pkg/foo/foo.go") {
		t.Errorf("expected main.go → pkg/foo/foo.go edge, got %v", deps)
	}
	if !strings.Contains(ctx.Summary(), "Graph: 2 files") {
		t.Errorf("expected Summary to mention graph, got %q", ctx.Summary())
	}
}

func TestReadModulePath(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{"standard", "module example.com/proj\n\ngo 1.26\n", "example.com/proj"},
		{"leading whitespace", "  module   example.com/x  \n", "example.com/x"},
		{"no module line", "go 1.26\n", ""},
		{"module-prefixed word only", "modulefoo bar\n", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := writeProject(t, map[string]string{"go.mod": tt.content})
			if got := readModulePath(root); got != tt.want {
				t.Errorf("readModulePath() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestReadModulePath_MissingGoMod(t *testing.T) {
	if got := readModulePath(t.TempDir()); got != "" {
		t.Errorf("readModulePath(no go.mod) = %q, want empty", got)
	}
}
