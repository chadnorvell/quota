package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/chadnorvell/quota/internal/provider"
	"github.com/chadnorvell/quota/internal/tui"
)

func main() {
	config, err := parseConfig(os.Args[1:])
	if err != nil {
		if err == flag.ErrHelp {
			return
		}
		log.Fatal(err)
	}
	sources := config.sources()
	if config.once {
		runOnce(sources)
		return
	}

	model := tui.New(sources)
	if _, err := tea.NewProgram(model, tea.WithAltScreen()).Run(); err != nil {
		log.Fatal(err)
	}
}

type directories []string

func (d *directories) String() string { return strings.Join(*d, ",") }

func (d *directories) Set(value string) error {
	*d = append(*d, value)
	return nil
}

type config struct {
	once       bool
	codexDirs  directories
	claudeDirs directories
}

func parseConfig(args []string) (config, error) {
	var config config
	filtered := make([]string, 0, len(args))
	for _, arg := range args {
		if arg == "once" && !config.once {
			config.once = true
			continue
		}
		filtered = append(filtered, arg)
	}
	flags := flag.NewFlagSet("quota", flag.ContinueOnError)
	flags.Var(&config.codexDirs, "codex-dir", "add a Codex credentials directory (repeatable)")
	flags.Var(&config.claudeDirs, "claude-dir", "add a Claude credentials directory (repeatable)")
	if err := flags.Parse(filtered); err != nil {
		return config, err
	}
	if flags.NArg() != 0 {
		return config, fmt.Errorf("unexpected arguments: %s", strings.Join(flags.Args(), " "))
	}
	return config, nil
}

func (c config) sources() []provider.Source {
	sources := []provider.Source{provider.NewCodex(), provider.NewClaude()}
	for _, dir := range c.codexDirs {
		sources = append(sources, provider.NewCodexFromDir(dir))
	}
	for _, dir := range c.claudeDirs {
		sources = append(sources, provider.NewClaudeFromDir(dir))
	}
	return sources
}

func runOnce(sources []provider.Source) {
	ctx := context.Background()
	for _, source := range sources {
		snapshot := source.Fetch(ctx)
		statusParts := []string{snapshot.Status}
		if snapshot.Source != "" {
			statusParts = append(statusParts, snapshot.Source)
		}
		if snapshot.Plan != "" {
			statusParts = append(statusParts, "plan "+provider.PlanName(snapshot.Plan))
		}
		if snapshot.Account != "" {
			statusParts = append(statusParts, snapshot.Account)
		}
		fmt.Printf("%s: %s\n", snapshot.Name, strings.Join(statusParts, " | "))
		for _, lane := range snapshot.Lanes {
			fmt.Printf("  %-18s %6.1f%% %s\n", lane.Label, lane.Percent, lane.Detail)
		}
		if snapshot.Error != "" {
			fmt.Printf("  error: %s\n", snapshot.Error)
		}
	}
}
