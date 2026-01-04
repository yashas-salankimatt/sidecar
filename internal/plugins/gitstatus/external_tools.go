package gitstatus

import (
	"bytes"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"sync"
)

// terminalModeRegex matches terminal escape sequences that can interfere with bubbletea.
// This includes:
// - Mouse mode: \x1b[?1000h through \x1b[?1006h/l (enable/disable various mouse modes)
// - Alternate screen: \x1b[?1049h/l (switch to/from alternate screen buffer)
// - Bracketed paste: \x1b[?2004h/l (enable/disable bracketed paste mode)
var terminalModeRegex = regexp.MustCompile(`\x1b\[\?(100[0-6]|1049|2004)[hl]`)

// ExternalToolMode specifies which diff renderer to use.
type ExternalToolMode string

const (
	ToolModeAuto    ExternalToolMode = "auto"    // Auto-detect best tool
	ToolModeDelta   ExternalToolMode = "delta"   // Force delta
	ToolModeBuiltin ExternalToolMode = "builtin" // Force built-in renderer
)

// ExternalTool manages external diff rendering tools.
type ExternalTool struct {
	mode       ExternalToolMode
	deltaPath  string
	deltaMu    sync.Once
	tipShown   bool
	tipShownMu sync.Mutex
}

// NewExternalTool creates a new external tool manager.
func NewExternalTool(mode ExternalToolMode) *ExternalTool {
	return &ExternalTool{mode: mode}
}

// DetectDelta checks if delta is installed and returns its path.
func (e *ExternalTool) DetectDelta() string {
	e.deltaMu.Do(func() {
		path, err := exec.LookPath("delta")
		if err == nil {
			e.deltaPath = path
		}
	})
	return e.deltaPath
}

// HasDelta returns true if delta is available.
func (e *ExternalTool) HasDelta() bool {
	return e.DetectDelta() != ""
}

// ShouldUseDelta returns true if delta should be used for rendering.
func (e *ExternalTool) ShouldUseDelta() bool {
	switch e.mode {
	case ToolModeDelta:
		return e.HasDelta()
	case ToolModeBuiltin:
		return false
	default: // ToolModeAuto
		return e.HasDelta()
	}
}

// RenderWithDelta pipes raw diff through delta.
func (e *ExternalTool) RenderWithDelta(rawDiff string, sideBySide bool, width int) (string, error) {
	if e.deltaPath == "" {
		return rawDiff, nil
	}

	args := []string{"--paging=never"}
	if width > 0 {
		args = append(args, fmt.Sprintf("--width=%d", width))
	}
	if sideBySide {
		args = append(args, "--side-by-side")
	}

	cmd := exec.Command(e.deltaPath, args...)
	cmd.Stdin = strings.NewReader(rawDiff)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		// Fall back to raw diff on error
		return rawDiff, nil
	}

	// Strip any terminal mode escape sequences that could interfere with bubbletea
	output := terminalModeRegex.ReplaceAllString(stdout.String(), "")
	return output, nil
}

// ShouldShowTip returns true if we should show the delta install tip.
// Returns true only once per session.
func (e *ExternalTool) ShouldShowTip() bool {
	if e.HasDelta() {
		return false
	}

	e.tipShownMu.Lock()
	defer e.tipShownMu.Unlock()

	if e.tipShown {
		return false
	}
	e.tipShown = true
	return true
}

// GetTipMessage returns the install recommendation message.
func (e *ExternalTool) GetTipMessage() string {
	return "Tip: Install delta for enhanced diffs: brew install git-delta"
}

// Mode returns the current tool mode.
func (e *ExternalTool) Mode() ExternalToolMode {
	return e.mode
}

// SetMode updates the tool mode.
func (e *ExternalTool) SetMode(mode ExternalToolMode) {
	e.mode = mode
}
