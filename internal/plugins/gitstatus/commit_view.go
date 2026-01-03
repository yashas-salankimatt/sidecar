package gitstatus

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/marcus/sidecar/internal/styles"
)

// commitModalWidth returns the width for the commit modal content.
func (p *Plugin) commitModalWidth() int {
	w := p.width - 8 // 4-char margin each side
	if w > 80 {
		w = 80
	}
	if w < 40 {
		w = 40
	}
	return w
}

// renderCommit renders the commit message modal box.
func (p *Plugin) renderCommit() string {
	modalWidth := p.commitModalWidth()
	contentWidth := modalWidth - 6 // Account for border (2) + padding (4)

	var sb strings.Builder

	// Calculate stats
	additions, deletions := p.tree.StagedStats()
	fileCount := len(p.tree.Staged)

	// Header with stats
	statsStr := fmt.Sprintf("[%d: +%d -%d]", fileCount, additions, deletions)
	title := styles.Title.Render(" Commit ")
	statsRendered := styles.Muted.Render(statsStr)
	padding := contentWidth - lipgloss.Width(title) - lipgloss.Width(statsStr)
	if padding < 1 {
		padding = 1
	}
	sb.WriteString(title + strings.Repeat(" ", padding) + statsRendered)
	sb.WriteString("\n")
	sb.WriteString(styles.Muted.Render(strings.Repeat("─", contentWidth)))
	sb.WriteString("\n")

	// Staged files section - show more files based on available height
	sb.WriteString(styles.StatusStaged.Render(fmt.Sprintf("Staged (%d)", fileCount)))
	sb.WriteString("\n")

	// Dynamic maxFiles: allow up to 8, fewer on small terminals
	maxFiles := 8
	if p.height < 30 {
		maxFiles = 6
	}
	if p.height < 24 {
		maxFiles = 4
	}
	for i, entry := range p.tree.Staged {
		if i >= maxFiles {
			remaining := len(p.tree.Staged) - maxFiles
			sb.WriteString(styles.Muted.Render(fmt.Sprintf("  ... +%d more", remaining)))
			sb.WriteString("\n")
			break
		}

		// Status indicator
		status := styles.StatusStaged.Render(string(entry.Status))

		// Path - truncate for modal width
		path := entry.Path
		maxPathWidth := contentWidth - 18
		if maxPathWidth < 10 {
			maxPathWidth = 10
		}
		if len(path) > maxPathWidth {
			path = "..." + path[len(path)-maxPathWidth+3:]
		}

		// Diff stats
		stats := ""
		if entry.DiffStats.Additions > 0 || entry.DiffStats.Deletions > 0 {
			addStr := styles.DiffAdd.Render(fmt.Sprintf("+%d", entry.DiffStats.Additions))
			delStr := styles.DiffRemove.Render(fmt.Sprintf("-%d", entry.DiffStats.Deletions))
			stats = fmt.Sprintf(" %s %s", addStr, delStr)
		}

		sb.WriteString(fmt.Sprintf("  %s %s%s\n", status, path, stats))
	}

	sb.WriteString("\n")

	// Textarea
	sb.WriteString(p.commitMessage.View())
	sb.WriteString("\n")

	// Commit button + hints
	buttonStyle := styles.Button
	if p.commitButtonFocus {
		buttonStyle = styles.ButtonFocused
	}
	sb.WriteString(buttonStyle.Render(" Commit "))
	sb.WriteString("  ")
	sb.WriteString(styles.Muted.Render("Tab/Enter"))

	// Error message if any
	if p.commitError != "" {
		sb.WriteString("\n")
		sb.WriteString(styles.StatusDeleted.Render("✗ " + p.commitError))
	}

	// Progress indicator
	if p.commitInProgress {
		sb.WriteString("\n")
		sb.WriteString(styles.Muted.Render("Committing..."))
	}

	sb.WriteString("\n")
	sb.WriteString(styles.Muted.Render(strings.Repeat("─", contentWidth)))
	sb.WriteString("\n")

	// Footer keybindings
	escKey := styles.KeyHint.Render(" Esc ")
	commitKey := styles.KeyHint.Render(" ^S ")
	sb.WriteString(escKey + " Cancel  " + commitKey + " Commit")

	// Wrap in modal box
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.Primary).
		Padding(1, 2).
		Width(modalWidth).
		Render(sb.String())
}

// renderCommitModal renders the commit modal overlaid on the status view.
func (p *Plugin) renderCommitModal() string {
	// Render background (three-pane view)
	background := p.renderThreePaneView()

	// Get modal content
	modalContent := p.renderCommit()

	// Center modal over background
	centered := lipgloss.Place(
		p.width, p.height,
		lipgloss.Center, lipgloss.Center,
		modalContent,
	)

	// Overlay on background
	return overlayMenu(background, centered, p.width, p.height)
}
