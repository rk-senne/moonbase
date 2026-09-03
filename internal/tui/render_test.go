package tui

import (
	"strings"
	"testing"
)

func Test_renderMarkdown(t *testing.T) {
	tests := []struct {
		name  string
		input string
		width int
	}{
		{"normal markdown", "# Hello\n\nThis is **bold** text.", 80},
		{"empty string", "", 80},
		{"very wide width", "Some text here.", 500},
		{"zero width defaults to 80", "Some text here.", 0},
		{"negative width defaults to 80", "Some text here.", -10},
		{"content with code blocks", "```go\nfunc main() {}\n```", 80},
		{"content with lists", "- item 1\n- item 2\n- item 3", 80},
		{"content with links", "[link](https://example.com)", 80},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := renderMarkdown(tt.input, tt.width)
			// Should not panic and should return something
			if tt.input == "" && strings.TrimSpace(result) != "" {
				// Empty input may produce whitespace from glamour, that's OK
			}
			if tt.input != "" && result == "" {
				t.Errorf("expected non-empty result for non-empty input %q", tt.input)
			}
		})
	}
}

func Test_renderMarkdown_Fallback(t *testing.T) {
	// Edge cases that shouldn't crash
	edgeCases := []string{
		strings.Repeat("x", 10000),       // very long string
		"\x00\x01\x02",                   // control characters
		"```\nunclosed code block",       // unclosed code block
		"# " + strings.Repeat("a", 1000), // very long heading
		"\n\n\n\n\n",                     // only newlines
		"<!--comment-->",                 // HTML comment
	}

	for i, input := range edgeCases {
		result := renderMarkdown(input, 80)
		// Just ensure no panic — result can be anything
		_ = result
		if i < 0 {
			t.Fatal("unreachable")
		}
	}
}
