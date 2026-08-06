// Package tools provides a curated catalog of developer terminal tools that
// moonbase can help install, plus package-manager-aware construction of the
// install command for the host platform.
//
// SECURITY MODEL
//
// The catalog is a fixed allowlist. Install commands are built ONLY from the
// constant package names in the catalog and the detected system package manager
// — never from user-supplied strings — so there is no command-injection surface.
// The TUI always asks the operator to confirm before an install runs, and the
// install executes through the OS package manager (interactively, so any sudo
// prompt is visible). Tools without a package-manager path for the host are
// reported as "manual" with guidance and are never auto-executed.
package tools

import (
	"os/exec"
	"runtime"
	"strings"
)

// Category classifies a tool.
type Category int

const (
	// Critical marks tools most developers want on any machine.
	Critical Category = iota
	// Cool marks quality-of-life / aesthetic tools.
	Cool
	// Runtime marks language runtimes / package managers (python, node, go, …)
	// and Homebrew itself.
	Runtime
)

// String returns the lower-case category label.
func (c Category) String() string {
	switch c {
	case Cool:
		return "cool"
	case Runtime:
		return "runtime"
	default:
		return "critical"
	}
}

// Tool describes a curated developer tool and how to install it per platform.
// A manager field left empty means "not available via that manager" — in that
// case BuildInstall reports the tool as manual rather than guessing a package
// name (fail-safe: moonbase never runs an install it isn't sure about).
type Tool struct {
	ID          string // binary name used for availability detection (exec.LookPath)
	Display     string // human-friendly name
	Description string
	Category    Category

	Brew     string // Homebrew formula (macOS + Linuxbrew)
	BrewCask bool   // install via `brew install --cask`
	Apt      string // Debian/Ubuntu apt-get package
	Dnf      string // Fedora/RHEL dnf package
	Pacman   string // Arch pacman package

	// Manual holds install guidance shown when no package-manager path exists
	// for the host (e.g. tools distributed via an install script). Never run.
	Manual string

	// MacOnly marks tools that only make sense on macOS (e.g. cmux).
	MacOnly bool

	// Bootstrap marks a tool that installs itself via its own official script
	// rather than a package manager (Homebrew). Handled by BootstrapInstall.
	Bootstrap bool
}

