package gitstatus

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/marcus/sidecar/internal/plugin"
	"github.com/marcus/sidecar/internal/styles"
	"github.com/marcus/sidecar/internal/ui"
)

// updateBranchPicker handles key events in the branch picker modal.
func (p *Plugin) updateBranchPicker(msg tea.KeyMsg) (plugin.Plugin, tea.Cmd) {
	switch msg.String() {
	case "esc", "q":
		// Close picker
		p.viewMode = p.branchReturnMode
		p.branches = nil
		return p, nil

	case "j", "down":
		if len(p.branches) > 0 && p.branchCursor < len(p.branches)-1 {
			p.branchCursor++
		}
		return p, nil

	case "k", "up":
		if p.branchCursor > 0 {
			p.branchCursor--
		}
		return p, nil

	case "g":
		p.branchCursor = 0
		return p, nil

	case "G":
		if len(p.branches) > 0 {
			p.branchCursor = len(p.branches) - 1
		}
		return p, nil

	case "enter":
		// Switch to selected branch
		if len(p.branches) > 0 && p.branchCursor < len(p.branches) {
			branch := p.branches[p.branchCursor]
			if !branch.IsCurrent {
				return p, p.doSwitchBranch(branch.Name)
			}
		}
		return p, nil
	}

	return p, nil
}

// doSwitchBranch switches to a different branch.
func (p *Plugin) doSwitchBranch(branchName string) tea.Cmd {
	workDir := p.ctx.WorkDir
	return func() tea.Msg {
		err := CheckoutBranch(workDir, branchName)
		if err != nil {
			return BranchErrorMsg{Err: err}
		}
		return BranchSwitchSuccessMsg{Branch: branchName}
	}
}

// loadBranches loads the branch list.
func (p *Plugin) loadBranches() tea.Cmd {
	workDir := p.ctx.WorkDir
	return func() tea.Msg {
		branches, err := GetBranches(workDir)
		if err != nil {
			return BranchErrorMsg{Err: err}
		}
		return BranchListLoadedMsg{Branches: branches}
	}
}

// renderBranchPicker renders the branch picker modal.
func (p *Plugin) renderBranchPicker() string {
	// Render the background (status view dimmed)
	background := p.renderThreePaneView()

	var sb strings.Builder

	// Title
	title := styles.Title.Render(" Branches ")
	sb.WriteString(title)
	sb.WriteString("\n\n")

	if len(p.branches) == 0 {
		sb.WriteString(styles.Muted.Render("  Loading branches..."))
	} else {
		// Calculate visible range (max 15 branches visible)
		maxVisible := 15
		if p.height-10 < maxVisible {
			maxVisible = p.height - 10
		}
		if maxVisible < 5 {
			maxVisible = 5
		}

		start := 0
		if p.branchCursor >= maxVisible {
			start = p.branchCursor - maxVisible + 1
		}
		end := start + maxVisible
		if end > len(p.branches) {
			end = len(p.branches)
		}

		for i := start; i < end; i++ {
			branch := p.branches[i]
			selected := i == p.branchCursor

			line := p.renderBranchLine(branch, selected)
			sb.WriteString(line)
			if i < end-1 {
				sb.WriteString("\n")
			}
		}

		// Scroll indicator
		if len(p.branches) > maxVisible {
			sb.WriteString("\n\n")
			sb.WriteString(styles.Muted.Render(fmt.Sprintf("  %d/%d branches", p.branchCursor+1, len(p.branches))))
		}
	}

	sb.WriteString("\n\n")
	sb.WriteString(styles.Muted.Render("  Enter to switch, Esc to cancel"))

	// Calculate modal width
	modalWidth := 50
	for _, b := range p.branches {
		lineLen := len(b.Name) + 10
		if lineLen > modalWidth {
			modalWidth = lineLen
		}
	}
	if modalWidth > p.width-10 {
		modalWidth = p.width - 10
	}

	modalContent := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.Primary).
		Padding(1, 2).
		Width(modalWidth).
		Render(sb.String())

	return ui.OverlayModal(background, modalContent, p.width, p.height)
}

// renderBranchLine renders a single branch line.
func (p *Plugin) renderBranchLine(branch *Branch, selected bool) string {
	// Current branch indicator
	indicator := "  "
	if branch.IsCurrent {
		indicator = "* "
	}

	// Branch name
	name := branch.Name

	// Tracking info
	trackingInfo := branch.FormatTrackingInfo()
	if trackingInfo != "" {
		trackingInfo = " " + styles.StatusModified.Render(trackingInfo)
	}

	// Upstream indicator
	upstream := ""
	if branch.Upstream != "" {
		upstream = styles.Muted.Render(" → " + branch.Upstream)
	}

	if selected {
		// Build plain text and pad
		plainLine := fmt.Sprintf("%s%s", indicator, name)
		if branch.FormatTrackingInfo() != "" {
			plainLine += " " + branch.FormatTrackingInfo()
		}
		maxWidth := 45
		if len(plainLine) < maxWidth {
			plainLine += strings.Repeat(" ", maxWidth-len(plainLine))
		}
		return styles.ListItemSelected.Render(plainLine)
	}

	// Style based on current branch
	nameStyle := styles.Body
	if branch.IsCurrent {
		nameStyle = styles.StatusStaged
	}

	return styles.ListItemNormal.Render(fmt.Sprintf("%s%s%s%s", indicator, nameStyle.Render(name), trackingInfo, upstream))
}
