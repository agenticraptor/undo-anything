// Package timeline implements the interactive `ua timeline` TUI: a scrollable
// list of snapshots on the left, details and a diff on the right, and one-key
// restore. It is intentionally self-contained so it can be screenshotted as the
// product's hero image.
package timeline

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/agenticraptor/undo-anything/internal/config"
	"github.com/agenticraptor/undo-anything/internal/humanize"
	"github.com/agenticraptor/undo-anything/internal/ignore"
	"github.com/agenticraptor/undo-anything/internal/restore"
	"github.com/agenticraptor/undo-anything/internal/snapshot"
	"github.com/agenticraptor/undo-anything/internal/store"
)

const listWidth = 36

var (
	accent     = lipgloss.AdaptiveColor{Light: "#7C3AED", Dark: "#A78BFA"}
	green      = lipgloss.AdaptiveColor{Light: "#15803D", Dark: "#4ADE80"}
	yellow     = lipgloss.AdaptiveColor{Light: "#B45309", Dark: "#FBBF24"}
	red        = lipgloss.AdaptiveColor{Light: "#B91C1C", Dark: "#F87171"}
	mutedFg    = lipgloss.AdaptiveColor{Light: "#6B7280", Dark: "#9CA3AF"}
	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(accent)
	muted      = lipgloss.NewStyle().Foreground(mutedFg)
	selStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFFFFF")).Background(accent)
	keyStyle   = lipgloss.NewStyle().Foreground(accent).Bold(true)
	listBox    = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(mutedFg).Padding(0, 1)
	detailBox  = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(mutedFg).Padding(0, 1)
)

type model struct {
	store   *store.Store
	matcher *ignore.Matcher
	cfg     config.Config

	entries []store.IndexEntry // newest first
	cursor  int
	offset  int

	detail        viewport.Model
	width, height int
	status        string
	confirming    bool
	ready         bool
	quitting      bool
}

// Run launches the timeline TUI for the given store.
func Run(s *store.Store, m *ignore.Matcher, cfg config.Config) error {
	entries, err := s.IndexDesc()
	if err != nil {
		return err
	}
	mod := &model{store: s, matcher: m, cfg: cfg, entries: entries}
	_, err = tea.NewProgram(mod, tea.WithAltScreen()).Run()
	return err
}

func (m *model) Init() tea.Cmd { return nil }

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		detailW := msg.Width - listWidth - 6
		if detailW < 20 {
			detailW = 20
		}
		detailH := msg.Height - 6
		if detailH < 3 {
			detailH = 3
		}
		if !m.ready {
			m.detail = viewport.New(detailW, detailH)
			m.ready = true
		} else {
			m.detail.Width = detailW
			m.detail.Height = detailH
		}
		m.refreshDetail()

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit
		case "esc", "n":
			if m.confirming {
				m.confirming = false
				m.status = ""
			}
		case "up", "k":
			if !m.confirming {
				m.moveCursor(-1)
			}
		case "down", "j":
			if !m.confirming {
				m.moveCursor(1)
			}
		case "g", "home":
			if !m.confirming {
				m.cursor = 0
				m.refreshDetail()
			}
		case "G", "end":
			if !m.confirming {
				m.cursor = len(m.entries) - 1
				m.refreshDetail()
			}
		case "pgdown", "ctrl+d":
			m.detail.HalfViewDown()
		case "pgup", "ctrl+u":
			m.detail.HalfViewUp()
		case "r":
			if !m.confirming && len(m.entries) > 0 {
				m.confirming = true
				m.status = fmt.Sprintf("Restore the whole folder to %s? (a safety snapshot is taken first)  [y/n]", m.entries[m.cursor].ID)
			}
		case "y":
			if m.confirming {
				m.doRestore()
				m.confirming = false
			}
		}
	}
	return m, nil
}

func (m *model) moveCursor(delta int) {
	if len(m.entries) == 0 {
		return
	}
	m.cursor += delta
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor >= len(m.entries) {
		m.cursor = len(m.entries) - 1
	}
	m.refreshDetail()
}

func (m *model) doRestore() {
	if len(m.entries) == 0 {
		return
	}
	id := m.entries[m.cursor].ID
	snap, err := m.store.ReadSnapshot(id)
	if err != nil {
		m.status = "Error: " + err.Error()
		return
	}
	res, err := restore.Snapshot(m.store, snap, restore.Options{
		Safety:      true,
		Matcher:     m.matcher,
		MaxFileSize: m.cfg.MaxFileSizeBytes(),
	})
	if err != nil {
		m.status = "Restore failed: " + err.Error()
		return
	}
	m.status = fmt.Sprintf("✓ Restored to %s · %d files · safety snapshot %s (press r on it to undo)",
		id, len(res.Restored), res.SafetyID)
	// Reload entries so the new safety snapshot appears.
	if entries, err := m.store.IndexDesc(); err == nil {
		m.entries = entries
	}
	m.refreshDetail()
}

