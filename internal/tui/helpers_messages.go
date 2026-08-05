package tui

type copyDoneMsg struct{ agent string }
type deployDoneMsg struct{ agent string }
type gitOutputMsg struct{ output string }
type spawnHookMsg struct {
	agent  string
	output string
}
type streamChunkMsg struct {
	text string
	done bool
	err  error
}

// PhaseChunkMsg carries a single incremental text chunk from a streaming
// pipeline phase. Distinct from the COMMS streamChunkMsg — they are routed
// separately in the Update loop (AC-4.1).
type PhaseChunkMsg struct {
	Phase int
	Text  string
}

type termOutputMsg struct {
	cmd    string
	output string
}

type toolExitMsg struct{ tool string }

type prCreatedMsg struct{ output string }
