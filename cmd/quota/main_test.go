package main

import "testing"

func TestParseConfigMultipleAccounts(t *testing.T) {
	config, err := parseConfig([]string{
		"once",
		"--claude-dir", "~/.claude-work",
		"--codex-dir=~/.codex-work",
		"--claude-dir", "/tmp/claude-other",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !config.once {
		t.Fatal("once was not enabled")
	}
	if len(config.codexDirs) != 1 || len(config.claudeDirs) != 2 {
		t.Fatalf("unexpected directories: codex=%v claude=%v", config.codexDirs, config.claudeDirs)
	}
	if got := len(config.sources()); got != 5 {
		t.Fatalf("got %d sources, want 5", got)
	}
}

func TestParseConfigAcceptsFlagsBeforeOnce(t *testing.T) {
	config, err := parseConfig([]string{"--codex-dir", "/tmp/work", "once"})
	if err != nil {
		t.Fatal(err)
	}
	if !config.once || len(config.codexDirs) != 1 {
		t.Fatalf("unexpected config: %+v", config)
	}
}
