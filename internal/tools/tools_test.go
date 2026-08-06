package tools

import (
	"os/exec"
	"strings"
	"testing"
)

func lookProvides(names ...string) func(string) (string, error) {
	set := make(map[string]bool, len(names))
	for _, n := range names {
		set[n] = true
	}
	return func(name string) (string, error) {
		if set[name] {
			return "/usr/bin/" + name, nil
		}
		return "", exec.ErrNotFound
	}
}

func TestCatalog_Integrity(t *testing.T) {
	cat := Catalog()
	if len(cat) == 0 {
		t.Fatal("catalog is empty")
	}
	seen := map[string]bool{}
	var haveCritical, haveCool, haveOhMyPosh bool
	for _, tool := range cat {
		if tool.ID == "" || tool.Display == "" || tool.Description == "" {
			t.Errorf("tool %+v missing required display fields", tool)
		}
		if tool.Why == "" {
			t.Errorf("tool %q missing Why (why it's useful)", tool.ID)
		}
		if seen[tool.ID] {
			t.Errorf("duplicate tool ID %q", tool.ID)
		}
		seen[tool.ID] = true

		// Every tool must be installable somewhere OR have manual guidance,
		// so the UI never offers a dead end.
		installable := tool.Brew != "" || tool.Apt != "" || tool.Dnf != "" || tool.Pacman != ""
		if !installable && tool.Manual == "" {
			t.Errorf("tool %q has neither a package nor manual guidance", tool.ID)
		}
		switch tool.Category {
		case Critical:
			haveCritical = true
		case Cool:
			haveCool = true
		}
		if tool.ID == "oh-my-posh" {
			haveOhMyPosh = true
		}
	}
	if !haveCritical || !haveCool {
		t.Error("expected both critical and cool tools in the catalog")
	}
	if !haveOhMyPosh {
		t.Error("expected oh-my-posh in the catalog (explicitly requested)")
	}
}

func TestCategory_String(t *testing.T) {
	if Critical.String() != "critical" || Cool.String() != "cool" || Runtime.String() != "runtime" {
		t.Errorf("unexpected category strings: %q %q %q", Critical.String(), Cool.String(), Runtime.String())
	}
}

func TestDevCatalog_IncludesBrewAndRuntimes(t *testing.T) {
	cat := DevCatalog()
	want := map[string]bool{"brew": false, "python3": false, "node": false, "go": false, "java": false, "rustc": false}
	var haveRuntime, haveBootstrap bool
	for _, tool := range cat {
		if _, ok := want[tool.ID]; ok {
			want[tool.ID] = true
		}
		if tool.Category == Runtime {
			haveRuntime = true
		}
		if tool.ID == "brew" && tool.Bootstrap {
			haveBootstrap = true
		}
	}
	for id, found := range want {
		if !found {
			t.Errorf("DevCatalog missing expected entry %q", id)
		}
	}
	if !haveRuntime {
		t.Error("expected Runtime-category tools in DevCatalog")
	}
	if !haveBootstrap {
		t.Error("expected Homebrew marked Bootstrap in DevCatalog")
	}
	// DevCatalog must include the terminal-tool catalog too.
	if len(cat) <= len(Catalog()) {
		t.Error("expected DevCatalog to extend the terminal-tool Catalog")
	}
}

func TestHomebrewInstallPlan(t *testing.T) {
	p := HomebrewInstallPlan()
	if p.Bin != "/bin/bash" {
		t.Errorf("expected bash bin, got %q", p.Bin)
	}
	d := p.Display()
	for _, want := range []string{"curl", "Homebrew/install", "install.sh"} {
		if !strings.Contains(d, want) {
			t.Errorf("Homebrew installer command missing %q: %s", want, d)
		}
	}
}