func (m *model) refreshDetail() {
	if !m.ready || len(m.entries) == 0 {
		return
	}
	e := m.entries[m.cursor]
	snap, err := m.store.ReadSnapshot(e.ID)
	if err != nil {
		m.detail.SetContent("Could not read snapshot: " + err.Error())
		return
	}
	var b strings.Builder
	fmt.Fprintln(&b, titleStyle.Render("Snapshot "+snap.ID))
	fmt.Fprintln(&b, muted.Render("taken  ")+snap.Time.Local().Format("2006-01-02 15:04:05")+muted.Render(" ("+humanize.RelTime(snap.Time)+")"))
	fmt.Fprintln(&b, muted.Render("kind   ")+snap.Trigger)
	if snap.Label != "" {
		fmt.Fprintln(&b, muted.Render("label  ")+snap.Label)
	}
	fmt.Fprintln(&b, muted.Render("files  ")+fmt.Sprintf("%d (%s)", snap.Stats.Files, humanize.Bytes(snap.Stats.TotalSize)))
	fmt.Fprintln(&b)

	var parent *store.Snapshot
	if snap.Parent != "" {
		parent, _ = m.store.ReadSnapshot(snap.Parent)
	}
	changes := snapshot.Diff(parent, snap)
	if len(changes) == 0 {
		fmt.Fprintln(&b, muted.Render("(no file-level changes vs parent)"))
	} else {
		fmt.Fprintln(&b, lipgloss.NewStyle().Bold(true).Render("Changes vs parent"))
		for _, c := range changes {
			switch c.Kind {
			case snapshot.Added:
				fmt.Fprintf(&b, "%s %s\n", lipgloss.NewStyle().Foreground(green).Render("+"), c.Path)
			case snapshot.Removed:
				fmt.Fprintf(&b, "%s %s\n", lipgloss.NewStyle().Foreground(red).Render("-"), c.Path)
			case snapshot.Modified:
				fmt.Fprintf(&b, "%s %s\n", lipgloss.NewStyle().Foreground(yellow).Render("~"), c.Path)
			}
		}
	}
	m.detail.SetContent(b.String())
	m.detail.GotoTop()
}

func (m *model) View() string {
	if m.quitting {
		return ""
	}
	if !m.ready {
		return "Loading timeline…"
	}
	if len(m.entries) == 0 {
		return titleStyle.Render("undo-anything timeline") + "\n\n" +
			muted.Render("No snapshots yet. Run `ua snapshot` or start the watcher, then come back.") +
			"\n\n" + muted.Render("Press q to quit.")
	}

	header := titleStyle.Render("undo-anything timeline") + "  " +
		muted.Render(fmt.Sprintf("%d snapshots · %s", len(m.entries), relRoot(m.store.Root)))

	// Left list with a scrolling window.
	listHeight := m.height - 6
	if listHeight < 3 {
		listHeight = 3
	}
	m.ensureVisible(listHeight)
	var rows []string
	for i := m.offset; i < len(m.entries) && i < m.offset+listHeight; i++ {
		rows = append(rows, m.renderRow(i))
	}
	left := listBox.Width(listWidth).Height(listHeight).Render(strings.Join(rows, "\n"))
	right := detailBox.Width(m.detail.Width + 2).Height(m.detail.Height).Render(m.detail.View())
	body := lipgloss.JoinHorizontal(lipgloss.Top, left, right)

	footer := m.status
	if footer == "" {
		footer = muted.Render("↑/↓ move · ") + keyStyle.Render("r") + muted.Render(" restore · ") +
			keyStyle.Render("pgup/pgdn") + muted.Render(" scroll · ") + keyStyle.Render("q") + muted.Render(" quit")
	}
	return header + "\n" + body + "\n" + footer
}

func (m *model) renderRow(i int) string {
	e := m.entries[i]
	dot := dotFor(e.Trigger)
	label := fmt.Sprintf("%s %s  %s", dot, e.ID, humanize.RelTime(e.Time))
	sub := muted.Render(fmt.Sprintf("   %d files", e.Files))
	if e.Label != "" {
		sub = muted.Render("   " + truncate(e.Label, listWidth-5))
	}
	line := label + "\n" + sub
	if i == m.cursor {
		return selStyle.Width(listWidth-4).Render(label) + "\n" + sub
	}
	return line
}

func (m *model) ensureVisible(listHeight int) {
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	if m.cursor >= m.offset+listHeight {
		m.offset = m.cursor - listHeight + 1
	}
	if m.offset < 0 {
		m.offset = 0
	}
}

func dotFor(trigger string) string {
	c := mutedFg
	switch trigger {
	case "watch":
		c = green
	case "manual":
		c = accent
	case "pre-restore":
		c = yellow
	}
	return lipgloss.NewStyle().Foreground(c).Render("●")
}

func truncate(s string, n int) string {
	if n < 1 {
		n = 1
	}
	if len(s) <= n {
		return s
	}
	if n <= 1 {
		return "…"
	}
	return s[:n-1] + "…"
}

func relRoot(p string) string {
	parts := strings.Split(p, "/")
	if len(parts) <= 2 {
		return p
	}
	return ".../" + strings.Join(parts[len(parts)-2:], "/")
}
