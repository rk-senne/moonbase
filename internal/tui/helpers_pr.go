package tui

import (
	"fmt"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// --- GitHub PR ---

func (a App) createPR() tea.Cmd {
	return func() tea.Msg {
		if _, err := exec.LookPath("gh"); err != nil {
			return prCreatedMsg{output: "gh CLI not found. Install: https://cli.github.com"}
		}
		branch, _ := exec.Command("git", "branch", "--show-current").Output()
		branchName := strings.TrimSpace(string(branch))
		if branchName == "main" || branchName == "master" {
			return prCreatedMsg{output: "Cannot create PR from main/master. Switch to a feature branch."}
		}
		out, err := exec.Command("gh", "pr", "create", "--fill").CombinedOutput()
		if err != nil {
			return prCreatedMsg{output: fmt.Sprintf("PR failed: %s", string(out))}
		}
		return prCreatedMsg{output: strings.TrimSpace(string(out))}
	}
}
