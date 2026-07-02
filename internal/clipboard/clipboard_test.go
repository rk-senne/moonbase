package clipboard

import (
	"runtime"
	"testing"
)

func TestAvailable(t *testing.T) {
	// On macOS in dev, pbcopy should be available
	if runtime.GOOS == "darwin" {
		if !Available() {
			t.Error("expected clipboard to be available on macOS")
		}
	}
	// On any platform, Available() should not panic
	_ = Available()
}

func TestCopy(t *testing.T) {
	if !Available() {
		t.Skip("no clipboard command available")
	}

	err := Copy("moonbase test")
	if err != nil {
		t.Fatalf("Copy failed: %v", err)
	}
}

func TestCopy_EmptyString(t *testing.T) {
	if !Available() {
		t.Skip("no clipboard command available")
	}

	err := Copy("")
	if err != nil {
		t.Fatalf("Copy empty string failed: %v", err)
	}
}

func TestCopy_LargeString(t *testing.T) {
	if !Available() {
		t.Skip("no clipboard command available")
	}

	// 10KB string — simulates a composed agent prompt
	large := make([]byte, 10*1024)
	for i := range large {
		large[i] = 'A' + byte(i%26)
	}

	err := Copy(string(large))
	if err != nil {
		t.Fatalf("Copy large string failed: %v", err)
	}
}

// === Gap Coverage: Available() returns bool without panic on any OS ===

func TestAvailable_ReturnsBool(t *testing.T) {
	// This test ensures Available() never panics, regardless of platform.
	// It should always return a bool without crashing.
	result := Available()
	// Just verify it returns without panic — the value depends on platform
	if result != true && result != false {
		t.Error("Available() returned something that's not true or false (impossible, but validates interface)")
	}
}

func TestAvailable_Idempotent(t *testing.T) {
	// Calling Available() multiple times should give consistent results
	first := Available()
	second := Available()
	if first != second {
		t.Error("Available() returned different values on consecutive calls")
	}
}

func TestCopy_SpecialCharacters(t *testing.T) {
	if !Available() {
		t.Skip("no clipboard command available")
	}

	// Test with special characters that could break shell escaping
	specials := []string{
		"hello\nworld",
		"tab\there",
		"quotes \"double\" and 'single'",
		"backtick `cmd`",
		"dollar $HOME",
		"unicode: 日本語 中文 한국어",
		"emoji: 🌙🚀",
		"backslash \\n \\t",
	}

	for _, s := range specials {
		t.Run(s[:min(10, len(s))], func(t *testing.T) {
			err := Copy(s)
			if err != nil {
				t.Fatalf("Copy failed for special chars: %v", err)
			}
		})
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
