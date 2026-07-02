package platform

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetect_ReturnsValidContext(t *testing.T) {
	ctx := Detect()
	// Should be either Personal or Work — never panic
	if ctx != Personal && ctx != Work {
		t.Errorf("unexpected context value: %d", ctx)
	}
}

func TestIsPersonal(t *testing.T) {
	if !Personal.IsPersonal() {
		t.Error("Personal.IsPersonal() should return true")
	}
	if Work.IsPersonal() {
		t.Error("Work.IsPersonal() should return false")
	}
}

func TestDetect_PersonalDirectory(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("cannot determine home dir")
	}

	personalDir := filepath.Join(home, "Workspace", "Personal")
	if _, err := os.Stat(personalDir); os.IsNotExist(err) {
		t.Skip("~/Workspace/Personal doesn't exist on this system")
	}

	// If we're currently in a personal workspace subdirectory, detect should return Personal
	cwd, _ := os.Getwd()
	if len(cwd) > len(personalDir) && cwd[:len(personalDir)] == personalDir {
		ctx := Detect()
		if ctx != Personal {
			t.Error("expected Personal context when CWD is under ~/Workspace/Personal")
		}
	}
}