func TestDetectManager(t *testing.T) {
	tests := []struct {
		name     string
		goos     string
		look     func(string) (string, error)
		wantName string
		wantSudo bool
		wantOK   bool
	}{
		{"macOS brew", "darwin", lookProvides("brew"), "brew", false, true},
		{"macOS no brew", "darwin", lookProvides(), "", false, false},
		{"linux prefers native apt over brew", "linux", lookProvides("brew", "apt-get"), "apt", true, true},
		{"linux prefers native dnf over brew", "linux", lookProvides("brew", "dnf"), "dnf", true, true},
		{"linux apt", "linux", lookProvides("apt-get"), "apt", true, true},
		{"linux dnf", "linux", lookProvides("dnf"), "dnf", true, true},
		{"linux pacman", "linux", lookProvides("pacman"), "pacman", true, true},
		{"linux brew fallback when no native manager", "linux", lookProvides("brew"), "brew", false, true},
		{"linux none", "linux", lookProvides(), "", false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, ok := detectManager(tt.goos, tt.look)
			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOK)
			}
			if !ok {
				return
			}
			if m.Name != tt.wantName {
				t.Errorf("manager = %q, want %q", m.Name, tt.wantName)
			}
			if m.NeedsSudo != tt.wantSudo {
				t.Errorf("sudo = %v, want %v", m.NeedsSudo, tt.wantSudo)
			}
		})
	}
}

func TestBuildInstall_Brew(t *testing.T) {
	mgr, _ := detectManager("darwin", lookProvides("brew"))
	tool := Tool{ID: "jq", Brew: "jq"}
	plan, ok, reason := BuildInstall(tool, mgr)
	if !ok {
		t.Fatalf("expected installable, got reason %q", reason)
	}
	if plan.Sudo {
		t.Error("brew install should not use sudo")
	}
	if !strings.Contains(plan.Display(), "install jq") {
		t.Errorf("unexpected command: %s", plan.Display())
	}
}

func TestBuildInstall_AptUsesSudo(t *testing.T) {
	mgr, _ := detectManager("linux", lookProvides("apt-get"))
	tool := Tool{ID: "jq", Apt: "jq"}
	plan, ok, _ := BuildInstall(tool, mgr)
	if !ok {
		t.Fatal("expected installable via apt")
	}
	if !plan.Sudo || plan.Bin != "sudo" {
		t.Errorf("apt install should use sudo, got bin=%q sudo=%v", plan.Bin, plan.Sudo)
	}
	got := plan.Display()
	for _, want := range []string{"sudo", "apt-get", "install", "-y", "jq"} {
		if !strings.Contains(got, want) {
			t.Errorf("apt command %q missing %q", got, want)
		}
	}
}

func TestBuildInstall_BrewCask(t *testing.T) {
	mgr, _ := detectManager("darwin", lookProvides("brew"))
	tool := Tool{ID: "x", Brew: "somecask", BrewCask: true}
	plan, ok, _ := BuildInstall(tool, mgr)
	if !ok {
		t.Fatal("expected installable")
	}
	if !strings.Contains(plan.Display(), "--cask") {
		t.Errorf("expected --cask in %q", plan.Display())
	}
}

func TestBuildInstall_ManualFallback(t *testing.T) {
	mgr, _ := detectManager("linux", lookProvides("apt-get"))
	// Tool with no apt package → not installable via apt, must surface manual.
	tool := Tool{ID: "oh-my-posh", Brew: "oh-my-posh", Manual: "1. Download: curl -s https://ohmyposh.dev/install.sh -o install.sh\n2. Verify: review the script or check SHA256 at https://ohmyposh.dev/docs/installation/linux\n3. Run: bash install.sh"}
	_, ok, reason := BuildInstall(tool, mgr)
	if ok {
		t.Fatal("expected non-installable via apt (no apt package)")
	}
	if !strings.Contains(reason, "Download") || !strings.Contains(reason, "Verify") || !strings.Contains(reason, "Run") {
		t.Errorf("expected download→verify→run guidance, got %q", reason)
	}
}

func TestManualTools_SecurityGuidance(t *testing.T) {
	// AC-5.1: Script-install tools must present download→verify→run guidance.
	cat := Catalog()
	scriptTools := []string{"starship", "oh-my-posh"}
	for _, id := range scriptTools {
		for _, tool := range cat {
			if tool.ID != id {
				continue
			}
			if tool.Manual == "" {
				t.Errorf("tool %q missing manual guidance", id)
				continue
			}
			// Must NOT contain bare "curl | bash" or "curl | sh" patterns.
			if strings.Contains(tool.Manual, "| bash") || strings.Contains(tool.Manual, "| sh") {
				t.Errorf("tool %q manual contains bare pipe-to-shell: %q", id, tool.Manual)
			}
			// Must contain verification step.
			if !strings.Contains(strings.ToLower(tool.Manual), "verify") &&
				!strings.Contains(strings.ToLower(tool.Manual), "sha") &&
				!strings.Contains(strings.ToLower(tool.Manual), "gpg") {
				t.Errorf("tool %q manual missing verification guidance: %q", id, tool.Manual)
			}
		}
	}
}

