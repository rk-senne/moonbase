package tui

import (
	"os"
	"syscall"
	"testing"
)

func TestMain(m *testing.M) {
	// Raise the file descriptor limit for tests.
	// NewApp() creates a file watcher that opens many FDs (watches home dir tree).
	// With 200+ tests each creating NewApp(), we need a high limit.
	var rLimit syscall.Rlimit
	if err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rLimit); err == nil {
		if rLimit.Cur < 10240 {
			rLimit.Cur = 10240
			if rLimit.Max < 10240 {
				rLimit.Max = 10240
			}
			syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rLimit)
		}
	}

	// Skip file watcher creation in NewApp during tests
	testMode = true

	os.Exit(m.Run())
}