// Catalog returns the curated, ordered tool catalog (critical first, then cool).
func Catalog() []Tool {
	return []Tool{
		// --- Critical ---
		{ID: "git", Display: "git", Description: "Version control — the backbone of every workflow.", Category: Critical,
			Brew: "git", Apt: "git", Dnf: "git", Pacman: "git"},
		{ID: "rg", Display: "ripgrep", Description: "Blazing-fast recursive code search (rg).", Category: Critical,
			Brew: "ripgrep", Apt: "ripgrep", Dnf: "ripgrep", Pacman: "ripgrep"},
		{ID: "fzf", Display: "fzf", Description: "Fuzzy finder for files, history, and anything piped.", Category: Critical,
			Brew: "fzf", Apt: "fzf", Dnf: "fzf", Pacman: "fzf"},
		{ID: "jq", Display: "jq", Description: "Command-line JSON processor.", Category: Critical,
			Brew: "jq", Apt: "jq", Dnf: "jq", Pacman: "jq"},
		{ID: "tmux", Display: "tmux", Description: "Terminal multiplexer — persistent sessions & splits (Linux default).", Category: Critical,
			Brew: "tmux", Apt: "tmux", Dnf: "tmux", Pacman: "tmux"},
		{ID: "nvim", Display: "neovim", Description: "Hyperextensible modal editor.", Category: Critical,
			Brew: "neovim", Apt: "neovim", Dnf: "neovim", Pacman: "neovim"},
		{ID: "lazygit", Display: "lazygit", Description: "Full-featured terminal UI for git.", Category: Critical,
			Brew: "lazygit", Pacman: "lazygit",
			Manual: "See https://github.com/jesseduffield/lazygit#installation"},
		{ID: "gh", Display: "GitHub CLI", Description: "GitHub from the terminal — PRs, issues, releases.", Category: Critical,
			Brew: "gh", Dnf: "gh", Pacman: "github-cli",
			Manual: "apt: add GitHub's apt repo — see https://cli.github.com"},

		// --- Cool but stable ---
		{ID: "btop", Display: "btop", Description: "Gorgeous resource monitor (CPU, mem, net, procs).", Category: Cool,
			Brew: "btop", Apt: "btop", Dnf: "btop", Pacman: "btop"},
		{ID: "bat", Display: "bat", Description: "cat clone with syntax highlighting & git integration.", Category: Cool,
			Brew: "bat", Apt: "bat", Dnf: "bat", Pacman: "bat",
			Manual: "Debian/Ubuntu install the binary as 'batcat'."},
		{ID: "eza", Display: "eza", Description: "Modern, colorful ls replacement.", Category: Cool,
			Brew: "eza", Dnf: "eza", Pacman: "eza",
			Manual: "apt: available on newer releases; see https://github.com/eza-community/eza"},
		{ID: "zoxide", Display: "zoxide", Description: "Smarter cd that learns your habits.", Category: Cool,
			Brew: "zoxide", Apt: "zoxide", Dnf: "zoxide", Pacman: "zoxide"},
		{ID: "delta", Display: "git-delta", Description: "Syntax-highlighting pager for git diffs.", Category: Cool,
			Brew: "git-delta", Dnf: "git-delta", Pacman: "git-delta",
			Manual: "apt: see https://github.com/dandavison/delta#installation"},
		{ID: "fish", Display: "fish", Description: "Friendly interactive shell with great defaults.", Category: Cool,
			Brew: "fish", Apt: "fish", Dnf: "fish", Pacman: "fish"},
		{ID: "starship", Display: "starship", Description: "Fast, minimal, customizable cross-shell prompt.", Category: Cool,
			Brew: "starship", Pacman: "starship",
			Manual: "1. Download: curl -sS https://starship.rs/install.sh -o install.sh\n2. Verify: shasum -a 256 install.sh (compare with https://starship.rs/install.sh.sha256)\n3. Run: sh install.sh"},
		{ID: "oh-my-posh", Display: "oh-my-posh", Description: "Prompt theme engine for any shell.", Category: Cool,
			Brew: "oh-my-posh",
			Manual: "1. Download: curl -s https://ohmyposh.dev/install.sh -o install.sh\n2. Verify: review the script or check SHA256 at https://ohmyposh.dev/docs/installation/linux\n3. Run: bash install.sh"},
		{ID: "lazydocker", Display: "lazydocker", Description: "Terminal UI for Docker & docker-compose.", Category: Cool,
			Brew: "lazydocker",
			Manual: "See https://github.com/jesseduffield/lazydocker#installation"},
		{ID: "cmux", Display: "cmux", Description: "macOS terminal built for AI coding agents (moonbase uses it in place of tmux).", Category: Cool,
			MacOnly: true,
			Manual:  "macOS only — see https://github.com/manaflow-ai/cmux"},
	}
}

// IsInstalled reports whether the tool's binary is resolvable on PATH.
func IsInstalled(id string) bool {
	_, err := exec.LookPath(id)
	return err == nil
}

// ToolsForOS returns the recommended dev tools installable on the given OS, in
// DevCatalog order. macOS gets Homebrew-installable tools (and mac-only ones like
// cmux); Linux gets tools with an apt/dnf/pacman package (or the Homebrew
// bootstrap for Linuxbrew), excluding mac-only tools.
func ToolsForOS(goos string) []Tool {
	var out []Tool
	for _, t := range DevCatalog() {
		if goos == "darwin" {
			if t.Bootstrap || t.MacOnly || t.Brew != "" {
				out = append(out, t)
			}
			continue
		}
		// Linux and others.
		if t.MacOnly {
			continue
		}
		if t.Bootstrap || t.Apt != "" || t.Dnf != "" || t.Pacman != "" {
			out = append(out, t)
		}
	}
	return out
}

// InstallAllPlan builds a single package-manager command that installs every
// not-yet-installed tool in list that has a package for mgr, in one invocation
// (e.g. `brew install a b c`). It returns the plan, the display names of tools
// that were skipped (already installed, bootstrap, or manual/no-package), and
// ok=false when there is nothing installable to run.
func InstallAllPlan(list []Tool, mgr Manager) (InstallPlan, []string, bool) {
	return installAllPlan(list, mgr, IsInstalled)
}

