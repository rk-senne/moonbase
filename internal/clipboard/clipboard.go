// Package clipboard provides cross-platform clipboard copy support.
// It detects the appropriate system clipboard command (pbcopy on macOS,
// xclip/xsel/wl-copy on Linux, clip on Windows) and provides a simple
// Copy interface for writing text to the clipboard.
package clipboard

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

// Copy copies text to the system clipboard.
// Supports macOS (pbcopy), Linux (xclip), and Windows (clip).
// Returns an error if no clipboard command is available.
func Copy(text string) error {
	cmd, err := clipboardCmd()
	if err != nil {
		return err
	}
	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
}

// Available returns true if a clipboard command exists on this system.
func Available() bool {
	_, err := clipboardCmd()
	return err == nil
}

// osName is the detected operating system. It defaults to runtime.GOOS
// but can be overridden in tests to exercise platform-specific paths.
var osName = runtime.GOOS

// lookPath is the function used to look up executables. It defaults to
// exec.LookPath but can be overridden in tests.
var lookPath = exec.LookPath

// clipboardCmd returns the appropriate clipboard command for the current OS.
func clipboardCmd() (*exec.Cmd, error) {
	switch osName {
	case "darwin":
		if path, err := lookPath("pbcopy"); err == nil {
			return exec.Command(path), nil
		}
	case "linux":
		if path, err := lookPath("xclip"); err == nil {
			return exec.Command(path, "-selection", "clipboard"), nil
		}
		if path, err := lookPath("xsel"); err == nil {
			return exec.Command(path, "--clipboard", "--input"), nil
		}
		if path, err := lookPath("wl-copy"); err == nil {
			return exec.Command(path), nil
		}
	case "windows":
		if path, err := lookPath("clip"); err == nil {
			return exec.Command(path), nil
		}
	}
	return nil, fmt.Errorf("no clipboard command found (tried: pbcopy, xclip, xsel, wl-copy, clip)")
}
