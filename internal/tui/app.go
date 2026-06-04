package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/chadnorvell/quota/internal/provider"
)

type Model struct {
	sources     []provider.Source
	snapshots   []provider.Snapshot
	width       int
	height      int
	loading     bool
	displayMode displayMode
}

type tickMsg time.Time
type snapshotsMsg []provider.Snapshot

type displayMode int

const (
	displayUsed displayMode = iota
	displayRemaining
)

func New(sources []provider.Source) Model {
	return Model{sources: sources, loading: true, displayMode: displayUsed}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(fetch(m.sources), tick())
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch typed := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = typed.Width
		m.height = typed.Height
	case tea.KeyMsg:
		switch typed.String() {
		case "q", "esc", "ctrl+c":
			return m, tea.Quit
		case "r":
			m.loading = true
			return m, fetch(m.sources)
		case "u", "tab":
			m.displayMode = m.displayMode.toggle()
		}
	case snapshotsMsg:
		m.snapshots = typed
		m.loading = false
	case tickMsg:
		m.loading = true
		return m, tea.Batch(fetch(m.sources), tick())
	}
	return m, nil
}

func (m Model) View() string {
	width := m.width
	if width < 80 {
		width = 80
	}

	title := styles.headerTitle.Render("quota")
	subtitle := styles.headerSubtitle.Render("AI service subscription dashboard")
	status := styles.subtle.Render(fmt.Sprintf("mode %s  u/tab toggle  r refresh  q quit", m.displayMode))
	if m.loading {
		status = styles.warn.Render("refreshing...") + styles.subtle.Render(fmt.Sprintf("  mode %s  u/tab toggle  q quit", m.displayMode))
	}
	header := lipgloss.JoinHorizontal(lipgloss.Center, title, styles.headerGap.Render(" "), subtitle)

	var panels []string
	if len(m.snapshots) == 0 {
		panels = []string{box("Status", styles.subtle.Render("Fetching providers..."), width-2)}
	} else {
		panelWidth := width - 2
		for _, snapshot := range m.snapshots {
			panels = append(panels, renderProvider(snapshot, panelWidth, m.displayMode))
		}
	}

	body := strings.Join(panels, "\n")

	return lipgloss.JoinVertical(
		lipgloss.Left,
		styles.header.Width(width-2).Render(header),
		body,
		styles.footer.Width(width-2).Render(status),
	)
}

func renderProvider(snapshot provider.Snapshot, width int, mode displayMode) string {
	lines := []string{}
	meta := []string{snapshot.Status}
	if snapshot.Source != "" {
		meta = append(meta, snapshot.Source)
	}
	if snapshot.Plan != "" {
		meta = append(meta, "plan "+provider.PlanName(snapshot.Plan))
	}
	if snapshot.Account != "" {
		meta = append(meta, snapshot.Account)
	}
	lines = append(lines, styles.subtle.Render(strings.Join(meta, " | ")))

	for _, lane := range snapshot.Lanes {
		lines = append(lines, renderLane(snapshot.Name, lane, width-6, mode))
	}
	if len(snapshot.Lanes) == 0 && snapshot.Error == "" {
		lines = append(lines, styles.subtle.Render("No quota lanes available yet."))
	}
	for _, note := range snapshot.Notes {
		lines = append(lines, styles.subtle.Render(note))
	}
	if snapshot.Error != "" {
		lines = append(lines, styles.error.Render(snapshot.Error))
	}
	if !snapshot.UpdatedAt.IsZero() {
		lines = append(lines, styles.subtle.Render("updated "+snapshot.UpdatedAt.Format("15:04:05")))
	}
	return box(snapshot.Name, strings.Join(lines, "\n"), width)
}

func renderLane(providerName string, lane provider.Lane, width int, mode displayMode) string {
	labelWidth := laneLabelWidth(width)
	detailWidth := laneDetailWidth(width)
	barWidth := width - labelWidth - detailWidth - 4
	if barWidth < 10 {
		barWidth = 10
	}
	percent := lane.Percent
	if mode == displayRemaining && lane.Limit > 0 {
		percent = 100 - lane.Percent
	}
	label := styles.label.Width(labelWidth).Render(truncate(lane.Label, labelWidth))
	bar := renderBar(providerName, percent, barWidth, mode)
	detail := laneDetail(lane, mode)
	return lipgloss.JoinHorizontal(
		lipgloss.Center,
		label,
		" ",
		bar,
		" ",
		styles.value.Width(detailWidth).Render(truncate(detail, detailWidth)),
	)
}

