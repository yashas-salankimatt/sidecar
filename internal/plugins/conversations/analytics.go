package conversations

import (
	"fmt"
	"strings"
	"time"

	"github.com/sst/sidecar/internal/adapter/claudecode"
	"github.com/sst/sidecar/internal/styles"
)

// renderAnalytics renders the global analytics view.
func (p *Plugin) renderAnalytics() string {
	var sb strings.Builder

	// Load stats
	stats, err := claudecode.LoadStatsCache()
	if err != nil {
		sb.WriteString(styles.PanelHeader.Render(" Usage Analytics"))
		sb.WriteString("\n")
		sb.WriteString(styles.Muted.Render(strings.Repeat("━", p.width-2)))
		sb.WriteString("\n")
		sb.WriteString(styles.Muted.Render(" Unable to load stats: " + err.Error()))
		return sb.String()
	}

	// Header
	sb.WriteString(styles.PanelHeader.Render(" Usage Analytics                                    [U to close]"))
	sb.WriteString("\n")
	sb.WriteString(styles.Muted.Render(strings.Repeat("━", p.width-2)))
	sb.WriteString("\n")

	// Summary line
	firstDate := stats.FirstSessionDate.Format("Jan 2")
	summary := fmt.Sprintf(" Since %s  │  %d sessions  │  %s messages",
		firstDate,
		stats.TotalSessions,
		formatLargeNumber(stats.TotalMessages))
	sb.WriteString(styles.Body.Render(summary))
	sb.WriteString("\n\n")

	// Weekly activity chart
	sb.WriteString(styles.PanelHeader.Render(" This Week's Activity"))
	sb.WriteString("\n")
	sb.WriteString(styles.Muted.Render(strings.Repeat("─", p.width-2)))
	sb.WriteString("\n")

	recentActivity := stats.GetRecentActivity(7)
	maxMsgs := 0
	for _, day := range recentActivity {
		if day.MessageCount > maxMsgs {
			maxMsgs = day.MessageCount
		}
	}

	for _, day := range recentActivity {
		date, _ := time.Parse("2006-01-02", day.Date)
		dayName := date.Format("Mon")
		bar := renderBar(day.MessageCount, maxMsgs, 16)
		line := fmt.Sprintf(" %s │ %s │ %5d msgs │ %2d sessions",
			dayName,
			bar,
			day.MessageCount,
			day.SessionCount)
		sb.WriteString(styles.Muted.Render(line))
		sb.WriteString("\n")
	}
	sb.WriteString("\n")

	// Model usage
	sb.WriteString(styles.PanelHeader.Render(" Model Usage"))
	sb.WriteString("\n")
	sb.WriteString(styles.Muted.Render(strings.Repeat("─", p.width-2)))
	sb.WriteString("\n")

	// Find max tokens for bar scaling
	var maxTokens int64
	for _, usage := range stats.ModelUsage {
		total := int64(usage.InputTokens) + int64(usage.OutputTokens)
		if total > maxTokens {
			maxTokens = total
		}
	}

	for model, usage := range stats.ModelUsage {
		shortName := modelShortName(model)
		if shortName == "" {
			continue
		}

		totalTokens := int64(usage.InputTokens) + int64(usage.OutputTokens)
		bar := renderBar64(totalTokens, maxTokens, 12)
		cost := claudecode.CalculateModelCost(model, usage)

		line := fmt.Sprintf(" %-6s │ %s │ %s in  %s out │ ~$%.0f",
			shortName,
			bar,
			formatLargeNumber64(int64(usage.InputTokens)),
			formatLargeNumber64(int64(usage.OutputTokens)),
			cost)
		sb.WriteString(styles.Muted.Render(line))
		sb.WriteString("\n")
	}
	sb.WriteString("\n")

	// Stats footer
	cacheEff := stats.CacheEfficiency()
	sb.WriteString(styles.Muted.Render(fmt.Sprintf(" Cache Efficiency: %.0f%%", cacheEff)))
	sb.WriteString("\n")

	// Peak hours
	peakHours := stats.GetPeakHours(3)
	if len(peakHours) > 0 {
		peakStr := " Peak Hours:"
		for i, ph := range peakHours {
			if i > 0 {
				peakStr += ","
			}
			peakStr += fmt.Sprintf(" %s:00", ph.Hour)
		}
		sb.WriteString(styles.Muted.Render(peakStr))
		sb.WriteString("\n")
	}

	// Longest session
	if stats.LongestSession.Duration > 0 {
		dur := time.Duration(stats.LongestSession.Duration) * time.Millisecond
		sb.WriteString(styles.Muted.Render(fmt.Sprintf(" Longest Session: %s", formatSessionDuration(dur))))
		sb.WriteString("\n")
	}

	// Total cost
	totalCost := stats.TotalCost()
	sb.WriteString(styles.Muted.Render(fmt.Sprintf(" Total Estimated Cost: ~$%.0f", totalCost)))
	sb.WriteString("\n")

	return sb.String()
}

// renderBar renders an ASCII bar chart segment.
func renderBar(value, max, width int) string {
	if max == 0 {
		return strings.Repeat("░", width)
	}
	filled := (value * width) / max
	if filled > width {
		filled = width
	}
	return strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
}

// renderBar64 renders an ASCII bar chart segment for int64 values.
func renderBar64(value, max int64, width int) string {
	if max == 0 {
		return strings.Repeat("░", width)
	}
	filled := int((value * int64(width)) / max)
	if filled > width {
		filled = width
	}
	return strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
}

// formatLargeNumber formats a number with K/M suffix.
func formatLargeNumber(n int) string {
	return formatLargeNumber64(int64(n))
}

// formatLargeNumber64 formats an int64 with K/M/B suffix.
func formatLargeNumber64(n int64) string {
	if n >= 1_000_000_000 {
		return fmt.Sprintf("%.1fB", float64(n)/1_000_000_000)
	}
	if n >= 1_000_000 {
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	}
	if n >= 1_000 {
		return fmt.Sprintf("%.1fK", float64(n)/1_000)
	}
	return fmt.Sprintf("%d", n)
}
