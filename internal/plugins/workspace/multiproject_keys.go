package workspace

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/marcus/sidecar/internal/app"
	"github.com/marcus/sidecar/internal/config"
	"github.com/marcus/sidecar/internal/state"
)

// enterMultiProjectView switches to the multi-project view, initializing if needed.
func (p *Plugin) enterMultiProjectView() tea.Cmd {
	p.viewMode = ViewModeMultiProject
	p.activePane = PaneSidebar

	if p.mpTree == nil {
		ti := textinput.New()
		ti.Placeholder = "Filter projects..."
		ti.CharLimit = 100
		p.mpFilterInput = ti
		p.mpTree = &MultiProjectTree{
			SortMode: MultiProjectSort(state.GetMultiProjectState().SortMode),
		}
	}

	p.mpScanGeneration++
	return p.scanAllProjects()
}

// exitMultiProjectView switches back to the list view.
func (p *Plugin) exitMultiProjectView() {
	p.saveMultiProjectState()
	p.viewMode = ViewModeList
	p.activePane = PaneSidebar
}

// handleMultiProjectKeys handles key input in the multi-project view.
func (p *Plugin) handleMultiProjectKeys(msg tea.KeyMsg) tea.Cmd {
	if p.mpTree == nil {
		return nil
	}

	if p.mpFilterActive {
		return p.handleMPFilterKeys(msg)
	}

	// When preview pane is focused, delegate safe read-only keys to the list
	// key handler for Output/Diff/Task tabs, scrolling, Ctrl+T, interactive mode.
	// Block destructive keys (delete, stop, push, merge) that would operate on
	// a foreign project's worktrees.
	if p.activePane == PanePreview {
		key := msg.String()
		switch key {
		case "P":
			p.exitMultiProjectView()
			return nil
		case "tab":
			p.activePane = PaneSidebar
			return nil
		case "D", "K", "p", "m", "n", "T", "F", "O", "r", "y", "Y", "N":
			// Block destructive/mutation keys on foreign-project worktrees
			return nil
		default:
			return p.handleListKeys(msg)
		}
	}

	key := msg.String()

	switch key {
	case "P":
		p.exitMultiProjectView()
		return nil

	case "esc":
		item := p.mpTree.SelectedItem()
		if item != nil && item.Kind == TreeItemProject {
			if item.ProjectIdx < len(p.mpTree.Projects) && p.mpTree.Projects[item.ProjectIdx].Expanded {
				p.mpTree.Projects[item.ProjectIdx].Expanded = false
				p.mpTree.Flatten()
			}
		} else if item != nil {
			p.mpTree.CollapseOrParent()
		}
		return nil

	case "j", "down":
		p.mpTree.CursorDown()
		return p.mpOnCursorMove()

	case "k", "up":
		p.mpTree.CursorUp()
		return p.mpOnCursorMove()

	case "enter", "l", "right":
		if p.mpTree.ExpandOrSelect() {
			p.activePane = PanePreview
		}
		return p.mpOnCursorMove()

	case "h", "left":
		p.mpTree.CollapseOrParent()
		return nil

	case " ":
		item := p.mpTree.SelectedItem()
		if item != nil && item.Kind == TreeItemProject {
			p.mpTree.ToggleExpand(item.ProjectIdx)
		}
		return nil

	case "tab":
		p.activePane = PanePreview
		return nil

	case "g":
		p.mpTree.JumpToTop()
		return nil

	case "G":
		p.mpTree.JumpToBottom()
		return nil

	case "/":
		p.mpFilterActive = true
		p.mpFilterInput.Focus()
		return nil

	case "s":
		nextSort := (p.mpTree.SortMode + 1) % 3
		p.mpTree.ApplySort(nextSort)
		return nil

	case "S":
		// Switch the entire sidecar instance to the selected project
		return p.mpSwitchToSelectedProject()

	case "n":
		item := p.mpTree.SelectedItem()
		if item != nil {
			proj := p.mpTree.SelectedProjectNode()
			if proj != nil {
				p.mpCreateTargetPath = config.ExpandPath(proj.Config.Path)
				p.viewMode = ViewModeTypeSelector
			}
		}
		return nil

	case "1", "2", "3", "4", "5", "6", "7", "8", "9":
		// Jump to Nth visible item (scroll-offset relative)
		visibleIdx := int(key[0]-'0') - 1
		flatIdx := p.mpTree.ScrollOffset + visibleIdx
		if flatIdx >= 0 && flatIdx < len(p.mpTree.FlatItems) {
			p.mpTree.Cursor = flatIdx
			return p.mpOnCursorMove()
		}
		return nil
	}

	return nil
}

// handleMPFilterKeys handles keys when the filter input is active.
func (p *Plugin) handleMPFilterKeys(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "esc":
		if p.mpFilterInput.Value() != "" {
			p.mpFilterInput.SetValue("")
			p.mpTree.ApplyFilter("")
		} else {
			p.mpFilterActive = false
			p.mpFilterInput.Blur()
		}
		return nil
	case "enter":
		p.mpFilterActive = false
		p.mpFilterInput.Blur()
		return nil
	default:
		var cmd tea.Cmd
		p.mpFilterInput, cmd = p.mpFilterInput.Update(msg)
		p.mpTree.ApplyFilter(p.mpFilterInput.Value())
		return cmd
	}
}

