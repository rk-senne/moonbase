package discovery

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestTruncateRunes_ShorterThanLimitIsUnchanged(t *testing.T) {
	in := "line one\nline two"
	if got := truncateRunes(in, 1000); got != in {
		t.Errorf("expected input returned unchanged, got %q", got)
	}
}

func TestTruncateRunes_ExactlyAtLimitIsUnchanged(t *testing.T) {
	in := strings.Repeat("a", 100)
	if got := truncateRunes(in, 100); got != in {
		t.Errorf("expected input at limit returned unchanged, got length %d", len([]rune(got)))
	}
}

func TestTruncateRunes_NonPositiveLimitIsEmpty(t *testing.T) {
	for _, limit := range []int{0, -1, -100} {
		if got := truncateRunes("content", limit); got != "" {
			t.Errorf("limit %d: expected empty string, got %q", limit, got)
		}
	}
}

func TestTruncateRunes_AppendsMarkerWhenTruncated(t *testing.T) {
	in := strings.Repeat("a", 500)
	got := truncateRunes(in, 100)
	if !strings.HasSuffix(got, truncationMarker) {
		t.Errorf("expected truncation marker suffix, got %q", got[max(0, len(got)-40):])
	}
}

// Regression: truncation must never split a multi-byte UTF-8 character.
// Byte-slicing produced invalid UTF-8 that kiro-cli rejects.
func TestTruncateRunes_NeverProducesInvalidUTF8(t *testing.T) {
	// Each rune here is 3 bytes, so any byte-based cut lands mid-character.
	in := strings.Repeat("日", 500)
	for _, limit := range []int{1, 7, 50, 99, 100, 101, 250} {
		got := truncateRunes(in, limit)
		if !utf8.ValidString(got) {
			t.Errorf("limit %d: produced invalid UTF-8", limit)
		}
	}
}

func TestTruncateRunes_RespectsRuneLimitNotByteLimit(t *testing.T) {
	// 300 three-byte runes = 900 bytes. A 200-rune limit must keep 200 runes,
	// not 200 bytes (which would be ~66 runes).
	in := strings.Repeat("日", 300)
	got := truncateRunes(in, 200)
	body := strings.TrimSuffix(got, truncationMarker)
	if n := len([]rune(body)); n != 200 {
		t.Errorf("expected 200 runes retained, got %d", n)
	}
}

func TestTruncateRunes_CutsAtLineBoundaryWhenAvailable(t *testing.T) {
	// A newline sits at rune 95, inside the final 20% of a 100-rune budget,
	// so the cut should land there rather than mid-line at 100.
	in := strings.Repeat("a", 95) + "\n" + strings.Repeat("b", 100)
	got := truncateRunes(in, 100)
	body := strings.TrimSuffix(got, truncationMarker)
	if strings.Contains(body, "b") {
		t.Errorf("expected cut at line boundary before any 'b', got %q", body)
	}
	if n := len([]rune(body)); n != 95 {
		t.Errorf("expected 95 runes up to the newline, got %d", n)
	}
}

func TestTruncateRunes_HardCutsWhenNoBoundaryInWindow(t *testing.T) {
	// The only newline is at rune 10, far outside the final 20% of a 100-rune
	// budget, so cutting there would discard too much — expect a hard cut.
	in := strings.Repeat("a", 10) + "\n" + strings.Repeat("b", 200)
	got := truncateRunes(in, 100)
	body := strings.TrimSuffix(got, truncationMarker)
	if n := len([]rune(body)); n != 100 {
		t.Errorf("expected hard cut at 100 runes, got %d", n)
	}
}

func TestTruncateRunes_TrimsTrailingWhitespaceAtBoundaryCut(t *testing.T) {
	in := strings.Repeat("a", 90) + "   \n" + strings.Repeat("b", 100)
	got := truncateRunes(in, 100)
	body := strings.TrimSuffix(got, truncationMarker)
	if strings.HasSuffix(body, " ") || strings.HasSuffix(body, "\t") {
		t.Errorf("expected trailing whitespace trimmed, got %q", body)
	}
}

// The curated skills library must load without truncation — that was the
// defect this budget split fixes.
func TestMaxSkillContentSize_FitsCuratedLibrary(t *testing.T) {
	// Largest curated skill body is ~3,100 runes; assert real headroom so the
	// limit is not silently re-tightened below the library it must serve.
	const largestCuratedSkillRunes = 3100
	if maxSkillContentSize < largestCuratedSkillRunes {
		t.Fatalf("maxSkillContentSize (%d) is below the largest curated skill (%d) — "+
			"progressive skill loads would be truncated",
			maxSkillContentSize, largestCuratedSkillRunes)
	}
}
