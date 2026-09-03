package updater

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// replaceBinary is the last step of a self-update: it swaps a freshly downloaded
// binary over the one currently on disk. A failure here can leave a user with no
// working binary at all, so the rollback path matters more than the happy path.

func TestReplaceBinary_SwapsContentsAndRemovesBackup(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "moonbase")
	incoming := filepath.Join(dir, "new-moonbase")

	if err := os.WriteFile(target, []byte("OLD"), 0o755); err != nil {
		t.Fatalf("seeding target: %v", err)
	}
	if err := os.WriteFile(incoming, []byte("NEW"), 0o644); err != nil {
		t.Fatalf("seeding incoming: %v", err)
	}

	if err := replaceBinary(target, incoming, 0o755); err != nil {
		t.Fatalf("replaceBinary: %v", err)
	}

	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("reading target: %v", err)
	}
	if string(got) != "NEW" {
		t.Errorf("target contents = %q, want NEW", got)
	}
	if _, err := os.Stat(target + ".bak"); !os.IsNotExist(err) {
		t.Error("backup should be removed after a successful replace")
	}
}

func TestReplaceBinary_PreservesRequestedMode(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix file modes not meaningful on windows")
	}
	dir := t.TempDir()
	target := filepath.Join(dir, "moonbase")
	incoming := filepath.Join(dir, "new-moonbase")
	os.WriteFile(target, []byte("OLD"), 0o755)
	os.WriteFile(incoming, []byte("NEW"), 0o600)

	if err := replaceBinary(target, incoming, 0o755); err != nil {
		t.Fatalf("replaceBinary: %v", err)
	}

	fi, err := os.Stat(target)
	if err != nil {
		t.Fatalf("stat target: %v", err)
	}
	if perm := fi.Mode().Perm(); perm != 0o755 {
		t.Errorf("mode = %o, want 755 — an update must not drop the exec bit", perm)
	}
}

// The critical safety property: if installing the new binary fails, the original
// must be restored rather than left missing.
func TestReplaceBinary_RestoresOriginalWhenInstallFails(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "moonbase")
	os.WriteFile(target, []byte("ORIGINAL"), 0o755)

	// Point at a source that does not exist so copyBinary fails.
	missing := filepath.Join(dir, "does-not-exist")

	err := replaceBinary(target, missing, 0o755)
	if err == nil {
		t.Fatal("expected an error when the new binary cannot be read")
	}
	if !strings.Contains(err.Error(), "installing new binary") {
		t.Errorf("error = %q, want it to mention installing", err)
	}

	got, readErr := os.ReadFile(target)
	if readErr != nil {
		t.Fatalf("SAFETY: original binary missing after failed update: %v", readErr)
	}
	if string(got) != "ORIGINAL" {
		t.Errorf("SAFETY: original contents not restored, got %q", got)
	}
	if _, err := os.Stat(target + ".bak"); err == nil {
		t.Error("backup should have been renamed back, not left behind")
	}
}

func TestReplaceBinary_ClearsStaleBackupFirst(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "moonbase")
	incoming := filepath.Join(dir, "new-moonbase")
	os.WriteFile(target, []byte("OLD"), 0o755)
	os.WriteFile(incoming, []byte("NEW"), 0o644)
	// A leftover backup from a previous interrupted update.
	os.WriteFile(target+".bak", []byte("STALE"), 0o755)

	if err := replaceBinary(target, incoming, 0o755); err != nil {
		t.Fatalf("replaceBinary: %v", err)
	}

	if _, err := os.Stat(target + ".bak"); !os.IsNotExist(err) {
		t.Error("stale backup should not survive a successful replace")
	}
	got, _ := os.ReadFile(target)
	if string(got) != "NEW" {
		t.Errorf("target contents = %q, want NEW", got)
	}
}

func TestReplaceBinary_MissingTargetReportsBackupFailure(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "absent")
	incoming := filepath.Join(dir, "new-moonbase")
	os.WriteFile(incoming, []byte("NEW"), 0o644)

	err := replaceBinary(target, incoming, 0o755)
	if err == nil || !strings.Contains(err.Error(), "backing up current binary") {
		t.Fatalf("expected backup failure, got %v", err)
	}
}
