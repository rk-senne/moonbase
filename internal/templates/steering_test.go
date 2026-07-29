package templates

import (
	"strings"
	"testing"
)

var allStacks = []string{"go", "java", "javascript", "python", "rust", "unknown"}

func TestGenerateSteeringFiles_ProducesSixFiles(t *testing.T) {
	expectedFiles := []string{
		"dev-rules.md",
		"production-standards.md",
		"test-alignment.md",
		"reasoning-protocol.md",
		"quality-gates.md",
		"changelog.md",
	}

	for _, stack := range allStacks {
		t.Run(stack, func(t *testing.T) {
			files := GenerateSteeringFiles(stack, "", "")
			if len(files) != 6 {
				t.Errorf("expected 6 files for stack %q, got %d", stack, len(files))
			}
			for _, name := range expectedFiles {
				if _, ok := files[name]; !ok {
					t.Errorf("stack %q: missing expected file %q", stack, name)
				}
			}
			// data-access-performance.md is opt-in — must NOT be in the default set.
			if _, ok := files["data-access-performance.md"]; ok {
				t.Errorf("stack %q: data-access-performance.md should not be in the default set", stack)
			}
		})
	}
}

func TestGenerateSteeringFiles_GoStackContent(t *testing.T) {
	files := GenerateSteeringFiles("go", "", "")

	// dev-rules.md should have Go-specific content
	devRules := files["dev-rules.md"]
	assertContains(t, devRules, "go test", "dev-rules.md should mention go test")
	assertContains(t, devRules, "go build", "dev-rules.md should mention go build")
	assertContains(t, devRules, "fmt.Errorf", "dev-rules.md should mention fmt.Errorf")

	// production-standards.md should have defer
	prodStandards := files["production-standards.md"]
	assertContains(t, prodStandards, "defer", "production-standards.md should mention defer")
	assertContains(t, prodStandards, "defer f.Close()", "production-standards.md should mention defer f.Close()")

	// test-alignment.md should have Go test patterns
	testAlignment := files["test-alignment.md"]
	assertContains(t, testAlignment, "*_test.go", "test-alignment.md should mention *_test.go")
	assertContains(t, testAlignment, "go test -race", "test-alignment.md should mention go test -race")
	assertContains(t, testAlignment, "t.Helper()", "test-alignment.md should mention t.Helper()")

	// quality-gates.md should have Go build commands
	qualityGates := files["quality-gates.md"]
	assertContains(t, qualityGates, "go build ./...", "quality-gates.md should mention go build")
	assertContains(t, qualityGates, "go vet ./...", "quality-gates.md should mention go vet")
}

func TestGenerateSteeringFiles_JavaStackContent(t *testing.T) {
	files := GenerateSteeringFiles("java", "", "")

	// dev-rules.md should have Java-specific content
	devRules := files["dev-rules.md"]
	assertContains(t, devRules, "mvn", "dev-rules.md should mention mvn")
	assertContains(t, devRules, "Java 17+", "dev-rules.md should mention Java 17+")
	assertContains(t, devRules, "Constructor injection", "dev-rules.md should mention constructor injection")

	// production-standards.md should have try-with-resources
	prodStandards := files["production-standards.md"]
	assertContains(t, prodStandards, "try-with-resources", "production-standards.md should mention try-with-resources")

	// test-alignment.md should have Java test patterns
	testAlignment := files["test-alignment.md"]
	assertContains(t, testAlignment, "src/test/java/", "test-alignment.md should mention src/test/java/")
	assertContains(t, testAlignment, "MockitoExtension", "test-alignment.md should mention MockitoExtension")

	// quality-gates.md should have mvn
	qualityGates := files["quality-gates.md"]
	assertContains(t, qualityGates, "mvn clean verify", "quality-gates.md should mention mvn clean verify")
}

func TestGenerateSteeringFiles_JavaScriptStackContent(t *testing.T) {
	files := GenerateSteeringFiles("javascript", "", "")

	devRules := files["dev-rules.md"]
	assertContains(t, devRules, "TypeScript", "dev-rules.md should mention TypeScript")
	assertContains(t, devRules, "npm run build", "dev-rules.md should mention npm run build")

	prodStandards := files["production-standards.md"]
	assertContains(t, prodStandards, "AbortController", "production-standards.md should mention AbortController")

	qualityGates := files["quality-gates.md"]
	assertContains(t, qualityGates, "npm run lint", "quality-gates.md should mention npm run lint")
}

func TestGenerateSteeringFiles_PythonStackContent(t *testing.T) {
	files := GenerateSteeringFiles("python", "", "")

	devRules := files["dev-rules.md"]
	assertContains(t, devRules, "Python 3.10+", "dev-rules.md should mention Python 3.10+")
	assertContains(t, devRules, "pytest", "dev-rules.md should mention pytest")

	prodStandards := files["production-standards.md"]
	assertContains(t, prodStandards, "raise ... from ...", "production-standards.md should mention raise from")

	qualityGates := files["quality-gates.md"]
	assertContains(t, qualityGates, "ruff check", "quality-gates.md should mention ruff check")
}

func TestGenerateSteeringFiles_RustStackContent(t *testing.T) {
	files := GenerateSteeringFiles("rust", "", "")

	devRules := files["dev-rules.md"]
	assertContains(t, devRules, "cargo build", "dev-rules.md should mention cargo build")
	assertContains(t, devRules, "thiserror", "dev-rules.md should mention thiserror")

	prodStandards := files["production-standards.md"]
	assertContains(t, prodStandards, "Drop trait", "production-standards.md should mention Drop trait")
	assertContains(t, prodStandards, "RAII", "production-standards.md should mention RAII")

	qualityGates := files["quality-gates.md"]
	assertContains(t, qualityGates, "cargo clippy", "quality-gates.md should mention cargo clippy")
}

