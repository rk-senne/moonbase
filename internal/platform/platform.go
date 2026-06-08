package platform

import (
	"os/exec"
	"strings"
)

type Context int

const (
	Personal Context = iota
	Work
)

const personalEmail = "rego.senne@icloud.com"

// Detect checks git user.email to determine platform context
func Detect() Context {
	out, err := exec.Command("git", "config", "user.email").Output()
	if err != nil {
		return Work // default to restricted
	}
	email := strings.TrimSpace(string(out))
	if email == personalEmail {
		return Personal
	}
	return Work
}

func (c Context) IsPersonal() bool {
	return c == Personal
}
