package tui

// SystemModel holds detected environment state — git branch/cleanliness,
// running docker containers, and the derived working-tree threat signals —
// that the dashboard's SYSTEM and THREAT LEVEL panels render. Extracted from
// App to keep the top-level struct focused on orchestration.
type SystemModel struct {
	Branch         string
	Clean          bool
	Docker         int
	ChangedLines   int // insertions+deletions vs HEAD (staged + unstaged)
	FilesChanged   int
	UntrackedFiles int
	SensitiveHits  int
	NoRepo         bool
}

// GitStatus returns a short clean/dirty indicator for the git panel.
func (m SystemModel) GitStatus() string {
	if m.Clean {
		return "✓ clean"
	}
	return "● dirty"
}

// Signals assembles the working-tree signals for threat computation.
func (m SystemModel) Signals() ThreatSignals {
	return ThreatSignals{
		ChangedLines:   m.ChangedLines,
		FilesChanged:   m.FilesChanged,
		UntrackedFiles: m.UntrackedFiles,
		SensitiveHits:  m.SensitiveHits,
		Dirty:          !m.Clean,
		NoRepo:         m.NoRepo,
	}
}

// Threat returns the composite threat level for the current working tree.
func (m SystemModel) Threat() ThreatLevel {
	return computeThreat(m.Signals())
}
