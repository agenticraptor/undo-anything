package cli

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

// A compact, cohesive palette shared across every command so the CLI looks
// like one tool. Colors are adaptive where it matters for light terminals.
var (
	accent  = lipgloss.AdaptiveColor{Light: "#7C3AED", Dark: "#A78BFA"} // violet
	green   = lipgloss.AdaptiveColor{Light: "#15803D", Dark: "#4ADE80"}
	yellow  = lipgloss.AdaptiveColor{Light: "#B45309", Dark: "#FBBF24"}
	red     = lipgloss.AdaptiveColor{Light: "#B91C1C", Dark: "#F87171"}
	blue    = lipgloss.AdaptiveColor{Light: "#1D4ED8", Dark: "#60A5FA"}
	mutedFg = lipgloss.AdaptiveColor{Light: "#6B7280", Dark: "#9CA3AF"}

	titleStyle   = lipgloss.NewStyle().Bold(true).Foreground(accent)
	headingStyle = lipgloss.NewStyle().Bold(true)
	mutedStyle   = lipgloss.NewStyle().Foreground(mutedFg)
	accentStyle  = lipgloss.NewStyle().Foreground(accent)
	successStyle = lipgloss.NewStyle().Foreground(green)
	warnStyle    = lipgloss.NewStyle().Foreground(yellow)
	dangerStyle  = lipgloss.NewStyle().Foreground(red)
	idStyle      = lipgloss.NewStyle().Foreground(blue).Bold(true)
)

// triggerStyle renders a snapshot trigger as a small colored badge.
func triggerBadge(trigger string) string {
	label := trigger
	var c lipgloss.TerminalColor = mutedFg
	switch trigger {
	case "watch":
		c, label = green, "watch"
	case "manual":
		c, label = accent, "manual"
	case "pre-restore":
		c, label = yellow, "safety"
	case "init":
		c, label = blue, "init"
	}
	return lipgloss.NewStyle().Foreground(c).Render("●") + " " + mutedStyle.Render(label)
}

// icon prefixes for status lines.
const (
	iconOK    = "✓"
	iconWarn  = "⚠"
	iconInfo  = "•"
	iconArrow = "→"
)

func okLine(format string, a ...any) string {
	return successStyle.Render(iconOK) + " " + fmt.Sprintf(format, a...)
}

func warnLine(format string, a ...any) string {
	return warnStyle.Render(iconWarn) + " " + fmt.Sprintf(format, a...)
}

func infoLine(format string, a ...any) string {
	return mutedStyle.Render(iconInfo) + " " + fmt.Sprintf(format, a...)
}
