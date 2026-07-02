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
