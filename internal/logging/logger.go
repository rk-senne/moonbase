package logging

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
)

// Logger is the global structured logger for moonbase.
var Logger *slog.Logger

// Init initializes the global logger.
// If debug is true, writes JSON logs to ~/.config/moonbase/debug.log (append mode, 0600).
// If debug is false, uses a discard handler (no output).
func Init(debug bool) {
	if debug {
		dir := filepath.Join(homeDir(), ".config", "moonbase")
		if err := os.MkdirAll(dir, 0700); err != nil {
			// Fall back to discard if we can't create the directory
			Logger = slog.New(slog.NewTextHandler(io.Discard, nil))
			return
		}

		logPath := filepath.Join(dir, "debug.log")
		f, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
		if err != nil {
			Logger = slog.New(slog.NewTextHandler(io.Discard, nil))
			return
		}

		Logger = slog.New(slog.NewJSONHandler(f, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}))
	} else {
		Logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
}

func homeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "/tmp"
	}
	return home
}
