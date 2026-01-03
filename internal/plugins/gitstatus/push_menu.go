package gitstatus

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/marcus/sidecar/internal/styles"
)

// dimStyle is used to dim background content behind modals.
var dimStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

// renderPushMenu renders the push options popup menu.
func (p *Plugin) renderPushMenu() string {
	// Render the background (current view dimmed)
	var background string
	switch p.pushMenuReturnMode {
	case ViewModeHistory:
		background = p.renderHistory()
	case ViewModeCommitDetail:
		background = p.renderCommitDetail()
	default:
		background = p.renderThreePaneView()
	}

	// Build menu content
	var sb strings.Builder

	// Menu options
	options := []struct{ key, label string }{
		{"p", "Push to origin"},
		{"f", "Force push (--force-with-lease)"},
		{"u", "Push & set upstream (-u)"},
	}

	for i, opt := range options {
		key := styles.KeyHint.Render(" " + opt.key + " ")
		sb.WriteString(fmt.Sprintf("  %s  %s", key, opt.label))
		if i < len(options)-1 {
			sb.WriteString("\n\n") // Spacing between options
		} else {
			sb.WriteString("\n")
		}
	}

	sb.WriteString("\n")
	sb.WriteString(styles.Muted.Render("  Esc to cancel"))

	// Create menu box - wide enough for longest option
	menuWidth := 44
	title := styles.Title.Render(" Push ")

	menuContent := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.Primary).
		Padding(1, 2).
		Width(menuWidth).
		Render(title + "\n\n" + sb.String())

	// Center menu over background
	menu := lipgloss.Place(
		p.width, p.height,
		lipgloss.Center, lipgloss.Center,
		menuContent,
	)

	// Overlay menu on dimmed background
	return overlayMenu(background, menu, p.width, p.height)
}

// dimBackground dims the background content by stripping colors and applying a muted style.
func dimBackground(background string, width, height int) string {
	lines := strings.Split(background, "\n")
	for i, line := range lines {
		// Strip ANSI codes and re-render with dim style
		stripped := ansi.Strip(line)
		lines[i] = dimStyle.Render(stripped)
	}
	return strings.Join(lines, "\n")
}

// overlayMenu overlays the menu on top of the dimmed background.
func overlayMenu(background, menu string, width, height int) string {
	// Dim the background
	background = dimBackground(background, width, height)

	bgLines := strings.Split(background, "\n")
	menuLines := strings.Split(menu, "\n")

	// Ensure we have enough lines
	for len(bgLines) < height {
		bgLines = append(bgLines, "")
	}

	// The menu is already positioned via lipgloss.Place, so just return it
	// The menu's transparent areas will show the background
	result := make([]string, height)
	for i := 0; i < height; i++ {
		if i < len(menuLines) {
			result[i] = menuLines[i]
		} else if i < len(bgLines) {
			result[i] = bgLines[i]
		} else {
			result[i] = ""
		}
	}

	return strings.Join(result, "\n")
}
