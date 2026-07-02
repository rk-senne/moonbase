package pipeline

import (
	"strings"
)

// RiskLevel represents the QA risk assessment.
type RiskLevel string

const (
	RiskLow      RiskLevel = "LOW"
	RiskMedium   RiskLevel = "MEDIUM"
	RiskHigh     RiskLevel = "HIGH"
	RiskCritical RiskLevel = "CRITICAL"
	RiskUnknown  RiskLevel = "UNKNOWN"
)

// RiskRouting represents where the pipeline should go based on risk assessment.
type RiskRouting struct {
	Level       RiskLevel // The assessed risk level
	TargetPhase int       // Which phase to route to (0 = stop pipeline)
	Action      string    // Human-readable action description
}

// ParseRiskGate parses QA output (Numbuh 4) to extract the risk verdict.
// It looks for "## Verdict" section or "RISK" indicators in the output.
func ParseRiskGate(qaOutput string) RiskRouting {
	level := extractRiskLevel(qaOutput)

	switch level {
	case RiskLow:
		return RiskRouting{
			Level:       RiskLow,
			TargetPhase: 5,
			Action:      "Proceed to review (Numbuh 5)",
		}
	case RiskMedium:
		return RiskRouting{
			Level:       RiskMedium,
			TargetPhase: 3,
			Action:      "Back to implementation (Numbuh 3) for fixes",
		}
	case RiskHigh:
		return RiskRouting{
			Level:       RiskHigh,
			TargetPhase: 2,
			Action:      "Back to design (Numbuh 2) for redesign",
		}
	case RiskCritical:
		return RiskRouting{
			Level:       RiskCritical,
			TargetPhase: 0,
			Action:      "STOP — Critical risk, escalate to human",
		}
	default:
		// Unknown = treat as MEDIUM (cautious)
		return RiskRouting{
			Level:       RiskMedium,
			TargetPhase: 3,
			Action:      "Risk level unclear — treating as MEDIUM, back to implementation",
		}
	}
}

// extractRiskLevel parses the risk level from QA output.
// Looks for patterns like:
//   - "## Verdict\nLOW"
//   - "## Verdict\n\nLOW"
//   - "RISK_LEVEL: LOW"
//   - "Risk: LOW"
//   - "**Verdict:** LOW"
func extractRiskLevel(output string) RiskLevel {
	lines := strings.Split(output, "\n")

	// Strategy 1: Look for "## Verdict" header followed by the level
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "## Verdict" || trimmed == "**Verdict**" || trimmed == "### Verdict" {
			// Check next non-empty lines for the level
			for j := i + 1; j < len(lines) && j <= i+3; j++ {
				level := matchRiskLevel(lines[j])
				if level != RiskUnknown {
					return level
				}
			}
		}
	}

	// Strategy 2: Look for "Verdict: LEVEL" or "Risk: LEVEL" patterns
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Match patterns like "Verdict: LOW", "RISK_LEVEL: HIGH", "Risk: MEDIUM"
		prefixes := []string{
			"verdict:", "risk:", "risk_level:", "risk level:",
			"**verdict:**", "**risk:**", "**risk_level:**",
		}
		for _, prefix := range prefixes {
			lower := strings.ToLower(trimmed)
			if strings.HasPrefix(lower, prefix) {
				rest := strings.TrimSpace(trimmed[len(prefix):])
				level := matchRiskLevel(rest)
				if level != RiskUnknown {
					return level
				}
			}
		}
	}

	// Strategy 3: Look for standalone risk level words (less reliable)
	// Only match if it appears to be a verdict line (short, uppercase)
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if len(trimmed) <= 20 {
			level := matchRiskLevel(trimmed)
			if level != RiskUnknown {
				return level
			}
		}
	}

	return RiskUnknown
}

// matchRiskLevel checks if a string contains a risk level.
func matchRiskLevel(s string) RiskLevel {
	upper := strings.ToUpper(strings.TrimSpace(s))

	// Remove common formatting
	upper = strings.Trim(upper, "*`_")
	upper = strings.TrimSpace(upper)

	switch {
	case strings.Contains(upper, "CRITICAL"):
		return RiskCritical
	case strings.Contains(upper, "HIGH"):
		return RiskHigh
	case strings.Contains(upper, "MEDIUM"):
		return RiskMedium
	case strings.Contains(upper, "LOW"):
		return RiskLow
	}

	return RiskUnknown
}