func TestPackageManagerInstalls_Unchanged(t *testing.T) {
	// AC-5.2: Package-manager installs must not change behavior.
	mgr, _ := detectManager("darwin", lookProvides("brew"))
	tool := Tool{ID: "starship", Brew: "starship"}
	plan, ok, _ := BuildInstall(tool, mgr)
	if !ok {
		t.Fatal("expected installable via brew")
	}
	// Should be a normal brew install, no verification gating.
	if !strings.Contains(plan.Display(), "install starship") {
		t.Errorf("unexpected brew command: %s", plan.Display())
	}
}

// === ToolsForOS ===

func TestToolsForOS(t *testing.T) {
	mac := ToolsForOS("darwin")
	linux := ToolsForOS("linux")

	has := func(list []Tool, id string) bool {
		for _, x := range list {
			if x.ID == id {
				return true
			}
		}
		return false
	}

	// cmux is macOS-only: present on darwin, absent on linux.
	if !has(mac, "cmux") {
		t.Error("expected cmux in macOS tools")
	}
	if has(linux, "cmux") {
		t.Error("cmux (MacOnly) must not appear in Linux tools")
	}
	// Homebrew bootstrap belongs to both (Linuxbrew).
	if !has(mac, "brew") || !has(linux, "brew") {
		t.Error("expected Homebrew in both OS lists")
	}
	// Cross-platform staples appear on both.
	for _, id := range []string{"git", "rg", "jq", "tmux"} {
		if !has(mac, id) || !has(linux, id) {
			t.Errorf("expected %q on both macOS and Linux", id)
		}
	}
	// Linux entries must never be MacOnly.
	for _, x := range linux {
		if x.MacOnly {
			t.Errorf("Linux list contains MacOnly tool %q", x.ID)
		}
	}
}

// === InstallAllPlan ===

func TestInstallAllPlan_Brew(t *testing.T) {
	mgr, ok := detectManager("darwin", lookProvides("brew"))
	if !ok {
		t.Fatal("expected brew manager")
	}
	// Nothing installed yet → all installable macOS tools go into one command.
	none := func(string) bool { return false }
	plan, skipped, ok := installAllPlan(ToolsForOS("darwin"), mgr, none)
	if !ok {
		t.Fatal("expected an install plan when nothing is installed")
	}
	if plan.Bin != mgr.Bin || len(plan.Args) < 2 || plan.Args[0] != "install" {
		t.Fatalf("unexpected brew plan: %+v", plan)
	}
	// Homebrew itself (bootstrap) and cmux (manual, no brew formula) are skipped.
	joined := strings.Join(skipped, " | ")
	if !strings.Contains(joined, "Homebrew") {
		t.Errorf("expected Homebrew bootstrap skipped, got %q", joined)
	}
	// git's formula should be in the batch.
	if !contains(plan.Args, "git") {
		t.Errorf("expected git in batch args: %v", plan.Args)
	}
}

func TestInstallAllPlan_AptNeedsSudo(t *testing.T) {
	mgr, ok := detectManager("linux", lookProvides("apt-get"))
	if !ok {
		t.Fatal("expected apt manager")
	}
	none := func(string) bool { return false }
	plan, _, ok := installAllPlan(ToolsForOS("linux"), mgr, none)
	if !ok {
		t.Fatal("expected an install plan")
	}
	if !plan.Sudo || plan.Bin != "sudo" || plan.Args[0] != mgr.Bin {
		t.Fatalf("apt plan should be sudo-prefixed: %+v", plan)
	}
}

func TestInstallAllPlan_NothingToDo(t *testing.T) {
	mgr, _ := detectManager("darwin", lookProvides("brew"))
	all := func(string) bool { return true } // everything already installed
	_, _, ok := installAllPlan(ToolsForOS("darwin"), mgr, all)
	if ok {
		t.Error("expected ok=false when every tool is already installed")
	}
}

func contains(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}

// Every tool surfaced in the Settings dev catalog (runtimes + terminal tools)
// must explain both what it is and why it's useful.
func TestDevCatalog_AllHaveDescriptionAndWhy(t *testing.T) {
	for _, tool := range DevCatalog() {
		if tool.Description == "" {
			t.Errorf("DevCatalog tool %q missing Description", tool.ID)
		}
		if tool.Why == "" {
			t.Errorf("DevCatalog tool %q missing Why", tool.ID)
		}
	}
}
