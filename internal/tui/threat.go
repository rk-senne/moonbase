package tui

import (
	"fmt"
	"strings"
)

// ThreatSignals are portable, git-derived inputs describing the current
// working-tree state of whatever project moonbase is launched in. They feed
// computeThreat to produce a composite "threat level" — a real reflection of
// how much uncommitted risk is currently in flight, for moonbase itself or any
// other repository.
type ThreatSignals struct {
	ChangedLines   int  // insertions+deletions vs HEAD (staged + unstaged)
	FilesChanged   int  // tracked files with modifications (blast radius)
	UntrackedFiles int  // new, not-yet-tracked files
	SensitiveHits  int  // changed/new files matching security-sensitive patterns
	Dirty          bool // working tree has any uncommitted change
	NoRepo         bool // not a git repository (or git unavailable)
}

// ThreatLevel is the computed classification shown by the threat gauge.
type ThreatLevel struct {
	Name   string // LOW, MEDIUM, HIGH, CRITICAL
	Score  int    // 0..100 composite risk score
	Reason string // short, human-readable explanation of the dominant signals
}

// Threat scoring weights and thresholds. Documented constants — no magic
// numbers. Volume and breadth raise the score gradually; sensitive files are
// weighted heavily and additionally force at least HIGH (see computeThreat).
const (
	threatLinesPerPoint = 15 // 1 point per N changed lines
	threatLinesCap      = 40 // volume alone can never dominate outright
	threatFilePoints    = 3  // points per changed file (blast radius)
	threatFilesCap      = 25
	threatUntrackedPts  = 2 // points per untracked file
	threatUntrackedCap  = 10
	threatSensitivePts  = 20 // points per sensitive file (heavy)
	threatSensitiveCap  = 40

	threatCritical = 75 // score >= 75 -> CRITICAL
	threatHigh     = 50 // score >= 50 -> HIGH
	threatMedium   = 25 // score >= 25 -> MEDIUM
)

// computeThreat maps raw working-tree signals to a composite threat level.
// It mirrors moonbase's own risk-gate philosophy: change volume and breadth
// raise the level, and touching security- or infra-sensitive files forces at
// least HIGH — the same class of signal that triggers Numbuh 274 (security)
// and Numbuh 362 (infra) in the pipeline. The function is pure and portable:
// the same signals produce the same level in any repository.
func computeThreat(s ThreatSignals) ThreatLevel {
	if s.NoRepo {
		return ThreatLevel{Name: "LOW", Score: 0, Reason: "no git repo"}
	}
	if !s.Dirty && s.ChangedLines == 0 && s.FilesChanged == 0 && s.UntrackedFiles == 0 {
		return ThreatLevel{Name: "LOW", Score: 0, Reason: "clean tree"}
	}

	score := min(s.ChangedLines/threatLinesPerPoint, threatLinesCap)
	score += min(s.FilesChanged*threatFilePoints, threatFilesCap)
	score += min(s.UntrackedFiles*threatUntrackedPts, threatUntrackedCap)
	score += min(s.SensitiveHits*threatSensitivePts, threatSensitiveCap)
	score = min(score, 100)

	name := "LOW"
	switch {
	case score >= threatCritical:
		name = "CRITICAL"
	case score >= threatHigh:
		name = "HIGH"
	case score >= threatMedium:
		name = "MEDIUM"
	}

	// Security override: any sensitive file in flight is at least HIGH,
	// regardless of raw volume — a two-line change to an auth file is riskier
	// than a 400-line change to test fixtures.
	if s.SensitiveHits > 0 && (name == "LOW" || name == "MEDIUM") {
		name = "HIGH"
	}

	return ThreatLevel{Name: name, Score: score, Reason: threatReason(s)}
}

// threatReason builds a concise explanation of the dominant signals, leading
// with the security signal when present.
func threatReason(s ThreatSignals) string {
	var parts []string
	if s.SensitiveHits > 0 {
		parts = append(parts, fmt.Sprintf("⚠ %d sensitive", s.SensitiveHits))
	}
	if s.FilesChanged > 0 {
		parts = append(parts, fmt.Sprintf("%d files", s.FilesChanged))
	}
	if s.ChangedLines > 0 {
		parts = append(parts, fmt.Sprintf("%d lines", s.ChangedLines))
	}
	if s.UntrackedFiles > 0 {
		parts = append(parts, fmt.Sprintf("%d new", s.UntrackedFiles))
	}
	if len(parts) == 0 {
		return "clean tree"
	}
	if len(parts) > 3 { // keep the line short
		parts = parts[:3]
	}
	return strings.Join(parts, " · ")
}

// sensitiveTokens are lowercased substrings that mark a changed path as
// security- or infra-sensitive. This is a heuristic (not a security control):
// it errs toward caution to match moonbase's risk-gate triggers, and is
// portable across languages and stacks.
var sensitiveTokens = []string{
	// secrets & auth
	"auth", "oauth", "login", "session", "jwt", "saml",
	"secret", "credential", "password", "passwd", "token",
	"apikey", "api_key", "security", "crypto", "vault", "kms",
	"iam", "rbac",
	// key material (extensions / well-known filenames)
	".env", ".pem", ".key", "id_rsa", "keystore", ".p12", ".pfx",
	// infra / deploy / CI
	"dockerfile", "docker-compose", ".github/workflows", ".gitlab-ci",
	".circleci", "jenkinsfile", "terraform", ".tf", "kubernetes",
	"/k8s/", "helm", "deployment.yaml", "ingress",
	// data & money
	"/migrations/", "payment", "billing", "stripe",
}

// isSensitivePath reports whether a changed path looks security- or
// infra-sensitive. Matching is case-insensitive substring matching against
// sensitiveTokens.
func isSensitivePath(path string) bool {
	p := strings.ToLower(strings.TrimSpace(path))
	if p == "" {
		return false
	}
	for _, tok := range sensitiveTokens {
		if strings.Contains(p, tok) {
			return true
		}
	}
	return false
}