func laneLabelWidth(width int) int {
	switch {
	case width >= 100:
		return 28
	case width >= 80:
		return 24
	case width >= 64:
		return 20
	default:
		return 16
	}
}

func laneDetailWidth(width int) int {
	switch {
	case width >= 120:
		return 34
	case width >= 96:
		return 30
	case width >= 76:
		return 26
	default:
		return 22
	}
}

func laneDetail(lane provider.Lane, mode displayMode) string {
	if mode == displayRemaining && lane.Limit > 0 {
		remainingPercent := 100 - lane.Percent
		if remainingPercent < 0 {
			remainingPercent = 0
		}
		if lane.Unit == "%" || lane.Limit == 100 {
			detail := fmt.Sprintf("%-5s left", percentText(remainingPercent))
			if lane.Reset != "" {
				detail += " reset " + lane.Reset
			}
			return detail
		}
		remaining := lane.Limit - lane.Used
		if remaining < 0 {
			remaining = 0
		}
		return fmt.Sprintf("%.0f left", remaining)
	}
	if lane.Detail != "" {
		return normalizeDetailAmount(lane.Detail)
	}
	if lane.Limit > 0 {
		return fmt.Sprintf("%.0f/%.0f", lane.Used, lane.Limit)
	}
	return ""
}

func normalizeDetailAmount(detail string) string {
	parts := strings.SplitN(detail, " ", 2)
	if len(parts) != 2 {
		return detail
	}
	if !isAmountToken(parts[0]) {
		return detail
	}
	return fmt.Sprintf("%-5s %s", parts[0], parts[1])
}

func isAmountToken(value string) bool {
	if value == "" || len(value) > 5 {
		return false
	}
	for _, char := range value {
		if char >= '0' && char <= '9' {
			continue
		}
		if char == '.' || char == '%' || char == '$' {
			continue
		}
		return false
	}
	return true
}

func percentText(value float64) string {
	if value >= 99.95 {
		return "100%"
	}
	return fmt.Sprintf("%.1f%%", value)
}

func renderBar(providerName string, percent float64, width int, mode displayMode) string {
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}
	filled := int(percent / 100 * float64(width))
	if filled > width {
		filled = width
	}
	fillStyle := providerBarStyle(providerName)
	if mode == displayRemaining {
		if percent <= 15 {
			fillStyle = styles.error
		} else if percent <= 35 {
			fillStyle = providerBarStyle(providerName)
		}
	} else {
		if percent >= 85 {
			fillStyle = styles.error
		} else if percent >= 65 {
			fillStyle = providerBarStyle(providerName)
		}
	}
	return fillStyle.Render(strings.Repeat("━", filled)) + styles.track.Render(strings.Repeat("━", width-filled))
}

func providerBarStyle(providerName string) lipgloss.Style {
	switch strings.ToLower(providerName) {
	case "claude":
		return styles.claude
	default:
		return styles.ok
	}
}

func box(title, body string, width int) string {
	return styles.box.Width(width).Render(styles.boxTitle.Render(" "+title+" ") + "\n" + body)
}

func fetch(sources []provider.Source) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
		defer cancel()
		snapshots := make([]provider.Snapshot, 0, len(sources))
		for _, source := range sources {
			snapshots = append(snapshots, source.Fetch(ctx))
		}
		return snapshotsMsg(snapshots)
	}
}

func tick() tea.Cmd {
	return tea.Tick(5*time.Minute, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func (m displayMode) toggle() displayMode {
	if m == displayRemaining {
		return displayUsed
	}
	return displayRemaining
}

func (m displayMode) String() string {
	if m == displayRemaining {
		return "remaining"
	}
	return "used"
}

func truncate(value string, limit int) string {
	if len(value) <= limit {
		return value
	}
	if limit <= 1 {
		return value[:limit]
	}
	return value[:limit-1] + "…"
}
