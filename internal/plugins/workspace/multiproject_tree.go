package workspace

import (
	"sort"
	"strings"
)

// Flatten rebuilds FlatItems from the current project tree state,
// respecting expanded/collapsed projects and any active filter.
func (t *MultiProjectTree) Flatten() {
	t.FlatItems = t.FlatItems[:0]

	filter := strings.ToLower(t.Filter)

	for pi := range t.Projects {
		proj := &t.Projects[pi]

		// Check if project or any children match filter
		if filter != "" && !strings.Contains(strings.ToLower(proj.Config.Name), filter) && !projectHasMatchingChildren(proj, filter) {
			continue
		}

		t.FlatItems = append(t.FlatItems, TreeItem{
			Kind:       TreeItemProject,
			ProjectIdx: pi,
			Depth:      0,
		})

		// When filter is active, force-expand matching projects
		expanded := proj.Expanded || (filter != "" && projectHasMatchingChildren(proj, filter))
		if !expanded {
			continue
		}

		// Add shells first (matches existing workspace sidebar order)
		for si := range proj.Shells {
			if filter != "" && !strings.Contains(strings.ToLower(proj.Shells[si].Name), filter) {
				continue
			}
			t.FlatItems = append(t.FlatItems, TreeItem{
				Kind:       TreeItemShell,
				ProjectIdx: pi,
				ItemIdx:    si,
				Depth:      1,
			})
		}

		// Add worktrees
		for wi := range proj.Worktrees {
			if filter != "" && !worktreeMatchesFilter(proj.Worktrees[wi], filter) {
				continue
			}
			t.FlatItems = append(t.FlatItems, TreeItem{
				Kind:       TreeItemWorktree,
				ProjectIdx: pi,
				ItemIdx:    wi,
				Depth:      1,
			})
		}
	}

	// Clamp cursor
	if t.Cursor >= len(t.FlatItems) {
		t.Cursor = len(t.FlatItems) - 1
	}
	if t.Cursor < 0 {
		t.Cursor = 0
	}
}

// ToggleExpand toggles the expanded state of a project at the given index.
func (t *MultiProjectTree) ToggleExpand(projectIdx int) {
	if projectIdx >= 0 && projectIdx < len(t.Projects) {
		t.Projects[projectIdx].Expanded = !t.Projects[projectIdx].Expanded
		t.Flatten()
	}
}

// CursorUp moves the cursor up one position.
func (t *MultiProjectTree) CursorUp() {
	if t.Cursor > 0 {
		t.Cursor--

	}
}

// CursorDown moves the cursor down one position.
func (t *MultiProjectTree) CursorDown() {
	if t.Cursor < len(t.FlatItems)-1 {
		t.Cursor++

	}
}

// JumpToTop moves cursor to the first item.
func (t *MultiProjectTree) JumpToTop() {
	t.Cursor = 0
	t.ScrollOffset = 0
}

// JumpToBottom moves cursor to the last item.
func (t *MultiProjectTree) JumpToBottom() {
	if len(t.FlatItems) > 0 {
		t.Cursor = len(t.FlatItems) - 1

	}
}

// CollapseOrParent collapses the current project, or if on a child item,
// moves cursor to the parent project header.
func (t *MultiProjectTree) CollapseOrParent() {
	if len(t.FlatItems) == 0 {
		return
	}
	item := t.FlatItems[t.Cursor]

	if item.Kind == TreeItemProject {
		// Collapse the project
		if item.ProjectIdx < len(t.Projects) && t.Projects[item.ProjectIdx].Expanded {
			t.Projects[item.ProjectIdx].Expanded = false
			t.Flatten()
		}
	} else {
		// Move to parent project header
		for i := t.Cursor - 1; i >= 0; i-- {
			if t.FlatItems[i].Kind == TreeItemProject {
				t.Cursor = i
		
				break
			}
		}
	}
}

// ExpandOrSelect expands a project header, or if on a child item returns true
// to indicate the caller should switch focus to the preview pane.
func (t *MultiProjectTree) ExpandOrSelect() bool {
	if len(t.FlatItems) == 0 {
		return false
	}
	item := t.FlatItems[t.Cursor]

	if item.Kind == TreeItemProject {
		if item.ProjectIdx < len(t.Projects) && !t.Projects[item.ProjectIdx].Expanded {
			t.Projects[item.ProjectIdx].Expanded = true
			t.Flatten()
			return false
		}
		// Already expanded - move to first child if any
		if t.Cursor+1 < len(t.FlatItems) && t.FlatItems[t.Cursor+1].Depth > 0 {
			t.Cursor++
	
		}
		return false
	}

	// On a worktree or shell - signal to focus preview
	return true
}

