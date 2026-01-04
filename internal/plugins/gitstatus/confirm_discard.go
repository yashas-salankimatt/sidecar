package gitstatus

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/marcus/sidecar/internal/styles"
	"github.com/marcus/sidecar/internal/ui"
)

// renderConfirmDiscard renders the confirm discard modal overlay.
func (p *Plugin) renderConfirmDiscard() string {
	// Render the background (status view dimmed)
	background := p.renderThreePaneView()

	if p.discardFile == nil {
		return background
	}

	entry := p.discardFile

	// Build modal content
	var sb strings.Builder

	// Warning icon and title
	title := styles.StatusDeleted.Render(" Discard Changes ")
	sb.WriteString(title)
	sb.WriteString("\n\n")

	// File info
	statusLabel := "modified"
	if entry.Staged {
		statusLabel = "staged"
	} else if entry.Status == StatusUntracked {
		statusLabel = "untracked"
	}

	sb.WriteString(fmt.Sprintf("  Discard %s changes to:\n", statusLabel))
	sb.WriteString(fmt.Sprintf("  %s\n\n", styles.Subtitle.Render(entry.Path)))

	// Warning message
	if entry.Status == StatusUntracked {
		sb.WriteString(styles.StatusDeleted.Render("  This will permanently delete the file!"))
	} else {
		sb.WriteString(styles.Muted.Render("  This will revert to the last committed state."))
	}
	sb.WriteString("\n\n")

	// Options
	yKey := styles.KeyHint.Render(" y ")
	nKey := styles.KeyHint.Render(" n ")
	sb.WriteString(fmt.Sprintf("  %s Confirm    %s Cancel", yKey, nKey))

	// Create modal box
	modalWidth := 50
	if len(entry.Path) > 35 {
		modalWidth = len(entry.Path) + 15
	}
	if modalWidth > p.width-10 {
		modalWidth = p.width - 10
	}

	modalContent := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.StatusDeleted.GetForeground()).
		Padding(1, 2).
		Width(modalWidth).
		Render(sb.String())

	// Overlay modal on dimmed background
	return ui.OverlayModal(background, modalContent, p.width, p.height)
}
