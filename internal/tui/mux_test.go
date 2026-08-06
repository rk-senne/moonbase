package tui

import (
	"os/exec"
	"testing"
)

// fakeLookPath returns a lookPathFunc that reports the given tool names as
// installed (resolving to a fake path) and everything else as missing.
func fakeLookPath(installed ...string) lookPathFunc {
	set := make(map[string]bool, len(installed))
	for _, n := range installed {
		set[n] = true
	}
	return func(name string) (string, error) {
		if set[name] {
			return "/usr/bin/" + name, nil
		}
		return "", exec.ErrNotFound
	}
}

func TestSelectMultiplexer(t *testing.T) {
	tests := []struct {
		name     string
		goos     string
		look     lookPathFunc
		wantTool string
		wantOK   bool
	}{
		{"macOS prefers cmux when both present", "darwin", fakeLookPath("cmux", "tmux"), "cmux", true},
		{"macOS falls back to tmux", "darwin", fakeLookPath("tmux"), "tmux", true},
		{"macOS none installed", "darwin", fakeLookPath(), "", false},
		{"linux uses tmux even if cmux present", "linux", fakeLookPath("cmux", "tmux"), "tmux", true},
		{"linux tmux only", "linux", fakeLookPath("tmux"), "tmux", true},
		{"linux ignores lone cmux", "linux", fakeLookPath("cmux"), "", false},
		{"other OS uses tmux", "freebsd", fakeLookPath("tmux"), "tmux", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := selectMultiplexer(tt.goos, tt.look)
			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOK)
			}
			if !ok {
				return
			}
			if got.Tool != tt.wantTool {
				t.Errorf("tool = %q, want %q", got.Tool, tt.wantTool)
			}
			if got.Bin == "" {
				t.Error("expected a resolved binary path")
			}
			if got.Tool == "cmux" && len(got.Args) == 0 {
				t.Error("expected cmux launch args")
			}
		})
	}
}