// SelectedItem returns the currently selected tree item, or nil if empty.
func (t *MultiProjectTree) SelectedItem() *TreeItem {
	if len(t.FlatItems) == 0 || t.Cursor >= len(t.FlatItems) {
		return nil
	}
	return &t.FlatItems[t.Cursor]
}

// SelectedProjectNode returns the project node for the currently selected item.
func (t *MultiProjectTree) SelectedProjectNode() *ProjectNode {
	item := t.SelectedItem()
	if item == nil || item.ProjectIdx >= len(t.Projects) {
		return nil
	}
	return &t.Projects[item.ProjectIdx]
}

// ApplyFilter sets the filter text and rebuilds the flat list.
func (t *MultiProjectTree) ApplyFilter(query string) {
	t.Filter = query
	t.Flatten()
}

// ApplySort sorts projects and their children, then rebuilds the flat list.
func (t *MultiProjectTree) ApplySort(mode MultiProjectSort) {
	t.SortMode = mode

	switch mode {
	case SortByName:
		sort.Slice(t.Projects, func(i, j int) bool {
			return strings.ToLower(t.Projects[i].Config.Name) < strings.ToLower(t.Projects[j].Config.Name)
		})
	case SortByStatus:
		sort.Slice(t.Projects, func(i, j int) bool {
			return t.Projects[i].ActiveCount() > t.Projects[j].ActiveCount()
		})
	case SortByConfig:
		// Keep config file order (no sorting)
	}

	t.Flatten()
}

// EnsureVisibleInHeight adjusts scroll so cursor is visible within the given height.
func (t *MultiProjectTree) EnsureVisibleInHeight(visibleRows int) {
	if visibleRows <= 0 {
		return
	}
	if t.Cursor < t.ScrollOffset {
		t.ScrollOffset = t.Cursor
	}
	if t.Cursor >= t.ScrollOffset+visibleRows {
		t.ScrollOffset = t.Cursor - visibleRows + 1
	}
}

// ExpandedProjectPaths returns the paths of all expanded projects (for state persistence).
func (t *MultiProjectTree) ExpandedProjectPaths() []string {
	var paths []string
	for _, proj := range t.Projects {
		if proj.Expanded {
			paths = append(paths, proj.Config.Path)
		}
	}
	return paths
}

// RestoreExpanded sets expanded state based on a list of project paths.
func (t *MultiProjectTree) RestoreExpanded(paths []string) {
	pathSet := make(map[string]bool, len(paths))
	for _, p := range paths {
		pathSet[p] = true
	}
	for i := range t.Projects {
		if pathSet[t.Projects[i].Config.Path] {
			t.Projects[i].Expanded = true
		}
	}
	t.Flatten()
}

// RestoreCursor tries to select the item matching the given project path and item name.
func (t *MultiProjectTree) RestoreCursor(projectPath, itemName string) {
	for i, item := range t.FlatItems {
		if item.ProjectIdx >= len(t.Projects) {
			continue
		}
		proj := &t.Projects[item.ProjectIdx]
		if proj.Config.Path != projectPath {
			continue
		}
		switch item.Kind {
		case TreeItemWorktree:
			if item.ItemIdx < len(proj.Worktrees) && proj.Worktrees[item.ItemIdx].Name == itemName {
				t.Cursor = i
				return
			}
		case TreeItemShell:
			if item.ItemIdx < len(proj.Shells) && proj.Shells[item.ItemIdx].Name == itemName {
				t.Cursor = i
				return
			}
		case TreeItemProject:
			if itemName == "" {
				t.Cursor = i
				return
			}
		}
	}
}

// --- Filter helpers ---

func projectHasMatchingChildren(proj *ProjectNode, filter string) bool {
	for _, wt := range proj.Worktrees {
		if worktreeMatchesFilter(wt, filter) {
			return true
		}
	}
	for _, sh := range proj.Shells {
		if strings.Contains(strings.ToLower(sh.Name), filter) {
			return true
		}
	}
	return false
}

func worktreeMatchesFilter(wt *Worktree, filter string) bool {
	return strings.Contains(strings.ToLower(wt.Name), filter) ||
		strings.Contains(strings.ToLower(wt.Branch), filter)
}
