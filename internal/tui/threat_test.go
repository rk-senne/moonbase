package tui

import "testing"

func TestComputeThreat(t *testing.T) {
	tests := []struct {
		name      string
		signals   ThreatSignals
		wantLevel string
	}{
		{"no repo", ThreatSignals{NoRepo: true}, "LOW"},
		{"clean tree", ThreatSignals{Dirty: false}, "LOW"},
		{"tiny dirty change", ThreatSignals{Dirty: true, ChangedLines: 10, FilesChanged: 1}, "LOW"},
		{"moderate change", ThreatSignals{Dirty: true, ChangedLines: 150, FilesChanged: 6}, "MEDIUM"},
		{"broad heavy change", ThreatSignals{Dirty: true, ChangedLines: 300, FilesChanged: 8, UntrackedFiles: 3}, "HIGH"},
		{"sensitive forces high", ThreatSignals{Dirty: true, ChangedLines: 4, FilesChanged: 1, SensitiveHits: 1}, "HIGH"},
		{"sensitive + volume critical", ThreatSignals{Dirty: true, ChangedLines: 600, FilesChanged: 10, UntrackedFiles: 5, SensitiveHits: 2}, "CRITICAL"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := computeThreat(tt.signals)
			if got.Name != tt.wantLevel {
				t.Errorf("computeThreat(%+v) level = %q (score %d), want %q", tt.signals, got.Name, got.Score, tt.wantLevel)
			}
			if got.Score < 0 || got.Score > 100 {
				t.Errorf("score out of range: %d", got.Score)
			}
			if got.Reason == "" {
				t.Error("reason should never be empty")
			}
		})
	}
}

func TestComputeThreat_SecurityOverride(t *testing.T) {
	// A tiny change to a sensitive file must never be classified below HIGH,
	// even though its raw volume score is trivial.
	got := computeThreat(ThreatSignals{Dirty: true, ChangedLines: 1, FilesChanged: 1, SensitiveHits: 1})
	if got.Name != "HIGH" && got.Name != "CRITICAL" {
		t.Errorf("sensitive change should be at least HIGH, got %q", got.Name)
	}
}

func TestComputeThreat_Monotonic(t *testing.T) {
	// More change should never lower the score.
	small := computeThreat(ThreatSignals{Dirty: true, ChangedLines: 20, FilesChanged: 2}).Score
	big := computeThreat(ThreatSignals{Dirty: true, ChangedLines: 400, FilesChanged: 9}).Score
	if big < small {
		t.Errorf("expected larger change to score >= smaller: small=%d big=%d", small, big)
	}
}

func TestComputeThreat_Reason(t *testing.T) {
	r := computeThreat(ThreatSignals{Dirty: true, ChangedLines: 42, FilesChanged: 3, SensitiveHits: 1}).Reason
	if r == "" || r == "clean tree" {
		t.Errorf("expected a descriptive reason, got %q", r)
	}
	// sensitive signal must lead the reason
	if got := r[:len("⚠")]; got != "⚠" {
		t.Errorf("expected reason to lead with the sensitive marker, got %q", r)
	}
}

func TestIsSensitivePath(t *testing.T) {
	sensitive := []string{
		"internal/auth/login.go",
		"config/secrets.yaml",
		".env",
		"deploy/Dockerfile",
		".github/workflows/ci.yml",
		"infra/terraform/main.tf",
		"db/migrations/0001_init.sql",
		"src/payment/stripe.ts",
		"certs/server.pem",
	}
	for _, p := range sensitive {
		if !isSensitivePath(p) {
			t.Errorf("expected %q to be sensitive", p)
		}
	}

	benign := []string{
		"internal/tui/app.go",
		"README.md",
		"cmd/moonbase/main.go",
		"docs/design.md",
		"",
	}
	for _, p := range benign {
		if isSensitivePath(p) {
			t.Errorf("expected %q to be benign", p)
		}
	}
}
