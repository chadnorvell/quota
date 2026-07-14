package provider

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAdditionalAccountNamesAndPaths(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}

	codex := NewCodexFromDir("~/.codex-work")
	if got, want := codex.credentialsDir, filepath.Join(home, ".codex-work"); got != want {
		t.Fatalf("credentials dir = %q, want %q", got, want)
	}
	if got, want := codex.Name(), "Codex (.codex-work)"; got != want {
		t.Fatalf("name = %q, want %q", got, want)
	}

	claude := NewClaudeFromDir("/accounts/claude-team")
	if got, want := claude.Name(), "Claude (claude-team)"; got != want {
		t.Fatalf("name = %q, want %q", got, want)
	}
}

func TestDefaultAccountNames(t *testing.T) {
	if got := NewCodex().Name(); got != "Codex" {
		t.Fatalf("default Codex name = %q", got)
	}
	if got := NewClaude().Name(); got != "Claude" {
		t.Fatalf("default Claude name = %q", got)
	}
}