func TestGenerateSteeringFiles_UnknownStackDefaults(t *testing.T) {
	files := GenerateSteeringFiles("unknown-stack", "", "")

	// Should still produce 6 files
	if len(files) != 6 {
		t.Errorf("expected 6 files for unknown stack, got %d", len(files))
	}

	// dev-rules.md should have generic content
	devRules := files["dev-rules.md"]
	assertContains(t, devRules, "Development Rules", "dev-rules.md should have a title")
	assertContains(t, devRules, "existing code style", "dev-rules.md should mention following existing style")

	// production-standards.md should have generic error handling
	prodStandards := files["production-standards.md"]
	assertContains(t, prodStandards, "Zero Tolerance", "production-standards.md should have Zero Tolerance section")
	assertContains(t, prodStandards, "Wrap errors with context", "production-standards.md should have generic error guidance")

	// quality-gates.md should have placeholder build commands
	qualityGates := files["quality-gates.md"]
	assertContains(t, qualityGates, "project's build command", "quality-gates.md should have generic build guidance")
}

func TestGenerateSteeringFiles_ReasoningProtocolIdenticalAcrossStacks(t *testing.T) {
	var reference string
	for _, stack := range allStacks {
		files := GenerateSteeringFiles(stack, "", "")
		content := files["reasoning-protocol.md"]
		if reference == "" {
			reference = content
			continue
		}
		if content != reference {
			t.Errorf("reasoning-protocol.md differs for stack %q", stack)
		}
	}
}

func TestGenerateSteeringFiles_ChangelogIdenticalAcrossStacks(t *testing.T) {
	var reference string
	for _, stack := range allStacks {
		files := GenerateSteeringFiles(stack, "", "")
		content := files["changelog.md"]
		if reference == "" {
			reference = content
			continue
		}
		if content != reference {
			t.Errorf("changelog.md differs for stack %q", stack)
		}
	}
}

func TestGenerateDataAccessPerformance_UniversalCore(t *testing.T) {
	// Every stack must carry the universal data-access rules.
	for _, stack := range allStacks {
		content := GenerateDataAccessPerformance(stack)
		assertContains(t, content, "No unbounded queries", stack+": should ban unbounded queries")
		assertContains(t, content, "No N+1", stack+": should ban N+1")
		assertContains(t, content, "Push Down", stack+": should push work to data layer")
		assertContains(t, content, "Explicitly Allowed", stack+": should allow bounded in-memory ops")
	}
}

func TestGenerateDataAccessPerformance_StackSpecificORM(t *testing.T) {
	// Java should reference JPA/Hibernate + OSIV.
	java := GenerateDataAccessPerformance("java")
	assertContains(t, java, "Hibernate", "java data-access should mention Hibernate")
	assertContains(t, java, "open-in-view", "java data-access should mention OSIV")

	// Python should reference Django/SQLAlchemy prefetch idioms.
	py := GenerateDataAccessPerformance("python")
	assertContains(t, py, "select_related", "python data-access should mention select_related")

	// JS should reference Prisma/TypeORM include/relations.
	js := GenerateDataAccessPerformance("javascript")
	assertContains(t, js, "include", "js data-access should mention include")

	// Go should mention explicit, set-based access.
	golang := GenerateDataAccessPerformance("go")
	assertContains(t, golang, "batch", "go data-access should mention batch queries")
}

func TestNormalizeStack_Variants(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"go", "go"},
		{"Go", "go"},
		{"golang", "go"},
		{"GOLANG", "go"},
		{"java", "java"},
		{"Java", "java"},
		{"kotlin", "java"},
		{"spring", "java"},
		{"springboot", "java"},
		{"javascript", "javascript"},
		{"JavaScript", "javascript"},
		{"typescript", "javascript"},
		{"ts", "javascript"},
		{"js", "javascript"},
		{"node", "javascript"},
		{"nodejs", "javascript"},
		{"python", "python"},
		{"Python", "python"},
		{"py", "python"},
		{"rust", "rust"},
		{"Rust", "rust"},
		{"rs", "rust"},
		{"  go  ", "go"},
		{"unknown", "generic"},
		{"", "generic"},
		{"cobol", "generic"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := normalizeStack(tt.input)
			if got != tt.expected {
				t.Errorf("normalizeStack(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestGenerateSteeringFiles_AllFilesNonEmpty(t *testing.T) {
	for _, stack := range allStacks {
		t.Run(stack, func(t *testing.T) {
			files := GenerateSteeringFiles(stack, "", "")
			for name, content := range files {
				if strings.TrimSpace(content) == "" {
					t.Errorf("stack %q: file %q is empty", stack, name)
				}
			}
		})
	}
}

func TestGenerateSteeringFiles_AllFilesHaveTitle(t *testing.T) {
	for _, stack := range allStacks {
		t.Run(stack, func(t *testing.T) {
			files := GenerateSteeringFiles(stack, "", "")
			for name, content := range files {
				if !strings.HasPrefix(content, "# ") {
					t.Errorf("stack %q: file %q should start with a markdown title", stack, name)
				}
			}
		})
	}
}

func assertContains(t *testing.T, content, substr, msg string) {
	t.Helper()
	if !strings.Contains(content, substr) {
		t.Errorf("%s — content does not contain %q", msg, substr)
	}
}
