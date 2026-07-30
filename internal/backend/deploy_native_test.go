package backend

import (
	"testing"
)

func TestDeployNative_ArgsConstruction(t *testing.T) {
	tests := []struct {
		name       string
		agentName  string
		trustTools bool
		wantArgs   []string
		wantErr    bool
	}{
		{
			name:       "basic agent",
			agentName:  "numbuh-4",
			trustTools: false,
			wantArgs:   []string{"kiro-cli", "chat", "--agent", "numbuh-4"},
		},
		{
			name:       "with trust tools",
			agentName:  "numbuh-3",
			trustTools: true,
			wantArgs:   []string{"kiro-cli", "chat", "--agent", "numbuh-3", "--trust-all-tools", "--no-interactive"},
		},
		{
			name:      "empty name",
			agentName: "",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k := &Kiro{TrustTools: tt.trustTools}
			args, err := k.DeployNative(tt.agentName)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(args) != len(tt.wantArgs) {
				t.Fatalf("expected %d args, got %d: %v", len(tt.wantArgs), len(args), args)
			}

			for i, want := range tt.wantArgs {
				if args[i] != want {
					t.Errorf("args[%d]: expected %q, got %q", i, want, args[i])
				}
			}
		})
	}
}

func TestDeployNative_NoSafeEnvComment(t *testing.T) {
	// This test documents that DeployNative intentionally does NOT include SafeEnv.
	// The method returns args for exec — the caller decides env based on config.
	k := &Kiro{TrustTools: true}
	args, err := k.DeployNative("numbuh-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify it returns kiro-cli as the first arg (binary path)
	if args[0] != "kiro-cli" {
		t.Errorf("expected first arg 'kiro-cli', got %q", args[0])
	}
}
