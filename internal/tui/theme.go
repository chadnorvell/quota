package tui

import "github.com/charmbracelet/lipgloss"

var styles = struct {
	header         lipgloss.Style
	headerTitle    lipgloss.Style
	headerGap      lipgloss.Style
	headerSubtitle lipgloss.Style
	footer         lipgloss.Style
	title          lipgloss.Style
	box            lipgloss.Style
	boxTitle       lipgloss.Style
	label          lipgloss.Style
	value          lipgloss.Style
	subtle         lipgloss.Style
	track          lipgloss.Style
	ok             lipgloss.Style
	claude         lipgloss.Style
	warn           lipgloss.Style
	error          lipgloss.Style
}{
	header:         lipgloss.NewStyle().Foreground(lipgloss.Color("#deddda")).Background(lipgloss.Color("#1d1d1d")).Padding(0, 1),
	headerTitle:    lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#62a0ea")).Background(lipgloss.Color("#1d1d1d")),
	headerGap:      lipgloss.NewStyle().Background(lipgloss.Color("#1d1d1d")),
	headerSubtitle: lipgloss.NewStyle().Foreground(lipgloss.Color("#77767b")).Background(lipgloss.Color("#1d1d1d")),
	footer:         lipgloss.NewStyle().Foreground(lipgloss.Color("#77767b")).Background(lipgloss.Color("#1d1d1d")).Padding(0, 1),
	title:          lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#62a0ea")),
	box:            lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("#77767b")).Padding(0, 1).Margin(1, 0, 0, 0),
	boxTitle:       lipgloss.NewStyle().Foreground(lipgloss.Color("#deddda")).Bold(true),
	label:          lipgloss.NewStyle().Foreground(lipgloss.Color("#deddda")),
	value:          lipgloss.NewStyle().Foreground(lipgloss.Color("#deddda")),
	subtle:         lipgloss.NewStyle().Foreground(lipgloss.Color("#77767b")),
	track:          lipgloss.NewStyle().Foreground(lipgloss.Color("#303030")),
	ok:             lipgloss.NewStyle().Foreground(lipgloss.Color("#62a0ea")),
	claude:         lipgloss.NewStyle().Foreground(lipgloss.Color("#d97706")),
	warn:           lipgloss.NewStyle().Foreground(lipgloss.Color("#f6d32d")),
	error:          lipgloss.NewStyle().Foreground(lipgloss.Color("#e01b24")),
}
