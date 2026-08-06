package reboot

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeGoMod(t *testing.T, dir, module string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module "+module+"\n\ngo 1.26\n"), 0o600); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
}

func TestIsMoonbaseRepo(t *testing.T) {
	dir := t.TempDir()
	if isMoonbaseRepo(dir) {
		t.Error("empty dir should not be a moonbase repo")
	}
	writeGoMod(t, dir, "github.com/rk-senne/moonbase")
	if !isMoonbaseRepo(dir) {
		t.Error("expected moonbase repo detected via go.mod")
	}

	other := t.TempDir()
	writeGoMod(t, other, "example.com/other")
	if isMoonbaseRepo(other) {
		t.Error("non-moonbase module must not be detected")
	}
}

func TestWalkUpToRepo(t *testing.T) {
	repo := t.TempDir()
	writeGoMod(t, repo, "github.com/rk-senne/moonbase")
	nested := filepath.Join(repo, "bin")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	exe := filepath.Join(nested, "moonbase")
	if err := os.WriteFile(exe, []byte("x"), 0o755); err != nil {
		t.Fatal(err)
	}

	got, ok := walkUpToRepo(exe)
	if !ok || got != repo {
		t.Errorf("walkUpToRepo(%q) = %q,%v; want %q,true", exe, got, ok, repo)
	}

	if _, ok := walkUpToRepo(t.TempDir()); ok {
		t.Error("expected no repo found under a bare temp dir")
	}
}

func TestFindSourceDir_ConfigWins(t *testing.T) {
	repo := t.TempDir()
	writeGoMod(t, repo, "github.com/rk-senne/moonbase")
	got, ok := FindSourceDir(repo, "")
	if !ok || got != repo {
		t.Errorf("expected config dir %q used, got %q,%v", repo, got, ok)
	}
}

func TestFindSourceDir_ViaExeSymlinkWalkUp(t *testing.T) {
	repo := t.TempDir()
	writeGoMod(t, repo, "github.com/rk-senne/moonbase")
	bin := filepath.Join(repo, "bin")
	os.MkdirAll(bin, 0o755)
	exe := filepath.Join(bin, "moonbase")
	os.WriteFile(exe, []byte("x"), 0o755)

	t.Setenv("MOONBASE_SRC", "") // ensure env doesn't interfere
	got, ok := FindSourceDir("", exe)
	// EvalSymlinks canonicalizes paths (on macOS /var → /private/var), so compare
	// against the resolved repo path.
	wantRepo, _ := filepath.EvalSymlinks(repo)
	if !ok || got != wantRepo {
		t.Errorf("expected repo via exe walk-up %q, got %q,%v", wantRepo, got, ok)
	}
}

func TestSelectPlan(t *testing.T) {
	repo := t.TempDir()
	writeGoMod(t, repo, "github.com/rk-senne/moonbase")
	t.Setenv("MOONBASE_SRC", "")

	// Source available → source strategy regardless of version.
	if p := SelectPlan("dev", repo, "/usr/local/bin/moonbase"); p.Strategy != StrategySource || p.SourceDir != repo {
		t.Errorf("expected StrategySource with sourcedir, got %+v", p)
	}

	// No source + release version → release strategy.
	p := SelectPlan("1.2.3", "", filepath.Join(t.TempDir(), "moonbase"))
	if p.Strategy != StrategyRelease {
		t.Errorf("expected StrategyRelease for a release build w/o source, got %+v", p)
	}

	// No source + dev build → none (guide the user).
	p = SelectPlan("dev", "", filepath.Join(t.TempDir(), "moonbase"))
	if p.Strategy != StrategyNone {
		t.Errorf("expected StrategyNone for dev build w/o source, got %+v", p)
	}
	if p.Reason == "" {
		t.Error("expected a reason explaining the None strategy")
	}
}

func TestReinstallScript(t *testing.T) {
	s := ReinstallScript("/home/op/moonbase", "/home/op/.local/bin/moonbase")
	for _, want := range []string{"git pull --ff-only", "go build -o bin/moonbase ./cmd/moonbase", "cp bin/moonbase", "'/home/op/moonbase'", "'/home/op/.local/bin/moonbase'"} {
		if !strings.Contains(s, want) {
			t.Errorf("reinstall script missing %q:\n%s", want, s)
		}
	}
}

func TestShellSingleQuote_EscapesQuotes(t *testing.T) {
	got := shellSingleQuote("a'b")
	if got != `'a'\''b'` {
		t.Errorf("shellSingleQuote(a'b) = %q", got)
	}
}