// installAllPlan is the injectable core of InstallAllPlan (installed reports
// whether a tool's binary is already present — real PATH lookup in production,
// a fake in tests).
func installAllPlan(list []Tool, mgr Manager, installed func(string) bool) (InstallPlan, []string, bool) {
	var pkgs, skipped []string
	for _, t := range list {
		if installed(t.ID) {
			continue // already present — nothing to do
		}
		if t.Bootstrap {
			skipped = append(skipped, t.Display+" (bootstrap installer)")
			continue
		}
		pkg := t.pkgFor(mgr.Name)
		if pkg == "" {
			skipped = append(skipped, t.Display+" (manual)")
			continue
		}
		pkgs = append(pkgs, pkg)
	}
	if len(pkgs) == 0 {
		return InstallPlan{}, skipped, false
	}

	args := append([]string{}, mgr.baseArgs...)
	args = append(args, pkgs...)
	if mgr.NeedsSudo {
		return InstallPlan{
			Manager: mgr.Name,
			Bin:     "sudo",
			Args:    append([]string{mgr.Bin}, args...),
			Sudo:    true,
		}, skipped, true
	}
	return InstallPlan{Manager: mgr.Name, Bin: mgr.Bin, Args: args, Sudo: false}, skipped, true
}

// DevCatalog is the broader catalog surfaced in the Settings view: Homebrew
// itself, common language runtimes, and the full terminal-tool catalog. Ordered
// bootstrap → runtimes → critical → cool.
func DevCatalog() []Tool {
	head := []Tool{
		{ID: "brew", Display: "Homebrew", Description: "The package manager for macOS/Linux — prerequisite for everything below.", Category: Runtime, Bootstrap: true},

		{ID: "python3", Display: "Python", Description: "Python 3 interpreter + pip.", Category: Runtime,
			Brew: "python", Apt: "python3", Dnf: "python3", Pacman: "python"},
		{ID: "node", Display: "Node / npm", Description: "Node.js runtime with the npm package manager.", Category: Runtime,
			Brew: "node", Apt: "nodejs", Dnf: "nodejs", Pacman: "nodejs"},
		{ID: "go", Display: "Go", Description: "The Go toolchain (compiler + modules).", Category: Runtime,
			Brew: "go", Apt: "golang-go", Dnf: "golang", Pacman: "go"},
		{ID: "rustc", Display: "Rust", Description: "Rust compiler + cargo.", Category: Runtime,
			Brew: "rust", Apt: "rustc", Dnf: "rust", Pacman: "rust",
			Manual: "Recommended: rustup — https://rustup.rs (review the script first)"},
		{ID: "ruby", Display: "Ruby", Description: "Ruby interpreter + gem.", Category: Runtime,
			Brew: "ruby", Apt: "ruby", Dnf: "ruby", Pacman: "ruby"},
		{ID: "deno", Display: "Deno", Description: "Secure TypeScript/JavaScript runtime.", Category: Runtime,
			Brew: "deno", Pacman: "deno",
			Manual: "See https://docs.deno.com/runtime/manual/getting_started/installation"},
		{ID: "bun", Display: "Bun", Description: "Fast all-in-one JavaScript runtime + package manager.", Category: Runtime,
			Brew: "bun",
			Manual: "See https://bun.sh/docs/installation"},
		{ID: "java", Display: "OpenJDK", Description: "Java Development Kit (OpenJDK).", Category: Runtime,
			Brew: "openjdk", Apt: "default-jdk", Dnf: "java-latest-openjdk", Pacman: "jdk-openjdk"},
	}
	return append(head, Catalog()...)
}

// HomebrewInstallPlan returns the official Homebrew bootstrap installer. This is
// the only supported way to install Homebrew itself (it *is* the package
// manager). It is run ONLY after an explicit y/n confirmation that shows this
// exact command — the installer is the official one, fetched over HTTPS from
// Homebrew's repository.
func HomebrewInstallPlan() InstallPlan {
	return InstallPlan{
		Manager: "homebrew-installer",
		Bin:     "/bin/bash",
		Args:    []string{"-c", `NONINTERACTIVE=1 /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"`},
		Sudo:    false,
	}
}

