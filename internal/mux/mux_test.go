package mux

import (
	"os/exec"
	"strings"
	"testing"
)

func look(installed ...string) func(string) (string, error) {
	set := map[string]bool{}
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

func envWith(kv map[string]string) func(string) string {
	return func(k string) string { return kv[k] }
}

func TestDetect(t *testing.T) {
	tests := []struct {
		name     string
		goos     string
		look     func(string) (string, error)
		env      map[string]string
		wantKind Kind
		wantName string
	}{
		{"macOS prefers cmux", "darwin", look("cmux", "tmux"), nil, Cmux, "cmux"},
		{"macOS falls back to tmux", "darwin", look("tmux"), nil, Tmux, "tmux"},
		{"macOS none", "darwin", look(), nil, None, "none"},
		{"linux uses tmux (ignores cmux)", "linux", look("cmux", "tmux"), nil, Tmux, "tmux"},
		{"linux none", "linux", look("cmux"), nil, None, "none"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := detect(tt.goos, tt.look, envWith(tt.env))
			if m.Kind != tt.wantKind {
				t.Errorf("kind = %v, want %v", m.Kind, tt.wantKind)
			}
			if m.Name() != tt.wantName {
				t.Errorf("name = %q, want %q", m.Name(), tt.wantName)
			}
			if tt.wantKind == None && m.Available() {
				t.Error("expected not available for None")
			}
			if tt.wantKind != None && !m.Available() {
				t.Error("expected available")
			}
		})
	}
}

func TestInSession(t *testing.T) {
	// cmux always targets the running app.
	if !(Mux{Kind: Cmux, Bin: "/usr/bin/cmux"}).InSession() {
		t.Error("cmux should always report InSession")
	}
	// tmux depends on $TMUX.
	inside := detect("linux", look("tmux"), envWith(map[string]string{"TMUX": "/tmp/tmux-1/default,123,0"}))
	if !inside.InSession() {
		t.Error("tmux with $TMUX set should be InSession")
	}
	outside := detect("linux", look("tmux"), envWith(nil))
	if outside.InSession() {
		t.Error("tmux without $TMUX should not be InSession")
	}
}

func TestNotifyArgs(t *testing.T) {
	cmux := Mux{Kind: Cmux, Bin: "/usr/bin/cmux"}
	args, ok := cmux.notifyArgs("Phase 1 Complete", "Analysis done")
	if !ok || strings.Join(args, " ") != "notify --title Phase 1 Complete --body Analysis done" {
		t.Errorf("cmux notify args = %v", args)
	}

	// tmux inside session: no -t target.
	tin := detect("linux", look("tmux"), envWith(map[string]string{"TMUX": "x"}))
	args, ok = tin.notifyArgs("Done", "ok")
	if !ok || args[0] != "display-message" {
		t.Fatalf("tmux notify args = %v", args)
	}
	if contains(args, "-t") {
		t.Errorf("inside-session tmux should not target a session: %v", args)
	}

	// tmux outside session: targets the moonbase session.
	tout := detect("linux", look("tmux"), envWith(nil))
	args, _ = tout.notifyArgs("Done", "ok")
	if !contains(args, "-t") || !contains(args, "moonbase") {
		t.Errorf("outside-session tmux should target moonbase: %v", args)
	}

	if _, ok := (Mux{Kind: None}).notifyArgs("a", "b"); ok {
		t.Error("None must not produce notify args")
	}
}

func TestSplitArgs(t *testing.T) {
	cmux := Mux{Kind: Cmux, Bin: "/usr/bin/cmux"}
	args, ok := cmux.splitArgs(Right, "moonbase deploy 4")
	if !ok || !contains(args, "split") || !contains(args, "right") || !contains(args, "moonbase deploy 4") {
		t.Errorf("cmux split args = %v", args)
	}

	tmux := detect("linux", look("tmux"), envWith(map[string]string{"TMUX": "x"}))
	args, ok = tmux.splitArgs(Down, "echo hi")
	if !ok || args[0] != "split-window" || !contains(args, "-v") || !contains(args, "echo hi") {
		t.Errorf("tmux split args = %v", args)
	}
	args, _ = tmux.splitArgs(Right, "x")
	if !contains(args, "-h") {
		t.Errorf("Right should map to -h for tmux: %v", args)
	}
}

func TestWindowAndSendKeys(t *testing.T) {
	tmux := detect("linux", look("tmux"), envWith(map[string]string{"TMUX": "x"}))
	w, ok := tmux.windowArgs("mission", "moonbase mission x")
	if !ok || w[0] != "new-window" || !contains(w, "mission") {
		t.Errorf("tmux window args = %v", w)
	}
	sk, ok := tmux.sendKeysArgs("ls -la")
	if !ok || sk[0] != "send-keys" || !contains(sk, "ls -la") || !contains(sk, "Enter") {
		t.Errorf("tmux send-keys args = %v", sk)
	}

	cmux := Mux{Kind: Cmux, Bin: "/usr/bin/cmux"}
	w, ok = cmux.windowArgs("mission", "")
	if !ok || !contains(w, "workspace") || !contains(w, "mission") {
		t.Errorf("cmux workspace args = %v", w)
	}
	if _, ok := cmux.sendKeysArgs("x"); ok {
		t.Error("cmux has no send-keys; expected ok=false")
	}
}

// Unavailable multiplexer must make all run methods safe no-ops.
func TestNoneIsSafeNoop(t *testing.T) {
	none := Mux{Kind: None}
	if err := none.Notify("a", "b"); err != nil {
		t.Errorf("Notify on None should be nil, got %v", err)
	}
	if err := none.SplitRun(Right, "x"); err != nil {
		t.Errorf("SplitRun on None should be nil, got %v", err)
	}
	if err := none.NewWindow("n", "x"); err != nil {
		t.Errorf("NewWindow on None should be nil, got %v", err)
	}
	if err := none.SendKeys("x"); err != nil {
		t.Errorf("SendKeys on None should be nil, got %v", err)
	}
}

func contains(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}