// mpOnCursorMove handles side effects when the cursor moves to a new item.
// This is the core of the simplified multi-project view: just populate
// p.worktrees/p.shells from whichever project the cursor is on, and set
// selectedIdx. The preview updates instantly. No project switching.
func (p *Plugin) mpOnCursorMove() tea.Cmd {
	item := p.mpTree.SelectedItem()
	if item == nil {
		return nil
	}

	proj := p.mpTree.SelectedProjectNode()
	if proj == nil {
		return nil
	}

	// Populate worktrees/shells from this project's scan data
	p.worktrees = make([]*Worktree, len(proj.Worktrees))
	copy(p.worktrees, proj.Worktrees)
	p.shells = make([]*ShellSession, len(proj.Shells))
	copy(p.shells, proj.Shells)

	// Sync selectedIdx to the cursor item
	switch item.Kind {
	case TreeItemProject:
		p.selectedIdx = 0
		p.shellSelected = false
	case TreeItemWorktree:
		if item.ItemIdx < len(p.worktrees) {
			p.selectedIdx = item.ItemIdx
			p.shellSelected = false
		}
	case TreeItemShell:
		if item.ItemIdx < len(p.shells) {
			p.selectedShellIdx = item.ItemIdx
			p.shellSelected = true
		}
	}

	// Clamp
	if p.selectedIdx >= len(p.worktrees) && len(p.worktrees) > 0 {
		p.selectedIdx = 0
	}

	// Start polling the selected item's tmux session so the preview has output
	return p.mpPollSelected()
}

// mpPollDoneMsg is a no-op message that triggers a re-render after a
// multi-project poll completes. Must NOT use RefreshMsg which would
// overwrite p.worktrees with the current project's data.
type mpPollDoneMsg struct{}

// mpPollSelected starts a one-shot poll for the currently selected item's tmux session.
func (p *Plugin) mpPollSelected() tea.Cmd {
	var agent *Agent
	if p.shellSelected {
		if sh := p.getSelectedShell(); sh != nil {
			agent = sh.Agent
		}
	} else if wt := p.selectedWorktree(); wt != nil {
		agent = wt.Agent
	}
	if agent == nil || agent.TmuxSession == "" {
		return nil
	}
	sessionName := agent.TmuxSession
	buf := agent.OutputBuf
	return func() tea.Msg {
		output, err := capturePaneDirect(sessionName)
		if err == nil && buf != nil {
			buf.Update(output)
		}
		return mpPollDoneMsg{}
	}
}

// mpSwitchToSelectedProject does a full project switch to the selected project.
// This changes the entire sidecar instance's context (header, git, file browser, etc.).
func (p *Plugin) mpSwitchToSelectedProject() tea.Cmd {
	item := p.mpTree.SelectedItem()
	if item == nil {
		return nil
	}

	proj := p.mpTree.SelectedProjectNode()
	if proj == nil {
		return nil
	}

	// Determine target path
	targetPath := config.ExpandPath(proj.Config.Path)
	if item.Kind == TreeItemWorktree && item.ItemIdx < len(proj.Worktrees) {
		wt := proj.Worktrees[item.ItemIdx]
		if wt.Path != "" {
			targetPath = wt.Path
		}
	}

	// Exit the view (which saves state) and switch
	p.exitMultiProjectView()

	return func() tea.Msg {
		return app.SwitchWorktreeMsg{
			WorktreePath:        targetPath,
			SkipWorktreeRestore: true,
		}
	}
}

// saveMultiProjectState persists the multi-project view state.
func (p *Plugin) saveMultiProjectState() {
	if p.mpTree == nil {
		return
	}

	s := state.MultiProjectViewState{
		Active:           p.viewMode == ViewModeMultiProject,
		ExpandedProjects: p.mpTree.ExpandedProjectPaths(),
		SortMode:         int(p.mpTree.SortMode),
	}

	item := p.mpTree.SelectedItem()
	if item != nil {
		proj := p.mpTree.SelectedProjectNode()
		if proj != nil {
			s.SelectedProject = proj.Config.Path
			switch item.Kind {
			case TreeItemWorktree:
				if item.ItemIdx < len(proj.Worktrees) {
					s.SelectedItem = proj.Worktrees[item.ItemIdx].Name
				}
			case TreeItemShell:
				if item.ItemIdx < len(proj.Shells) {
					s.SelectedItem = proj.Shells[item.ItemIdx].Name
				}
			}
		}
	}

	_ = state.SetMultiProjectState(s)
}

// handleMultiProjectScanDone processes scan results for the multi-project view.
func (p *Plugin) handleMultiProjectScanDone(msg MultiProjectScanDoneMsg) tea.Cmd {
	if msg.Generation != p.mpScanGeneration {
		return nil
	}
	if p.mpTree == nil {
		return nil
	}

	// Mark the current project
	for i := range msg.Projects {
		expandedPath := config.ExpandPath(msg.Projects[i].Config.Path)
		mainPath := app.GetMainWorktreePath(expandedPath)
		if mainPath == "" {
			mainPath = expandedPath
		}
		if mainPath == p.ctx.ProjectRoot || expandedPath == p.ctx.ProjectRoot {
			msg.Projects[i].IsCurrent = true
			msg.Projects[i].Expanded = true
		}
	}

	p.mpTree.Projects = msg.Projects

	// Restore saved state
	saved := state.GetMultiProjectState()
	if len(saved.ExpandedProjects) > 0 {
		p.mpTree.RestoreExpanded(saved.ExpandedProjects)
	}

	p.mpTree.ApplySort(p.mpTree.SortMode)
	p.mpTree.Flatten()

	if saved.SelectedProject != "" {
		p.mpTree.RestoreCursor(saved.SelectedProject, saved.SelectedItem)
	}

	// Populate worktrees from the cursor's project so preview works immediately
	return p.mpOnCursorMove()
}
