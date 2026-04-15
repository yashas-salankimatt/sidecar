package workspace

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/marcus/sidecar/internal/config"
	"github.com/marcus/sidecar/internal/styles"
	"github.com/marcus/sidecar/internal/ui"
)

// mpHitRegionOffset distinguishes multi-project tree items from regular shell indices
// in regionWorktreeItem hit regions. Regular shells use -1..-N, multi-project uses
// -(flatIdx + mpHitRegionOffset) so they don't collide.
const mpHitRegionOffset = 10000

// renderMultiProjectView renders the multi-project view: project tree sidebar + existing preview pane.
// Only the sidebar changes — the preview pane is the same Output/Diff/Task tabs with Ctrl+T terminal panel.
func (p *Plugin) renderMultiProjectView(width, height int) string {
	if p.mpTree == nil {
		msg := "Loading projects..."
		pad := (width - len(msg)) / 2
		if pad < 0 {
			pad = 0
		}
		return strings.Repeat("\n", height/2) + strings.Repeat(" ", pad) + styles.Muted.Render(msg)
	}

	paneHeight := height
	if paneHeight < 4 {
		paneHeight = 4
	}
	innerHeight := paneHeight - 2
	if innerHeight < 1 {
		innerHeight = 1
	}

	// Clamp selectedIdx
	if p.selectedIdx >= len(p.worktrees) && len(p.worktrees) > 0 {
		p.selectedIdx = 0
	}

	// If sidebar is hidden, show only preview pane at full width
	if !p.sidebarVisible {
		previewW := width
		contentW := previewW - panelOverhead
		p.mouseHandler.HitMap.AddRect(regionPreviewPane, 0, 0, previewW, paneHeight, nil)
		if !p.shellSelected {
			tabWidths := []int{10, 8, 8}
			tabX := panelOverhead / 2
			for i, tabWidth := range tabWidths {
				p.mouseHandler.HitMap.AddRect(regionPreviewTab, tabX, 1, tabWidth, 1, i)
				tabX += tabWidth + 1
			}
		}
		previewContent := p.renderPreviewContent(contentW, innerHeight)
		return styles.RenderPanel(previewContent, previewW, paneHeight, true)
	}

	// Calculate pane widths
	available := width - dividerWidth
	sidebarW := (available * p.sidebarWidth) / 100
	if sidebarW < 15 {
		sidebarW = 15
	}
	if sidebarW > available-40 {
		sidebarW = available - 40
	}
	previewW := available - sidebarW
	if previewW < 40 {
		previewW = 40
	}

	sidebarContentW := sidebarW - panelOverhead
	previewContentW := previewW - panelOverhead

	sidebarActive := p.activePane == PaneSidebar
	previewActive := p.activePane == PanePreview

	// Register hit regions
	p.mouseHandler.HitMap.AddRect(regionSidebar, 0, 0, sidebarW, paneHeight, nil)
	p.mouseHandler.HitMap.AddRect(regionPreviewPane, sidebarW+dividerWidth, 0, previewW, paneHeight, nil)
	p.mouseHandler.HitMap.AddRect(regionPaneDivider, sidebarW, 0, dividerHitWidth, paneHeight, nil)

	// Register preview tab hit regions
	if !p.shellSelected {
		previewPaneX := sidebarW + dividerWidth + panelOverhead/2
		tabWidths := []int{10, 8, 8}
		tabX := previewPaneX
		for i, tabWidth := range tabWidths {
			p.mouseHandler.HitMap.AddRect(regionPreviewTab, tabX, 1, tabWidth, 1, i)
			tabX += tabWidth + 1
		}
	}

	// Render
	sidebarContent := p.renderMPSidebar(sidebarContentW, innerHeight)
	previewContent := p.renderPreviewContent(previewContentW, innerHeight)

	flashActive := !p.flashPreviewTime.IsZero() && time.Since(p.flashPreviewTime) < flashDuration

	leftPane := styles.RenderPanel(sidebarContent, sidebarW, paneHeight, sidebarActive)
	var rightPane string
	if p.viewMode == ViewModeInteractive {
		rightPane = styles.RenderPanelWithGradient(previewContent, previewW, paneHeight, styles.GetInteractiveGradient())
	} else if flashActive && previewActive {
		rightPane = styles.RenderPanelWithGradient(previewContent, previewW, paneHeight, styles.GetFlashGradient())
	} else {
		rightPane = styles.RenderPanel(previewContent, previewW, paneHeight, previewActive)
	}
	divider := ui.RenderDivider(paneHeight)

	return lipgloss.JoinHorizontal(lipgloss.Top, leftPane, divider, rightPane)
}

