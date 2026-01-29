package app

import (
	"encoding/json"
	"os/exec"

	tea "github.com/charmbracelet/bubbletea"
)

// IssuePreviewData holds lightweight issue data fetched via CLI.
type IssuePreviewData struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Status      string   `json:"status"`
	Type        string   `json:"type"`
	Priority    string   `json:"priority"`
	Points      int      `json:"points"`
	Description string   `json:"description"`
	ParentID    string   `json:"parent_id"`
	Labels      []string `json:"labels"`
	CreatedAt   string   `json:"created_at"`
	UpdatedAt   string   `json:"updated_at"`
}

// IssuePreviewResultMsg carries fetched issue data back to the app.
type IssuePreviewResultMsg struct {
	Data  *IssuePreviewData
	Error error
}

// OpenFullIssueMsg is broadcast to plugins to open the full rich issue view.
// Currently handled by the TD monitor plugin via monitor.OpenIssueByIDMsg.
type OpenFullIssueMsg struct {
	IssueID string
}

// fetchIssuePreviewCmd runs `td show <id> -f json` and returns the result.
func fetchIssuePreviewCmd(issueID string) tea.Cmd {
	return func() tea.Msg {
		out, err := exec.Command("td", "show", issueID, "-f", "json").Output()
		if err != nil {
			return IssuePreviewResultMsg{Error: err}
		}
		var data IssuePreviewData
		if err := json.Unmarshal(out, &data); err != nil {
			return IssuePreviewResultMsg{Error: err}
		}
		return IssuePreviewResultMsg{Data: &data}
	}
}
