package pi

// AggregatedUsage holds summed token counts and cost across messages.
type AggregatedUsage struct {
	InputTokens  int
	OutputTokens int
	CacheRead    int
	CacheWrite   int
	TotalCost    float64
}

// aggregateUsage sums token counts and pre-calculated costs from assistant
// message lines. Non-assistant lines and lines without usage data are skipped.
func aggregateUsage(lines []RawLine) AggregatedUsage {
	var agg AggregatedUsage
	for i := range lines {
		u := messageUsage(&lines[i])
		if u == nil {
			continue
		}
		agg.InputTokens += u.Input
		agg.OutputTokens += u.Output
		agg.CacheRead += u.CacheRead
		agg.CacheWrite += u.CacheWrite
		if u.Cost != nil {
			agg.TotalCost += u.Cost.Total
		}
	}
	return agg
}

// messageUsage returns the Usage pointer for a message line, or nil if
// the line is not an assistant message or has no usage data.
func messageUsage(line *RawLine) *Usage {
	if line.Type != "message" {
		return nil
	}
	if line.Message == nil {
		return nil
	}
	if line.Message.Role != "assistant" {
		return nil
	}
	return line.Message.Usage
}

// totalTokens returns the sum of all token fields in an AggregatedUsage.
func (a AggregatedUsage) totalTokens() int {
	return a.InputTokens + a.OutputTokens + a.CacheRead + a.CacheWrite
}
