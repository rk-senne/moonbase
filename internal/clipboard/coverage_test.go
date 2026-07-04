package clipboard

import (
	"errors"
	"os/exec"
	"runtime"
	"testing"
)

// TestClipboardCmd_Darwin tests the darwin branch of clipboardCmd.
func TestClipboardCmd_Darwin(t *testing.T) {
	origOS := osName
	origLookPath := lookPath
	defer func() {
		osName = origOS
		lookPath = origLookPath
	}()

	osName = "darwin"
	lookPath = func(file string) (string, error) {
		if file == "pbcopy" {
			return "/usr/bin/pbcopy", nil
		}
		return "", errors.New("not found")
	}

	cmd, err := clipboardCmd()
	if err != nil {
		t.Fatalf("expected no error for darwin with pbcopy, got: %v", err)
	}
	if cmd == nil {
		t.Fatal("expected non-nil command")
	}
	if cmd.Path != "/usr/bin/pbcopy" {
		t.Errorf("expected /usr/bin/pbcopy, got %s", cmd.Path)
	}
}

// TestClipboardCmd_Darwin_NoPbcopy tests darwin without pbcopy.
func TestClipboardCmd_Darwin_NoPbcopy(t *testing.T) {
	origOS := osName
	origLookPath := lookPath
	defer func() {
		osName = origOS
		lookPath = origLookPath
	}()

	osName = "darwin"
	lookPath = func(file string) (string, error) {
		return "", errors.New("not found")
	}

	_, err := clipboardCmd()
	if err == nil {
		t.Fatal("expected error when pbcopy not found")
	}
}

// TestClipboardCmd_Linux_Xclip tests the linux xclip branch.
func TestClipboardCmd_Linux_Xclip(t *testing.T) {
	origOS := osName
	origLookPath := lookPath
	defer func() {
		osName = origOS
		lookPath = origLookPath
	}()

	osName = "linux"
	lookPath = func(file string) (string, error) {
		if file == "xclip" {
			return "/usr/bin/xclip", nil
		}
		return "", errors.New("not found")
	}

	cmd, err := clipboardCmd()
	if err != nil {
		t.Fatalf("expected no error for linux with xclip, got: %v", err)
	}
	if cmd.Path != "/usr/bin/xclip" {
		t.Errorf("expected /usr/bin/xclip, got %s", cmd.Path)
	}
	// Check args include -selection clipboard
	args := cmd.Args
	found := false
	for _, a := range args {
		if a == "-selection" {
			found = true
		}
	}
	if !found {
		t.Error("expected -selection arg for xclip")
	}
}

// TestClipboardCmd_Linux_Xsel tests the linux xsel fallback.
func TestClipboardCmd_Linux_Xsel(t *testing.T) {
	origOS := osName
	origLookPath := lookPath
	defer func() {
		osName = origOS
		lookPath = origLookPath
	}()

	osName = "linux"
	lookPath = func(file string) (string, error) {
		if file == "xsel" {
			return "/usr/bin/xsel", nil
		}
		return "", errors.New("not found")
	}

	cmd, err := clipboardCmd()
	if err != nil {
		t.Fatalf("expected no error for linux with xsel, got: %v", err)
	}
	if cmd.Path != "/usr/bin/xsel" {
		t.Errorf("expected /usr/bin/xsel, got %s", cmd.Path)
	}
}

// TestClipboardCmd_Linux_WlCopy tests the linux wl-copy fallback.
func TestClipboardCmd_Linux_WlCopy(t *testing.T) {
	origOS := osName
	origLookPath := lookPath
	defer func() {
		osName = origOS
		lookPath = origLookPath
	}()

	osName = "linux"
	lookPath = func(file string) (string, error) {
		if file == "wl-copy" {
			return "/usr/bin/wl-copy", nil
		}
		return "", errors.New("not found")
	}

	cmd, err := clipboardCmd()
	if err != nil {
		t.Fatalf("expected no error for linux with wl-copy, got: %v", err)
	}
	if cmd.Path != "/usr/bin/wl-copy" {
		t.Errorf("expected /usr/bin/wl-copy, got %s", cmd.Path)
	}
}

