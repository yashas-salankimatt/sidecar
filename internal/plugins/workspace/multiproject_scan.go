package workspace

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/marcus/sidecar/internal/app"
	"github.com/marcus/sidecar/internal/config"
	"github.com/marcus/sidecar/internal/projectdir"
)

// scanAllProjects returns a command that scans all configured projects
// for worktrees and shell sessions.
func (p *Plugin) scanAllProjects() tea.Cmd {
	cfg := p.ctx.Config
	if cfg == nil {
		return nil
	}
	projects := cfg.Projects.List
	gen := p.mpScanGeneration

	return func() tea.Msg {
		nodes := make([]ProjectNode, 0, len(projects))
		for _, proj := range projects {
			node := scanProject(proj)
			nodes = append(nodes, node)
		}
		return MultiProjectScanDoneMsg{
			Generation: gen,
			Projects:   nodes,
		}
	}
}

// scanProject scans a single project for worktrees and shell sessions.
func scanProject(proj config.ProjectConfig) ProjectNode {
	node := ProjectNode{
		Config:   proj,
		Expanded: false,
	}

	projectPath := config.ExpandPath(proj.Path)
	if _, err := os.Stat(projectPath); err != nil {
		node.ScanErr = err
		return node
	}

	// Resolve the main worktree path (handles linked worktrees)
	mainPath := app.GetMainWorktreePath(projectPath)
	if mainPath == "" {
		mainPath = projectPath
	}

	// Discover git worktrees
	worktrees, err := scanProjectWorktrees(projectPath, mainPath)
	if err != nil {
		node.ScanErr = err
		// Still try to discover shells even if git fails
	} else {
		node.Worktrees = worktrees
	}

	// Load shell manifest and discover shell sessions
	shells := scanProjectShells(projectPath, mainPath)
	node.Shells = shells

	return node
}

// scanProjectWorktrees runs `git worktree list --porcelain` for a project.
func scanProjectWorktrees(projectPath, mainPath string) ([]*Worktree, error) {
	cmd := exec.Command("git", "worktree", "list", "--porcelain")
	cmd.Dir = projectPath
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	worktrees, err := parseWorktreeList(string(output), mainPath)
	if err != nil {
		return nil, err
	}

	// Filter missing worktrees and load metadata
	filtered := make([]*Worktree, 0, len(worktrees))
	for _, wt := range worktrees {
		if !wt.IsMain {
			if _, statErr := os.Stat(wt.Path); os.IsNotExist(statErr) {
				wt.IsMissing = true
				if !branchExists(projectPath, wt.Branch) {
					continue // fully gone, skip
				}
			}
		}

		// Load persisted metadata
		wt.ChosenAgentType = loadAgentType(mainPath, wt.Path)
		wt.PRURL = loadPRURL(mainPath, wt.Path)
		wt.TaskID = loadTaskLink(mainPath, wt.Path)
		wt.BaseBranch = loadBaseBranch(mainPath, wt.Path)

		// Check for running tmux session
		sessionName := tmuxSessionPrefix + sanitizeName(wt.Name)
		if sessionExists(sessionName) {
			paneID := getPaneID(sessionName)
			agentType := wt.ChosenAgentType
			if agentType == AgentNone {
				agentType = AgentClaude // default display
			}
			wt.Agent = &Agent{
				Type:        agentType,
				TmuxSession: sessionName,
				TmuxPane:    paneID,
				OutputBuf:   NewOutputBuffer(outputBufferCap),
				StartedAt:   time.Now(),
				Status:      AgentStatusRunning,
			}
			wt.Status = StatusActive
		} else {
			wt.Status = StatusPaused
		}

		filtered = append(filtered, wt)
	}

	return filtered, nil
}

// scanProjectShells discovers shell sessions for a project.
func scanProjectShells(projectPath, mainPath string) []*ShellSession {
	// Load shell manifest
	projDir, err := projectdir.Resolve(mainPath)
	if err != nil {
		return nil
	}
	manifestPath := filepath.Join(projDir, "shells.json")
	manifest, err := LoadShellManifest(manifestPath)
	if err != nil {
		return nil
	}

	// Discover running tmux sessions for this project
	projectName := filepath.Base(projectPath)
	basePrefix := shellSessionPrefix + sanitizeName(projectName)
	tmuxSessions := discoverShellSessions(basePrefix)
	tmuxMap := make(map[string]bool)
	for _, name := range tmuxSessions {
		tmuxMap[name] = true
	}

	var shells []*ShellSession

	// Process manifest entries
	for _, def := range manifest.Shells {
		isRunning := tmuxMap[def.TmuxName]
		if !isRunning {
			continue // Skip dead sessions in multi-project view
		}
		paneID := getPaneID(def.TmuxName)
		displayType := AgentShell
		chosenAgent := definitionToAgentType(def.AgentType)
		if chosenAgent != AgentNone {
			displayType = chosenAgent
		}
		shell := &ShellSession{
			Name:        def.DisplayName,
			TmuxName:    def.TmuxName,
			CreatedAt:   def.CreatedAt,
			ChosenAgent: chosenAgent,
			SkipPerms:   def.SkipPerms,
			Agent: &Agent{
				Type:        displayType,
				TmuxSession: def.TmuxName,
				TmuxPane:    paneID,
				OutputBuf:   NewOutputBuffer(outputBufferCap),
				StartedAt:   def.CreatedAt,
				Status:      AgentStatusRunning,
			},
		}
		shells = append(shells, shell)
		delete(tmuxMap, def.TmuxName)
	}

	// Add tmux sessions not in manifest
	for tmuxName := range tmuxMap {
		paneID := getPaneID(tmuxName)
		shells = append(shells, &ShellSession{
			Name:     deriveShellDisplayName(tmuxName, basePrefix),
			TmuxName: tmuxName,
			Agent: &Agent{
				Type:        AgentShell,
				TmuxSession: tmuxName,
				TmuxPane:    paneID,
				OutputBuf:   NewOutputBuffer(outputBufferCap),
				StartedAt:   time.Now(),
				Status:      AgentStatusRunning,
			},
			CreatedAt: time.Now(),
		})
	}

	return shells
}

// discoverShellSessions returns tmux session names matching the shell prefix for a project.
func discoverShellSessions(basePrefix string) []string {
	cmd := exec.Command("tmux", "list-sessions", "-F", "#{session_name}")
	output, err := cmd.Output()
	if err != nil {
		return nil
	}

	var result []string
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, basePrefix) {
			result = append(result, line)
		}
	}
	return result
}

// deriveShellDisplayName extracts a display name from a tmux session name.
func deriveShellDisplayName(tmuxName, basePrefix string) string {
	suffix := strings.TrimPrefix(strings.TrimPrefix(tmuxName, basePrefix), "-")
	if suffix == "" {
		return "Shell 1"
	}
	return "Shell " + suffix
}
