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

// clipboardCmd returns the appropriate clipboard command for the current OS.
func clipboardCmd() (*exec.Cmd, error) {
	switch runtime.GOOS {
	case "darwin":
		if path, err := exec.LookPath("pbcopy"); err == nil {
			return exec.Command(path), nil
		}
	case "linux":
		if path, err := exec.LookPath("xclip"); err == nil {
			return exec.Command(path, "-selection", "clipboard"), nil
		}
		if path, err := exec.LookPath("xsel"); err == nil {
			return exec.Command(path, "--clipboard", "--input"), nil
		}
		if path, err := exec.LookPath("wl-copy"); err == nil {
			return exec.Command(path), nil
		}
	case "windows":
		if path, err := exec.LookPath("clip"); err == nil {
			return exec.Command(path), nil
		}
	}
	return nil, fmt.Errorf("no clipboard command found (tried: pbcopy, xclip, xsel, wl-copy, clip)")
}
