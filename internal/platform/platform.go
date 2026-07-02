package platform

import (
	"os"
	"path/filepath"
	"strings"
)

type Context int

const (
	Personal Context = iota
	Work
)

// Detect determines platform context by checking if the current working
// directory is under ~/Workspace/Personal. No hardcoded credentials.
func Detect() Context {
	cwd, err := os.Getwd()
	if err != nil {
		return Work // default to restricted
	}
	home, _ := os.UserHomeDir()
	personalDir := filepath.Join(home, "Workspace", "Personal")
	if strings.HasPrefix(cwd, personalDir) {
		return Personal
	}
	return Work
}

func (c Context) IsPersonal() bool {
	return c == Personal
}
