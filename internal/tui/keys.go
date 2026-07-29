package tui

import "github.com/charmbracelet/bubbles/key"

// KeyMap defines all key bindings for the TUI. It is the single source of truth
// for what keys do in each view. Implements help.KeyMap for generated help.
type KeyMap struct {
	// Navigation
	Up    key.Binding
	Down  key.Binding
	Enter key.Binding
	Back  key.Binding
	Tab   key.Binding

	// Missions / Pipeline
	NewMission key.Binding
	NextPhase  key.Binding
	RetryPhase key.Binding
	SkipPhase  key.Binding

	// Views
	Help         key.Binding
	Protocol     key.Binding
	CycleTheme   key.Binding
	OpenComms    key.Binding
	Search       key.Binding
	History      key.Binding
	Docs         key.Binding
	Projects     key.Binding
	CopyPrompt   key.Binding
	SpawnHook    key.Binding
	JumpToAgent  key.Binding

	// Tools
	LaunchLazygit key.Binding
	LaunchBtop    key.Binding
	LaunchNvim    key.Binding
	LaunchCmux    key.Binding
	LaunchFish    key.Binding

	// System
	Quit          key.Binding
	GitDiff       key.Binding
	GitStatus     key.Binding
	ToggleWatcher key.Binding
	CreatePR      key.Binding

	// Comms
	SendMessage   key.Binding
	AttachFile    key.Binding
	SnippetPicker key.Binding
	CommsQuit     key.Binding

	// Search mode
	SearchConfirm key.Binding
	SearchCancel  key.Binding

	// Terminal mode
	TerminalEsc       key.Binding
	TerminalToBrowser key.Binding
	TerminalSubmit    key.Binding

	// File browser
	BrowserToTerminal key.Binding
	BrowserUp         key.Binding
	BrowserDown       key.Binding
	BrowserEnter      key.Binding
	BrowserBack       key.Binding
	BrowserEdit       key.Binding
	BrowserRefresh    key.Binding
	BrowserEsc        key.Binding

	// Snippet picker
	SnippetUp      key.Binding
	SnippetDown    key.Binding
	SnippetConfirm key.Binding
	SnippetCancel  key.Binding

	// Context file input
	ContextConfirm key.Binding
	ContextCancel  key.Binding

	// Docs view
	DocsPageDown key.Binding
	DocsPageUp   key.Binding
}

// DefaultKeyMap returns the standard key bindings for the TUI.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		// Navigation
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "open / deploy"),
		),
		Back: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "back / close"),
		),
		Tab: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "cycle panel focus"),
		),

		// Missions / Pipeline
		NewMission: key.NewBinding(
			key.WithKeys("m"),
			key.WithHelp("m", "new mission"),
		),
		NextPhase: key.NewBinding(
			key.WithKeys("n"),
			key.WithHelp("n", "next phase"),
		),
		RetryPhase: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "retry phase"),
		),
		SkipPhase: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "skip phase"),
		),

		// Views
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "operations manual"),
		),
		Protocol: key.NewBinding(
			key.WithKeys("F1"),
			key.WithHelp("F1", "protocol view"),
		),
		CycleTheme: key.NewBinding(
			key.WithKeys("T"),
			key.WithHelp("T", "cycle theme"),
		),
		OpenComms: key.NewBinding(
			key.WithKeys("C"),
			key.WithHelp("C", "open COMMS"),
		),
		Search: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "search operatives"),
		),
		History: key.NewBinding(
			key.WithKeys("H"),
			key.WithHelp("H", "mission history"),
		),
		Docs: key.NewBinding(
			key.WithKeys("W"),
			key.WithHelp("W", "document viewer"),
		),
		Projects: key.NewBinding(
			key.WithKeys("p"),
			key.WithHelp("p", "project navigator"),
		),
		CopyPrompt: key.NewBinding(
			key.WithKeys("c"),
			key.WithHelp("c", "copy prompt"),
		),
		SpawnHook: key.NewBinding(
			key.WithKeys("t"),
			key.WithHelp("t", "run spawn hook"),
		),
		JumpToAgent: key.NewBinding(
			key.WithKeys("0", "1", "2", "3", "4", "5", "6", "7", "8", "9"),
			key.WithHelp("0-9", "jump to operative"),
		),

		// Tools
		LaunchLazygit: key.NewBinding(
			key.WithKeys("L"),
			key.WithHelp("L", "lazygit"),
		),
		LaunchBtop: key.NewBinding(
			key.WithKeys("B"),
			key.WithHelp("B", "btop"),
		),
		LaunchNvim: key.NewBinding(
			key.WithKeys("V"),
			key.WithHelp("V", "nvim"),
		),
		LaunchCmux: key.NewBinding(
			key.WithKeys("M"),
			key.WithHelp("M", "cmux/tmux"),
		),
		LaunchFish: key.NewBinding(
			key.WithKeys("F"),
			key.WithHelp("F", "fish shell"),
		),

		// System
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "disconnect"),
		),
		GitDiff: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "git diff"),
		),
		GitStatus: key.NewBinding(
			key.WithKeys("g"),
			key.WithHelp("g", "git status"),
		),
		ToggleWatcher: key.NewBinding(
			key.WithKeys("w"),
			key.WithHelp("w", "toggle file watcher"),
		),
		CreatePR: key.NewBinding(
			key.WithKeys("P"),
			key.WithHelp("P", "create PR"),
		),

		// Comms
		SendMessage: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "send message"),
		),
		AttachFile: key.NewBinding(
			key.WithKeys("ctrl+f"),
			key.WithHelp("ctrl+f", "attach file"),
		),
		SnippetPicker: key.NewBinding(
			key.WithKeys("ctrl+s"),
			key.WithHelp("ctrl+s", "snippet picker"),
		),
		CommsQuit: key.NewBinding(
			key.WithKeys("ctrl+c"),
			key.WithHelp("ctrl+c", "quit"),
		),

		// Search mode
		SearchConfirm: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "confirm search"),
		),
		SearchCancel: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "cancel search"),
		),

		// Terminal mode
		TerminalEsc: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "exit terminal"),
		),
		TerminalToBrowser: key.NewBinding(
			key.WithKeys("`"),
			key.WithHelp("`", "file browser"),
		),
		TerminalSubmit: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "execute command"),
		),

		// File browser
		BrowserToTerminal: key.NewBinding(
			key.WithKeys("`"),
			key.WithHelp("`", "terminal"),
		),
		BrowserUp: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		BrowserDown: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		BrowserEnter: key.NewBinding(
			key.WithKeys("enter", "l"),
			key.WithHelp("enter/l", "open"),
		),
		BrowserBack: key.NewBinding(
			key.WithKeys("backspace", "h"),
			key.WithHelp("bs/h", "back"),
		),
		BrowserEdit: key.NewBinding(
			key.WithKeys("e"),
			key.WithHelp("e", "edit file"),
		),
		BrowserRefresh: key.NewBinding(
			key.WithKeys("."),
			key.WithHelp(".", "refresh"),
		),
		BrowserEsc: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "close browser"),
		),

		// Snippet picker
		SnippetUp: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		SnippetDown: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		SnippetConfirm: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "select snippet"),
		),
		SnippetCancel: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "cancel"),
		),

		// Context file input
		ContextConfirm: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "attach"),
		),
		ContextCancel: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "cancel"),
		),

		// Docs view
		DocsPageDown: key.NewBinding(
			key.WithKeys("pgdown", " "),
			key.WithHelp("pgdn/space", "page down"),
		),
		DocsPageUp: key.NewBinding(
			key.WithKeys("pgup"),
			key.WithHelp("pgup", "page up"),
		),
	}
}