// renderMPSidebar renders the multi-project tree sidebar using the same
// item renderers as the regular workspace sidebar.
func (p *Plugin) renderMPSidebar(width, height int) string {
	tree := p.mpTree
	if tree == nil || len(tree.FlatItems) == 0 {
		return p.renderMPEmptyState(width, height)
	}

	var lines []string

	// Header
	titleText := "Projects"
	sortLabel := tree.SortMode.String()
	sortBadge := styles.Muted.Render("[" + sortLabel + "]")
	sortWidth := lipgloss.Width(sortBadge)
	titleWidth := lipgloss.Width(titleText)
	spacing := width - titleWidth - sortWidth
	if spacing < 1 {
		spacing = 1
	}
	header := styles.Title.Render(titleText) + strings.Repeat(" ", spacing) + sortBadge
	lines = append(lines, header)

	// Filter bar
	if p.mpFilterActive {
		filterLine := "/ " + p.mpFilterInput.View()
		lines = append(lines, filterLine)
	}

	lines = append(lines, "") // spacing

	headerLines := len(lines)
	contentHeight := height - headerLines
	if contentHeight < 1 {
		contentHeight = 1
	}

	tree.EnsureVisibleInHeight(contentHeight)

	// Render tree items using existing item renderers
	currentY := headerLines + 1 // +1 for border
	itemLines := make([]string, 0, contentHeight)
	rendered := 0
	itemWidth := width - 1 // leave room for scrollbar

	for i := tree.ScrollOffset; i < len(tree.FlatItems) && rendered < contentHeight; i++ {
		item := tree.FlatItems[i]
		isSelected := i == tree.Cursor

		switch item.Kind {
		case TreeItemProject:
			line := p.renderMPProjectRow(item, isSelected, itemWidth)
			itemLines = append(itemLines, line)
			// Encode flat index as negative offset so the mouse handler can
			// distinguish multi-project tree items from regular worktree indices.
			// Scheme: -(flatIndex + mpHitRegionOffset)
			p.mouseHandler.HitMap.AddRect(regionWorktreeItem, 0, currentY, width, 1, -(i + mpHitRegionOffset))
			currentY++
			rendered++

		case TreeItemWorktree:
			if rendered+2 > contentHeight {
				break // Don't render a 2-line item if it would overflow
			}
			proj := &tree.Projects[item.ProjectIdx]
			if item.ItemIdx < len(proj.Worktrees) {
				wt := proj.Worktrees[item.ItemIdx]
				line := p.renderWorktreeItem(wt, isSelected, itemWidth)
				itemLines = append(itemLines, line)
				p.mouseHandler.HitMap.AddRect(regionWorktreeItem, 0, currentY, width, 2, -(i + mpHitRegionOffset))
				currentY += 2
				rendered += 2
			}

		case TreeItemShell:
			if rendered+2 > contentHeight {
				break // Don't render a 2-line item if it would overflow
			}
			proj := &tree.Projects[item.ProjectIdx]
			if item.ItemIdx < len(proj.Shells) {
				shell := proj.Shells[item.ItemIdx]
				line := p.renderShellEntryForSession(shell, isSelected, itemWidth)
				itemLines = append(itemLines, line)
				p.mouseHandler.HitMap.AddRect(regionWorktreeItem, 0, currentY, width, 2, -(i + mpHitRegionOffset))
				currentY += 2
				rendered += 2
			}
		}
	}

	// Scrollbar
	scrollbar := ui.RenderScrollbar(ui.ScrollbarParams{
		TotalItems:   len(tree.FlatItems),
		ScrollOffset: tree.ScrollOffset,
		VisibleItems: contentHeight,
		TrackHeight:  contentHeight,
	})
	content := strings.Join(itemLines, "\n")
	contentWithScrollbar := lipgloss.JoinHorizontal(lipgloss.Top, content, scrollbar)
	lines = append(lines, contentWithScrollbar)

	return strings.Join(lines, "\n")
}

// renderMPProjectRow renders a project header row in the tree.
func (p *Plugin) renderMPProjectRow(item TreeItem, selected bool, width int) string {
	if item.ProjectIdx >= len(p.mpTree.Projects) {
		return ""
	}
	proj := &p.mpTree.Projects[item.ProjectIdx]

	caret := "▶"
	if proj.Expanded {
		caret = "▼"
	}

	name := proj.Config.Name
	if name == "" {
		name = config.ExpandPath(proj.Config.Path)
	}

	total := proj.ItemCount()
	active := proj.ActiveCount()
	badge := ""
	if total > 0 {
		badge = fmt.Sprintf(" %d/%d", active, total)
	}

	currentMark := ""
	if proj.IsCurrent {
		currentMark = " ←"
	}

	if proj.ScanErr != nil {
		name += " ✗"
	}

	rightContent := badge + currentMark
	rightWidth := lipgloss.Width(rightContent)
	caretWidth := lipgloss.Width(caret)
	nameMaxWidth := width - caretWidth - 1 - rightWidth
	if nameMaxWidth < 4 {
		nameMaxWidth = 4
	}
	truncName := p.truncateCache.Truncate(name, nameMaxWidth, "…")

	line := caret + " " + truncName + strings.Repeat(" ", max(0, width-caretWidth-1-lipgloss.Width(truncName)-rightWidth)) + rightContent

	if selected && p.activePane == PaneSidebar {
		return styles.ListItemFocused.Width(width).Bold(true).Render(line)
	}
	if selected {
		return styles.ListItemSelected.Width(width).Bold(true).Render(line)
	}
	return lipgloss.NewStyle().Bold(true).Width(width).Render(line)
}

// renderMPEmptyState renders the empty state when no projects are configured.
func (p *Plugin) renderMPEmptyState(width, height int) string {
	center := func(s string) string {
		pad := max(0, (width-len(s))/2)
		return strings.Repeat(" ", pad) + styles.Muted.Render(s)
	}
	return strings.Repeat("\n", height/2) + center("No projects configured") + "\n" + center("Press @ to add projects")
}


