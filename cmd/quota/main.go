package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/chadnorvell/quota/internal/provider"
	"github.com/chadnorvell/quota/internal/tui"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "once" {
		runOnce()
		return
	}

	sources := []provider.Source{
		provider.NewCodex(),
		provider.NewClaude(),
	}
	model := tui.New(sources)
	if _, err := tea.NewProgram(model, tea.WithAltScreen()).Run(); err != nil {
		log.Fatal(err)
	}
}

func runOnce() {
	ctx := context.Background()
	for _, source := range []provider.Source{provider.NewCodex(), provider.NewClaude()} {
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
