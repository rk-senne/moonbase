package backend

import (
	"os"
	"testing"
)

// === Kimi backend tests ===

func TestKimi_Name(t *testing.T) {
	k := &Kimi{}
	if k.Name() != "kimi" {
		t.Errorf("expected 'kimi', got %q", k.Name())
	}
}

func TestKimi_Available_WithKey(t *testing.T) {
	// Set the key temporarily
	original := os.Getenv("MOONSHOT_API_KEY")
	os.Setenv("MOONSHOT_API_KEY", "test-key-12345")
	defer func() {
		if original == "" {
			os.Unsetenv("MOONSHOT_API_KEY")
		} else {
			os.Setenv("MOONSHOT_API_KEY", original)
		}
	}()

	k := &Kimi{}
	if !k.Available() {
		t.Error("expected Available=true when MOONSHOT_API_KEY is set")
	}
}

func TestKimi_Available_WithoutKey(t *testing.T) {
	// Ensure the key is not set
	original := os.Getenv("MOONSHOT_API_KEY")
	os.Unsetenv("MOONSHOT_API_KEY")
	defer func() {
		if original != "" {
			os.Setenv("MOONSHOT_API_KEY", original)
		}
	}()

	k := &Kimi{}
	if k.Available() {
		t.Error("expected Available=false when MOONSHOT_API_KEY is not set")
	}
}

func TestKimi_InDetectAll(t *testing.T) {
	all := DetectAll()
	found := false
	for _, b := range all {
		if b.Name() == "kimi" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'kimi' backend in DetectAll() results")
	}
}

// === Cmux utility tests ===

func TestCmuxAvailable_ReturnsBool(t *testing.T) {
	// cmux is likely not installed in test environment
	// Just verify it doesn't panic and returns a bool
	result := CmuxAvailable()
	// On most systems, cmux won't be installed
	_ = result
}

func TestCmuxNotify_WhenUnavailable(t *testing.T) {
	if CmuxAvailable() {
		t.Skip("cmux is available, skipping unavailable test")
	}

	// When cmux is not available, CmuxNotify should return nil (silent no-op)
	err := CmuxNotify("Test Title", "Test Body")
	if err != nil {
		t.Errorf("expected nil error when cmux unavailable, got: %v", err)
	}
}

func TestCmuxSplit_WhenUnavailable(t *testing.T) {
	if CmuxAvailable() {
		t.Skip("cmux is available, skipping unavailable test")
	}

	// When cmux is not available, CmuxSplit should return an error
	err := CmuxSplit("right", "echo hello")
	if err == nil {
		t.Error("expected error when cmux is not available")
	}
}

func TestCmuxWorkspace_WhenUnavailable(t *testing.T) {
	if CmuxAvailable() {
		t.Skip("cmux is available, skipping unavailable test")
	}

	// When cmux is not available, CmuxWorkspace should return an error
	err := CmuxWorkspace("test-workspace")
	if err == nil {
		t.Error("expected error when cmux is not available")
	}
}

// === SafeEnv tests ===

func TestSafeEnv_ContainsBasicVars(t *testing.T) {
	env := SafeEnv()

	// HOME and PATH should always be present
	hasHome := false
	hasPath := false
	for _, e := range env {
		if len(e) >= 5 && e[:5] == "HOME=" {
			hasHome = true
		}
		if len(e) >= 5 && e[:5] == "PATH=" {
			hasPath = true
		}
	}
	if !hasHome {
		t.Error("SafeEnv should include HOME")
	}
	if !hasPath {
		t.Error("SafeEnv should include PATH")
	}
}

func TestSafeEnv_ExcludesSensitiveVars(t *testing.T) {
	// Set some sensitive vars
	os.Setenv("AWS_SECRET_ACCESS_KEY", "secret123")
	os.Setenv("GITHUB_TOKEN", "ghp_xxx")
	defer func() {
		os.Unsetenv("AWS_SECRET_ACCESS_KEY")
		os.Unsetenv("GITHUB_TOKEN")
	}()

	env := SafeEnv()
	for _, e := range env {
		if len(e) >= 22 && e[:22] == "AWS_SECRET_ACCESS_KEY=" {
			t.Error("SafeEnv should NOT include AWS_SECRET_ACCESS_KEY")
		}
		if len(e) >= 13 && e[:13] == "GITHUB_TOKEN=" {
			t.Error("SafeEnv should NOT include GITHUB_TOKEN")
		}
	}
}

func TestSafeEnv_IncludesAPIKeys(t *testing.T) {
	// Set allowed API keys
	os.Setenv("OPENAI_API_KEY", "sk-test")
	os.Setenv("MOONSHOT_API_KEY", "msk-test")
	defer func() {
		os.Unsetenv("OPENAI_API_KEY")
		os.Unsetenv("MOONSHOT_API_KEY")
	}()

	env := SafeEnv()
	hasOpenAI := false
	hasMoonshot := false
	for _, e := range env {
		if len(e) >= 14 && e[:14] == "OPENAI_API_KEY" {
			hasOpenAI = true
		}
		if len(e) >= 16 && e[:16] == "MOONSHOT_API_KEY" {
			hasMoonshot = true
		}
	}
	if !hasOpenAI {
		t.Error("SafeEnv should include OPENAI_API_KEY when set")
	}
	if !hasMoonshot {
		t.Error("SafeEnv should include MOONSHOT_API_KEY when set")
	}
}

// === envExists tests ===

func TestEnvExists_Set(t *testing.T) {
	os.Setenv("TEST_MOONBASE_EXISTS", "value")
	defer os.Unsetenv("TEST_MOONBASE_EXISTS")

	if !envExists("TEST_MOONBASE_EXISTS") {
		t.Error("expected envExists=true for set variable")
	}
}

func TestEnvExists_NotSet(t *testing.T) {
	os.Unsetenv("TEST_MOONBASE_NOT_EXISTS")

	if envExists("TEST_MOONBASE_NOT_EXISTS") {
		t.Error("expected envExists=false for unset variable")
	}
}

func TestEnvExists_EmptyValue(t *testing.T) {
	os.Setenv("TEST_MOONBASE_EMPTY", "")
	defer os.Unsetenv("TEST_MOONBASE_EMPTY")

	// Empty string is still "exists"
	if !envExists("TEST_MOONBASE_EMPTY") {
		t.Error("expected envExists=true for empty-value variable")
	}
}

// === Truncate helper tests ===

func TestTruncate_ExactLength(t *testing.T) {
	result := truncate("hello", 5)
	if result != "hello" {
		t.Errorf("expected 'hello', got %q", result)
	}
}
