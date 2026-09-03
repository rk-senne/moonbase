package discovery

import "strings"

// truncationMarker is appended to any content shortened by truncateRunes.
const truncationMarker = "\n...(truncated)"

// boundarySearchFraction controls how far back from the limit truncateRunes will
// look for a line break to cut on. At 0.2 it searches the final 20% of the
// budget, which keeps the cut near the limit while still landing on a readable
// boundary in almost all real documents.
const boundarySearchFraction = 0.2

// truncateRunes shortens s to at most limit runes, appending truncationMarker
// when it actually truncates. It returns s unchanged when it already fits.
//
// Truncation is rune-based, never byte-based. Slicing a string by bytes can cut
// a multi-byte UTF-8 character in half, producing an invalid string that
// downstream arg-parsers (notably kiro-cli) reject with "invalid UTF-8". All
// prompt-content truncation in this package must go through this function so
// that guarantee holds in one place rather than at each call site.
//
// When a newline exists within the final boundarySearchFraction of the budget,
// the cut is made there instead of mid-sentence, so truncated content ends at a
// line (usually a paragraph or list item) rather than partway through a word.
//
// A non-positive limit yields an empty string.
func truncateRunes(s string, limit int) string {
	if limit <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= limit {
		return s
	}

	head := string(r[:limit])

	// Prefer cutting at a line boundary inside the search window.
	minCut := limit - int(float64(limit)*boundarySearchFraction)
	if idx := strings.LastIndexByte(head, '\n'); idx >= 0 {
		// idx is a byte offset into head; convert to a rune count to compare
		// against minCut so the window check stays rune-based.
		if cutRunes := len([]rune(head[:idx])); cutRunes >= minCut {
			return strings.TrimRight(head[:idx], " \t\n") + truncationMarker
		}
	}

	return head + truncationMarker
}
