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
	if Critical.String() != "critical" || Cool.String() != "cool" {
		t.Errorf("unexpected category strings: %q %q", Critical.String(), Cool.String())
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
		{"linux prefers linuxbrew", "linux", lookProvides("brew", "apt-get"), "brew", false, true},
		{"linux apt", "linux", lookProvides("apt-get"), "apt", true, true},
		{"linux dnf", "linux", lookProvides("dnf"), "dnf", true, true},
		{"linux pacman", "linux", lookProvides("pacman"), "pacman", true, true},
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
