package tui

import "unicode"

// fuzzyMatch performs a subsequence fuzzy match of pattern against target.
// It returns matched=true if every rune of pattern appears in target in order
// (case-insensitive). Score reflects match quality using a Sublime-Text-inspired
// heuristic.
//
// Scoring rewards:
//   - Contiguous matches: bonus for each consecutive matched rune (+5 per consecutive)
//   - Start-of-string: bonus if the first pattern rune matches the first target rune (+10)
//   - Separator boundary: bonus when a match occurs right after '-', '_', '/', or space (+8)
//   - camelCase boundary: bonus when a match is an uppercase char after a lowercase (+8)
//   - Earlier position: penalty for leading gap before first match (-1 per char)
//   - Base score: +1 per matched rune
//
// Empty pattern always matches with score 0. O(len(target)) greedy scan.
func fuzzyMatch(pattern, target string) (score int, matched bool) {
	if pattern == "" {
		return 0, true
	}

	patternRunes := []rune(pattern)
	targetRunes := []rune(target)

	pi := 0 // index into pattern runes
	lastMatchIdx := -1
	firstMatchIdx := -1

	const (
		bonusConsecutive = 5
		bonusStart       = 10
		bonusSeparator   = 8
		bonusCamelCase   = 8
		penaltyLeadGap   = -1
		scorePerMatch    = 1
	)

	for ti := 0; ti < len(targetRunes) && pi < len(patternRunes); ti++ {
		tc := unicode.ToLower(targetRunes[ti])
		pc := unicode.ToLower(patternRunes[pi])

		if tc == pc {
			score += scorePerMatch

			if firstMatchIdx == -1 {
				firstMatchIdx = ti
			}

			// Bonus: start of target
			if ti == 0 {
				score += bonusStart
			}

			// Bonus: consecutive match
			if lastMatchIdx >= 0 && ti == lastMatchIdx+1 {
				score += bonusConsecutive
			}

			// Bonus: after separator
			if ti > 0 {
				prev := targetRunes[ti-1]
				if prev == '-' || prev == '_' || prev == '/' || prev == ' ' {
					score += bonusSeparator
				}
			}

			// Bonus: camelCase boundary (lowercase→uppercase)
			if ti > 0 && unicode.IsUpper(targetRunes[ti]) && unicode.IsLower(targetRunes[ti-1]) {
				score += bonusCamelCase
			}

			lastMatchIdx = ti
			pi++
		}
	}

	// All pattern runes must have been consumed
	if pi < len(patternRunes) {
		return 0, false
	}

	// Penalty: leading gap before first match
	if firstMatchIdx > 0 {
		score += firstMatchIdx * penaltyLeadGap
	}

	return score, true
}
