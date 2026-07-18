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

type termOutputMsg struct {
	cmd    string
	output string
}

type toolExitMsg struct{ tool string }

type prCreatedMsg struct{ output string }
