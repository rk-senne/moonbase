package main

import (
	"fmt"
	"os"
)

// mustGetwd returns the current working directory or exits with a clear error.
// Production code must not discard os.Getwd errors — a missing cwd indicates a
// broken process state that cannot be recovered from.
func mustGetwd() string {
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Cannot determine current directory: %v\n", err)
		osExit(1)
	}
	return cwd
}

// mustUserHomeDir returns the user's home directory or exits with a clear error.
// Production code must not discard os.UserHomeDir errors — a missing home directory
// prevents writing checkpoints, config, and history.
func mustUserHomeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Cannot determine home directory: %v\n", err)
		osExit(1)
	}
	return home
}