// Manager identifies a system package manager and how to build its install
// command.
type Manager struct {
	Name      string   // "brew", "apt", "dnf", "pacman"
	Bin       string   // resolved executable path
	baseArgs  []string // arguments before the package name
	NeedsSudo bool     // whether the install requires sudo
}

// DetectManager detects the host's package manager.
func DetectManager() (Manager, bool) {
	return detectManager(runtime.GOOS, exec.LookPath)
}

// detectManager is the injectable core of DetectManager (testable across OSes).
// macOS uses Homebrew. Linux tries Homebrew (Linuxbrew, no sudo) first, then the
// common system managers in order.
func detectManager(goos string, look func(string) (string, error)) (Manager, bool) {
	brew := func() (Manager, bool) {
		if bin, err := look("brew"); err == nil {
			return Manager{Name: "brew", Bin: bin, baseArgs: []string{"install"}, NeedsSudo: false}, true
		}
		return Manager{}, false
	}

	if goos == "darwin" {
		return brew()
	}

	if m, ok := brew(); ok { // Linuxbrew
		return m, true
	}
	if bin, err := look("apt-get"); err == nil {
		return Manager{Name: "apt", Bin: bin, baseArgs: []string{"install", "-y"}, NeedsSudo: true}, true
	}
	if bin, err := look("dnf"); err == nil {
		return Manager{Name: "dnf", Bin: bin, baseArgs: []string{"install", "-y"}, NeedsSudo: true}, true
	}
	if bin, err := look("pacman"); err == nil {
		return Manager{Name: "pacman", Bin: bin, baseArgs: []string{"-S", "--noconfirm"}, NeedsSudo: true}, true
	}
	return Manager{}, false
}

// pkgFor returns the package name for the given manager, or "" when the tool has
// no known package for it.
func (t Tool) pkgFor(mgr string) string {
	switch mgr {
	case "brew":
		return t.Brew
	case "apt":
		return t.Apt
	case "dnf":
		return t.Dnf
	case "pacman":
		return t.Pacman
	}
	return ""
}

// InstallableWith reports whether the tool has a package for mgr (and is thus
// eligible for a batch "install all"). Bootstrap and manual-only tools return
// false.
func (t Tool) InstallableWith(mgr Manager) bool {
	return !t.Bootstrap && t.pkgFor(mgr.Name) != ""
}

// InstallPlan is a fully-resolved, ready-to-run install command.
type InstallPlan struct {
	Manager string   // package manager name
	Bin     string   // executable to exec (the manager, or "sudo")
	Args    []string // argument vector
	Sudo    bool     // whether the command uses sudo
}

// Display renders the plan as a copy-pasteable command string (for the UI).
func (p InstallPlan) Display() string {
	return strings.TrimSpace(p.Bin + " " + strings.Join(p.Args, " "))
}

// BuildInstall constructs the install command for tool t using manager mgr.
// It returns ok=false with a human-readable reason when there is no
// package-manager path (the caller should then surface t.Manual). Commands are
// assembled solely from allowlisted constants — never from user input.
func BuildInstall(t Tool, mgr Manager) (InstallPlan, bool, string) {
	pkg := t.pkgFor(mgr.Name)
	if pkg == "" {
		return InstallPlan{}, false, manualReason(t)
	}

	var args []string
	if mgr.Name == "brew" && t.BrewCask {
		args = []string{"install", "--cask"}
	} else {
		args = append(args, mgr.baseArgs...)
	}
	args = append(args, pkg)

	if mgr.NeedsSudo {
		return InstallPlan{
			Manager: mgr.Name,
			Bin:     "sudo",
			Args:    append([]string{mgr.Bin}, args...),
			Sudo:    true,
		}, true, ""
	}
	return InstallPlan{Manager: mgr.Name, Bin: mgr.Bin, Args: args, Sudo: false}, true, ""
}

// manualReason returns the tool's manual guidance, or a generic message.
func manualReason(t Tool) string {
	if t.Manual != "" {
		return t.Manual
	}
	return "No package-manager path for this host — install manually."
}
