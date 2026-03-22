package gitstatus

import "testing"

func TestClampDiffScroll_FullFileMode(t *testing.T) {
	tests := []struct {
		name       string
		totalLines int
		height     int
		scroll     int
		wantScroll int
	}{
		{"no clamping needed", 100, 40, 10, 10},
		{"clamp to max", 100, 40, 200, 64},     // maxScroll = 100 - (40-4) = 64
		{"content shorter than viewport", 10, 40, 5, 0}, // maxScroll = 0
		{"negative scroll clamped", 100, 40, -5, 0},
		{"exact fit", 36, 40, 0, 0}, // maxScroll = 36 - 36 = 0
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lines := make([]FullFileLine, tt.totalLines)
			p := &Plugin{
				diffViewMode: DiffViewFullFile,
				fullFileDiff: &FullFileDiff{Lines: lines},
				height:       tt.height,
				diffScroll:   tt.scroll,
			}
			p.clampDiffScroll()
			if p.diffScroll != tt.wantScroll {
				t.Errorf("clampDiffScroll: scroll = %d, want %d", p.diffScroll, tt.wantScroll)
			}
		})
	}
}

func TestClampDiffScroll_ParsedMode(t *testing.T) {
	p := &Plugin{
		diffViewMode: DiffViewUnified,
		diffContent:  "line1\nline2\nline3\nline4\nline5\n",
		height:       10, // maxScroll = 5 - (10-4) = -1 → 0
		diffScroll:   3,
	}
	p.clampDiffScroll()
	if p.diffScroll != 0 {
		t.Errorf("clampDiffScroll parsed mode: scroll = %d, want 0", p.diffScroll)
	}
}

func TestClampDiffPaneScroll_FullFileMode(t *testing.T) {
	tests := []struct {
		name       string
		totalLines int
		height     int
		scroll     int
		wantScroll int
	}{
		{"no clamping needed", 100, 40, 10, 10},
		{"clamp to max", 100, 40, 200, 64},
		{"content shorter than viewport", 10, 40, 5, 0},
		{"negative scroll clamped", 100, 40, -5, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lines := make([]FullFileLine, tt.totalLines)
			p := &Plugin{
				diffPaneViewMode:     DiffViewFullFile,
				diffPaneFullFileDiff: &FullFileDiff{Lines: lines},
				height:               tt.height,
				diffPaneScroll:       tt.scroll,
			}
			p.clampDiffPaneScroll()
			if p.diffPaneScroll != tt.wantScroll {
				t.Errorf("clampDiffPaneScroll: scroll = %d, want %d", p.diffPaneScroll, tt.wantScroll)
			}
		})
	}
}

func TestClampDiffPaneScroll_ParsedDiffMode(t *testing.T) {
	parsed := &ParsedDiff{
		Hunks: []Hunk{
			{Lines: make([]DiffLine, 10)}, // 10 lines + 1 header = 11
			{Lines: make([]DiffLine, 5)},  // 5 lines + 1 header = 6
		},
	}
	// Total parsed lines = 17
	p := &Plugin{
		diffPaneViewMode:   DiffViewUnified,
		diffPaneParsedDiff: parsed,
		height:             20, // maxScroll = 17 - (20-4) = 1
		diffPaneScroll:     5,
	}
	p.clampDiffPaneScroll()
	if p.diffPaneScroll != 1 {
		t.Errorf("clampDiffPaneScroll parsed: scroll = %d, want 1", p.diffPaneScroll)
	}
}

func TestCommitBodyHeight(t *testing.T) {
	tests := []struct {
		name   string
		height int
		want   int
	}{
		{"normal height", 40, 40 - 2 - commitPreviewHeaderLines + 1},  // 31
		{"small height clamps visibleHeight", 6, 3}, // visibleHeight=6-2=4, h=4-8+1=-3 → min 3
		{"very small height", 4, 3}, // visibleHeight clamps to 6, 6-8+1=-1 → min 3
		{"minimum body", 9, 3},      // visibleHeight=7, 7-8+1=0 → min 3
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Plugin{height: tt.height}
			got := p.commitBodyHeight()
			if got != tt.want {
				t.Errorf("commitBodyHeight() = %d, want %d (height=%d)", got, tt.want, tt.height)
			}
		})
	}
}

func TestCommitPreviewHeaderLines_Constant(t *testing.T) {
	// Verify the constant matches the documented layout:
	// 1 (initial currentY) + 2 (hash + blank) + 1 (author) + 2 (date + blank) + 1 (subject) + 1 (blank before body)
	expected := 1 + 2 + 1 + 2 + 1 + 1
	if commitPreviewHeaderLines != expected {
		t.Errorf("commitPreviewHeaderLines = %d, want %d (layout sum)", commitPreviewHeaderLines, expected)
	}
}
