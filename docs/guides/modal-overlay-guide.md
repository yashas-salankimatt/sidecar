# Modal Overlay Implementation Guide

This guide covers how to implement modals with dimmed backgrounds in Sidecar.

## Overview

Modals should dim the background to:
- Draw user focus to the modal content
- Provide visual separation between modal and underlying content
- Create a consistent, polished UX across the application

## Two Approaches

### 1. App-Level Modals (Full-Screen Control)

For modals rendered at the app level (`internal/app/view.go`), use `lipgloss.Place()` with whitespace options:

```go
func (m Model) renderMyModal(content string) string {
    modal := styles.ModalBox.Render(content)

    return lipgloss.Place(
        m.width, m.height,
        lipgloss.Center, lipgloss.Center,
        modal,
        lipgloss.WithWhitespaceChars(" "),
        lipgloss.WithWhitespaceForeground(lipgloss.Color("#000000")),
    )
}
```

**How it works:**
- `lipgloss.Place()` centers the modal and fills surrounding space with whitespace
- `WithWhitespaceChars(" ")` uses spaces as the fill character
- `WithWhitespaceForeground("#000000")` colors those spaces black, creating a dim effect

**Examples:** Help modal, Command palette

### 2. Plugin-Level Modals (Within Plugin Bounds)

For modals rendered within plugins, use `ui.OverlayModal()` from `internal/ui/overlay.go`:

```go
// In your plugin's render function:
func (p *Plugin) renderMyModal() string {
    // Render what should appear behind the modal
    background := p.renderNormalView()

    // Render your modal content with border
    modalContent := lipgloss.NewStyle().
        Border(lipgloss.RoundedBorder()).
        BorderForeground(styles.Primary).
        Padding(1, 2).
        Width(modalWidth).
        Render(content)

    // Overlay modal on dimmed background
    return ui.OverlayModal(background, modalContent, p.width, p.height)
}
```

**How `ui.OverlayModal()` works:**
1. Calculates modal position (centered horizontally and vertically)
2. Strips ANSI codes from background and applies dim gray styling (color 242)
3. Composites each row: `dimmed-left + modal + dimmed-right`
4. Shows dimmed background on all four sides of the modal

Note: Background colors are not preserved because ANSI SGR 2 (faint) doesn't reliably combine with existing color codes in most terminals. The gray overlay provides consistent dimming.

**Visual result:**
```
╔════════════════════════════════════════════════╗
║  [dimmed gray background text]                 ║
║  [gray left]  ┌─Modal─┐  [gray right]          ║
║  [gray left]  │ text  │  [gray right]          ║
║  [gray left]  └───────┘  [gray right]          ║
║  [dimmed gray background text]                 ║
╚════════════════════════════════════════════════╝
```

**Examples:** Git commit modal, Push menu, Branch picker

## Implementation Checklist

When adding dimmed background to a modal:

1. **Identify the modal type:**
   - App-level (full screen) → Use `lipgloss.Place()` with whitespace options
   - Plugin-level (bounded) → Use `ui.OverlayModal()` helper

2. **For app-level modals**, add these options to `lipgloss.Place()`:
   ```go
   lipgloss.WithWhitespaceChars(" "),
   lipgloss.WithWhitespaceForeground(lipgloss.Color("#000000")),
   ```

3. **For plugin-level modals:**
   - Import `github.com/marcus/sidecar/internal/ui`
   - Call `ui.OverlayModal(background, modalContent, width, height)`
   - Pass raw modal content (don't pre-center with `lipgloss.Place()`)

## Style Constants

```go
// Plugin-level dimming (strips ANSI and applies gray)
var DimStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("242"))

// App-level dim color (full black background)
const dimColor = "#000000"
```

**Why not preserve colors?** ANSI SGR 2 (faint) doesn't reliably combine with existing color codes in most terminals. Stripping colors and applying a consistent gray provides reliable dimming across all terminal emulators.

## Common Pitfalls

1. **Don't use `lipgloss.Place()` with `ui.OverlayModal()`** - they both handle centering, which causes layout issues

2. **Pass the full background** - `ui.OverlayModal()` needs the complete background content to composite correctly. Don't pre-truncate or pre-dim.

3. **Height constraints** - Ensure modal content respects available height to prevent overflow

## File Locations

- App-level modals: `internal/app/view.go`
- Plugin modal helper: `internal/ui/overlay.go` (`OverlayModal()`)
- Modal styles: `internal/styles/styles.go` (`ModalBox`, `ModalTitle`, etc.)
