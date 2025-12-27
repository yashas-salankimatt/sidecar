package app

import (
	"fmt"
	"math"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// IntroModel handles the intro animation state.
type IntroModel struct {
	Active    bool
	StartTime time.Time
	Letters   []*IntroLetter
	Width     int
	Height    int
	Done      bool // Set to true when animation is finished
}

type IntroLetter struct {
	Char     rune
	TargetX  float64
	CurrentX float64

	// Color interpolation
	StartColor   RGB
	EndColor     RGB
	CurrentColor RGB

	Delay time.Duration
}

type RGB struct {
	R, G, B float64
}

func hexToRGB(hex string) RGB {
	hex = strings.TrimPrefix(hex, "#")
	var r, g, b uint8
	fmt.Sscanf(hex, "%02x%02x%02x", &r, &g, &b)
	return RGB{float64(r), float64(g), float64(b)}
}

func (c RGB) toLipgloss() lipgloss.Color {
	return lipgloss.Color(fmt.Sprintf("#%02x%02x%02x", int(c.R), int(c.G), int(c.B)))
}

func NewIntroModel() IntroModel {
	text := "Sidecar"
	letters := make([]*IntroLetter, len(text))

	// Gradient endpoints for the final state
	// Orange: #F59E0B (245, 158, 11)
	// Dark Yellow: #D97706 (217, 119, 6) - slightly darker/more orange-yellow
	startGradient := hexToRGB("#F59E0B")
	endGradient := hexToRGB("#D97706")

	// Random distinct start colors
	startColors := []string{
		"#EF4444", // Red
		"#3B82F6", // Blue
		"#10B981", // Green
		"#8B5CF6", // Purple
		"#EC4899", // Pink
		"#06B6D4", // Cyan
		"#F97316", // Orange
	}

	for i, char := range text {
		// Calculate target color for this letter in the gradient
		t := float64(i) / float64(len(text)-1)
		targetColor := RGB{
			R: startGradient.R + t*(endGradient.R-startGradient.R),
			G: startGradient.G + t*(endGradient.G-startGradient.G),
			B: startGradient.B + t*(endGradient.B-startGradient.B),
		}

		letters[i] = &IntroLetter{
			Char:         char,
			CurrentX:     -20.0 - float64(i)*5.0, // Start well offscreen left
			StartColor:   hexToRGB(startColors[i%len(startColors)]),
			EndColor:     targetColor,
			CurrentColor: hexToRGB(startColors[i%len(startColors)]),
			Delay:        time.Duration(i) * 100 * time.Millisecond,
		}
	}

	return IntroModel{
		Active:  true,
		Letters: letters,
	}
}

// Update progresses the animation
func (m *IntroModel) Update(dt time.Duration) {
	if !m.Active || m.Done {
		return
	}

	allSettled := true
	
	// Center the text
	// "S i d e c a r" -> 7 chars + 6 spaces = 13 visual width?
	// Or we can just space them out with padding.
	// Let's assume a spacing of 3 visual units between centers.
	totalWidth := float64(len(m.Letters)-1) * 3.0
	startX := (float64(m.Width) - totalWidth) / 2.0

	for i, l := range m.Letters {
		l.TargetX = startX + float64(i)*3.0

		// Check delay
		if m.StartTime.IsZero() {
			m.StartTime = time.Now()
		}
		elapsed := time.Since(m.StartTime)
		if elapsed < l.Delay {
			allSettled = false
			continue
		}

		// Animation logic (Spring-like or EaseOut)
		// Move towards TargetX
		speed := 15.0 // pixels per second
		dist := l.TargetX - l.CurrentX
		
		// Simple ease-out
		move := dist * 5.0 * dt.Seconds()
		if move > 0 && move < speed*dt.Seconds() {
			move = speed * dt.Seconds()
		}
		if move > dist {
			move = dist
		}
		
		l.CurrentX += move

		// Color interpolation
		// Interpolate towards EndColor
		colorSpeed := 2.0 * dt.Seconds()
		l.CurrentColor.R += (l.EndColor.R - l.CurrentColor.R) * colorSpeed
		l.CurrentColor.G += (l.EndColor.G - l.CurrentColor.G) * colorSpeed
		l.CurrentColor.B += (l.EndColor.B - l.CurrentColor.B) * colorSpeed

		if math.Abs(l.TargetX-l.CurrentX) > 0.1 || 
		   math.Abs(l.EndColor.R-l.CurrentColor.R) > 1.0 {
			allSettled = false
		}
	}

	if allSettled {
		// Keep it visible for a moment?
		// Or just mark done.
		// For now, mark done immediately when settled.
		// Maybe add a small pause before switching to main UI?
		// We can handle that in the Model.
		m.Done = true
	}
}

func (m IntroModel) View() string {
	if !m.Active {
		return ""
	}

	// Create a canvas (string builder with spaces)
	// Since we can't easily do absolute positioning in a string without a 2D buffer,
	// we'll approximate by rendering lines.
	// But actually, we just need to render one line with the letters at correct positions?
	// Or maybe centered vertically.

	var b strings.Builder
	
	// Vertical centering
	centerY := m.Height / 2
	for y := 0; y < m.Height; y++ {
		if y == centerY {
			// Render the line with letters
			line := make([]string, m.Width)
			for k := range line {
				line[k] = " "
			}

			for _, l := range m.Letters {
				x := int(math.Round(l.CurrentX))
				if x >= 0 && x < m.Width {
					style := lipgloss.NewStyle().Foreground(l.CurrentColor.toLipgloss()).Bold(true)
					line[x] = style.Render(string(l.Char))
				}
			}
			b.WriteString(strings.Join(line, ""))
		} else {
			b.WriteString(strings.Repeat(" ", m.Width))
		}
		if y < m.Height-1 {
			b.WriteString("\n")
		}
	}

	return b.String()
}

// IntroTickMsg is sent to update the animation frame.
type IntroTickMsg time.Time

func IntroTick() tea.Cmd {
	return tea.Tick(time.Millisecond*16, func(t time.Time) tea.Msg {
		return IntroTickMsg(t)
	})
}
