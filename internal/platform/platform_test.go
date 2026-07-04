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

func TestDetect_WorkDirectory(t *testing.T) {
	// Change to a directory that's NOT under ~/Workspace/Personal
	tmpDir := t.TempDir()
	orig, _ := os.Getwd()
	defer os.Chdir(orig)

	os.Chdir(tmpDir)
	ctx := Detect()
	if ctx != Work {
		t.Errorf("expected Work context for temp dir %s, got Personal", tmpDir)
	}
}

func TestDetect_PersonalFromPersonalDir(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("cannot determine home dir")
	}

	personalDir := filepath.Join(home, "Workspace", "Personal")
	if _, err := os.Stat(personalDir); os.IsNotExist(err) {
		t.Skip("~/Workspace/Personal doesn't exist")
	}

	orig, _ := os.Getwd()
	defer os.Chdir(orig)

	os.Chdir(personalDir)
	ctx := Detect()
	// personalDir itself matches the HasPrefix check (it equals the prefix)
	// The HasPrefix check is strings.HasPrefix, so /Workspace/Personal matches /Workspace/Personal
	if ctx != Personal {
		t.Error("expected Personal context when CWD is ~/Workspace/Personal")
	}
}

func TestContext_Constants(t *testing.T) {
	if Personal != 0 {
		t.Errorf("expected Personal=0, got %d", Personal)
	}
	if Work != 1 {
		t.Errorf("expected Work=1, got %d", Work)
	}
}

func TestIsPersonal_WorkContext(t *testing.T) {
	var ctx Context = Work
	if ctx.IsPersonal() {
		t.Error("Work.IsPersonal() should be false")
	}
}

func TestIsPersonal_PersonalContext(t *testing.T) {
	var ctx Context = Personal
	if !ctx.IsPersonal() {
		t.Error("Personal.IsPersonal() should be true")
	}
}

func TestDetect_DeletedCWD(t *testing.T) {
	// When CWD is deleted, os.Getwd() returns an error.
	// Detect should return Work (default to restricted) in that case.
	tmpDir, err := os.MkdirTemp("", "platform-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	orig, _ := os.Getwd()
	defer os.Chdir(orig)

	// Change to the temp directory, then delete it
	os.Chdir(tmpDir)
	os.RemoveAll(tmpDir)

	// Now os.Getwd() should fail (CWD no longer exists)
	ctx := Detect()
	// Should default to Work when CWD can't be determined
	if ctx != Work {
		t.Errorf("expected Work context when CWD is deleted, got: %d", ctx)
	}
}