// ShortHelp returns the minimal key bindings shown in the footer.
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{
		k.Up, k.Down, k.Enter, k.Back, k.Help, k.Quit,
	}
}

// FullHelp returns all key bindings grouped by category for the help view.
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		// Navigation
		{k.Up, k.Down, k.Enter, k.Back, k.Tab, k.Search, k.JumpToAgent},
		// Missions
		{k.NewMission, k.NextPhase, k.RetryPhase, k.SkipPhase, k.History},
		// Views
		{k.Help, k.Protocol, k.Projects, k.Docs, k.OpenComms, k.CycleTheme},
		// Tools
		{k.LaunchLazygit, k.LaunchBtop, k.LaunchNvim, k.LaunchCmux, k.LaunchFish},
		// Comms
		{k.SendMessage, k.AttachFile, k.SnippetPicker},
		// System
		{k.Quit, k.GitDiff, k.GitStatus, k.ToggleWatcher, k.CreatePR},
	}
}

// keysFor returns the key bindings valid for the given view and sub-mode state.
// This is the single source for per-view contextual footers.
func (k KeyMap) keysFor(view View, searching bool, termActive bool, browsing bool) []key.Binding {
	if searching {
		return []key.Binding{k.SearchConfirm, k.SearchCancel}
	}

	if view == ViewDashboard && termActive {
		return []key.Binding{k.TerminalSubmit, k.TerminalToBrowser, k.TerminalEsc}
	}

	if view == ViewDashboard && browsing {
		return []key.Binding{k.BrowserUp, k.BrowserDown, k.BrowserEnter, k.BrowserBack, k.BrowserEdit, k.BrowserToTerminal, k.BrowserEsc}
	}

	switch view {
	case ViewDashboard:
		return []key.Binding{k.Help, k.Up, k.Down, k.Enter, k.NewMission, k.Projects, k.Docs, k.SpawnHook, k.Quit}
	case ViewDossier:
		return []key.Binding{k.Enter, k.CopyPrompt, k.SpawnHook, k.Up, k.Down, k.Back}
	case ViewPipeline:
		return []key.Binding{k.NextPhase, k.RetryPhase, k.SkipPhase, k.Back}
	case ViewHelp:
		return []key.Binding{k.Back}
	case ViewMission:
		return []key.Binding{k.Enter, k.Back}
	case ViewComms:
		return []key.Binding{k.SendMessage, k.AttachFile, k.SnippetPicker, k.Back}
	case ViewHistory:
		return []key.Binding{k.Back}
	case ViewDocs:
		return []key.Binding{k.Up, k.Down, k.Enter, k.DocsPageDown, k.DocsPageUp, k.Back}
	case ViewProjects:
		return []key.Binding{k.Up, k.Down, k.Enter, k.Back}
	case ViewProtocol:
		return []key.Binding{k.Back, k.Protocol}
	default:
		return k.ShortHelp()
	}
}