// TestClipboardCmd_Linux_NoTool tests linux without any clipboard tool.
func TestClipboardCmd_Linux_NoTool(t *testing.T) {
	origOS := osName
	origLookPath := lookPath
	defer func() {
		osName = origOS
		lookPath = origLookPath
	}()

	osName = "linux"
	lookPath = func(file string) (string, error) {
		return "", errors.New("not found")
	}

	_, err := clipboardCmd()
	if err == nil {
		t.Fatal("expected error when no linux clipboard tool available")
	}
}

// TestClipboardCmd_Windows tests the windows branch.
func TestClipboardCmd_Windows(t *testing.T) {
	origOS := osName
	origLookPath := lookPath
	defer func() {
		osName = origOS
		lookPath = origLookPath
	}()

	osName = "windows"
	lookPath = func(file string) (string, error) {
		if file == "clip" {
			return "C:\\Windows\\System32\\clip.exe", nil
		}
		return "", errors.New("not found")
	}

	cmd, err := clipboardCmd()
	if err != nil {
		t.Fatalf("expected no error for windows with clip, got: %v", err)
	}
	if cmd.Path != "C:\\Windows\\System32\\clip.exe" {
		t.Errorf("expected clip.exe path, got %s", cmd.Path)
	}
}

// TestClipboardCmd_Windows_NoClip tests windows without clip.
func TestClipboardCmd_Windows_NoClip(t *testing.T) {
	origOS := osName
	origLookPath := lookPath
	defer func() {
		osName = origOS
		lookPath = origLookPath
	}()

	osName = "windows"
	lookPath = func(file string) (string, error) {
		return "", errors.New("not found")
	}

	_, err := clipboardCmd()
	if err == nil {
		t.Fatal("expected error when clip not found on windows")
	}
}

// TestClipboardCmd_UnknownOS tests an unsupported OS.
func TestClipboardCmd_UnknownOS(t *testing.T) {
	origOS := osName
	origLookPath := lookPath
	defer func() {
		osName = origOS
		lookPath = origLookPath
	}()

	osName = "freebsd"
	lookPath = func(file string) (string, error) {
		return "/usr/bin/" + file, nil
	}

	_, err := clipboardCmd()
	if err == nil {
		t.Fatal("expected error for unsupported OS")
	}
}

// TestAvailable_AllPlatforms tests Available() with different simulated platforms.
func TestAvailable_AllPlatforms(t *testing.T) {
	origOS := osName
	origLookPath := lookPath
	defer func() {
		osName = origOS
		lookPath = origLookPath
	}()

	// Darwin with pbcopy
	osName = "darwin"
	lookPath = func(file string) (string, error) {
		if file == "pbcopy" {
			return "/usr/bin/pbcopy", nil
		}
		return "", errors.New("not found")
	}
	if !Available() {
		t.Error("expected Available=true on darwin with pbcopy")
	}

	// Unknown OS
	osName = "plan9"
	if Available() {
		t.Error("expected Available=false on plan9")
	}
}

// TestCopy_ErrorFromClipboardCmd tests Copy returns error when no clipboard tool.
func TestCopy_ErrorFromClipboardCmd(t *testing.T) {
	origOS := osName
	origLookPath := lookPath
	defer func() {
		osName = origOS
		lookPath = origLookPath
	}()

	osName = "freebsd"
	lookPath = func(file string) (string, error) {
		return "", errors.New("not found")
	}

	err := Copy("test")
	if err == nil {
		t.Fatal("expected error from Copy when no clipboard available")
	}
}

// TestCopy_RealClipboard verifies actual clipboard integration on macOS.
func TestCopy_RealClipboard(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("test only runs on macOS")
	}
	// Restore originals in case other tests modified them
	origOS := osName
	origLookPath := lookPath
	defer func() {
		osName = origOS
		lookPath = origLookPath
	}()
	osName = runtime.GOOS
	lookPath = exec.LookPath

	err := Copy("clipboard-integration-test")
	if err != nil {
		t.Fatalf("Copy failed on real macOS clipboard: %v", err)
	}
}
