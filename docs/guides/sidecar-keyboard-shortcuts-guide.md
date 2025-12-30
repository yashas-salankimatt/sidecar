# Sidecar Keyboard Shortcuts Guide

How to implement keyboard shortcuts for plugins.

## Quick Start: Adding a New Shortcut

**Three things must match for a shortcut to work:**

1. **Command ID** in `Commands()` → e.g., `"stage-file"`
2. **Binding command** in `bindings.go` → e.g., `Command: "stage-file"`
3. **Context** in both places → e.g., `"git-status"`

```go
// 1. In your plugin's Commands() method:
func (p *Plugin) Commands() []plugin.Command {
    return []plugin.Command{
        {ID: "stage-file", Name: "Stage", Context: "git-status"},
    }
}

// 2. In your plugin's FocusContext() method:
func (p *Plugin) FocusContext() string {
    return "git-status"  // Must match the Context above
}

// 3. In internal/keymap/bindings.go:
{Key: "s", Command: "stage-file", Context: "git-status"},
```

## Architecture: Bindings vs Handlers

The keymap system has two distinct purposes:

### 1. Footer Hints (Display Only)
Most bindings exist to show hints in the footer bar. These bindings:
- Map a key to a command ID
- Have a matching `Command` in the plugin's `Commands()` method
- **Do NOT have a registered handler** - the key falls through to the plugin's `Update()` method

### 2. App-Level Handlers (Intercepted Keys)
Some commands have registered handlers that intercept keys before they reach plugins:
- `quit` (ctrl+c) - Shows quit confirmation
- `next-plugin` / `prev-plugin` (tab/shift+tab) - Plugin switching
- `toggle-palette` (?) - Command palette
- `toggle-diagnostics` (!) - Diagnostics overlay
- `refresh` (r) - Global refresh

### Key Routing Flow

```
User presses key
       │
       ▼
┌─────────────────────────────────────────────┐
│ App handleKeyMsg()                          │
│  1. Check quit confirm modal                │
│  2. Check text input contexts (git-commit)  │
│  3. Check 'q' in root contexts → quit       │
│  4. Check app-level shortcuts (tab, ?, !)   │
│  5. keymap.Handle() - registered handlers   │
└─────────────────────────────────────────────┘
       │ If not handled
       ▼
┌─────────────────────────────────────────────┐
│ Plugin Update()                             │
│  - Receives tea.KeyMsg                      │
│  - Routes to view-specific handler          │
│  - Handles the key directly (j/k/G/etc)     │
└─────────────────────────────────────────────┘
```

### Why This Design?

1. **Separation of concerns**: App handles global behavior, plugins handle domain logic
2. **Footer consistency**: bindings.go is the single source of truth for what keys do
3. **Flexibility**: Plugins can handle keys directly without registering handlers

## Adding Shortcuts to the Footer Bar

The footer bar automatically shows shortcuts for the current context. To add your shortcut:

### Step 1: Add to Commands()

```go
func (p *Plugin) Commands() []plugin.Command {
    return []plugin.Command{
        // Context MUST match what FocusContext() returns
        {ID: "stage-file", Name: "Stage", Context: "git-status"},
        {ID: "show-diff", Name: "Diff", Context: "git-status"},

        // Different context for different view
        {ID: "close-diff", Name: "Close", Context: "git-diff"},
    }
}
```

### Step 2: Add to bindings.go

```go
// The Key here shows in the footer as the shortcut
{Key: "s", Command: "stage-file", Context: "git-status"},
{Key: "d", Command: "show-diff", Context: "git-status"},
{Key: "esc", Command: "close-diff", Context: "git-diff"},
```

### Step 3: Handle in Update()

```go
func (p *Plugin) Update(msg tea.Msg) (plugin.Plugin, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.String() {
        case "s":
            return p.stageFile()
        case "d":
            return p.showDiff()
        }
    }
    return p, nil
}
```

### Footer Width Considerations

The footer auto-truncates hints that exceed available width. To maximize visibility:
- Keep command names short: "Stage" not "Stage file"
- Prioritize most-used commands first in `Commands()`
- Test at different terminal widths

## FocusContext Reference

Each plugin returns a context string that determines which bindings are active.

### Global (App-Level)
| Context | Description |
|---------|-------------|
| `global` | Default when no plugin-specific context |
| `""` | Empty string treated as global |

### Conversations Plugin
| Context | View | Description |
|---------|------|-------------|
| `conversations` | Session list (single-pane) | Root view, 'q' quits |
| `conversations-sidebar` | Session list (two-pane) | Left pane focused, 'q' quits |
| `conversations-main` | Messages (two-pane) | Right pane focused |
| `conversations-search` | Search mode | Text input active |
| `conversations-filter` | Filter mode | Adapter filter active |
| `conversation-detail` | Turn list | Viewing turns for a session |
| `message-detail` | Single turn | Viewing one turn's content |
| `analytics` | Analytics view | Usage stats |

### Git Status Plugin
| Context | View | Description |
|---------|------|-------------|
| `git-status` | File list | Root view, 'q' quits |
| `git-status-commits` | Recent commits | Commit list in sidebar |
| `git-status-diff` | Inline diff | Diff pane focused |
| `git-commit-preview` | Commit preview | Commit detail in right pane |
| `git-diff` | Full diff | Full-screen diff view |
| `git-commit` | Commit editor | Text input active |
| `git-history` | History view | Commit history |
| `git-commit-detail` | Commit detail | Single commit view |

### File Browser Plugin
| Context | View | Description |
|---------|------|-------------|
| `file-browser-tree` | Tree view | Root view, 'q' quits |
| `file-browser-preview` | Preview pane | File preview focused |
| `file-browser-search` | Search mode | Filename search |
| `file-browser-content-search` | Content search | File content search |
| `file-browser-quick-open` | Quick open | Fuzzy file finder |

### TD Monitor Plugin
| Context | View | Description |
|---------|------|-------------|
| `td-monitor` | Issue list | Root view, 'q' quits |
| `td-detail` | Issue detail | Single issue view |

## Root Contexts ('q' to Quit)

In root contexts, pressing 'q' shows the quit confirmation. In non-root contexts, 'q' navigates back.

**Root contexts** (q = quit):
- `global`, `""`
- `conversations`, `conversations-sidebar`
- `git-status`, `git-status-commits`, `git-status-diff`
- `file-browser-tree`
- `td-monitor`

**Non-root contexts** (q = back):
- `conversation-detail`, `message-detail`, `analytics`
- `git-diff`, `git-commit`, `git-history`, etc.
- `file-browser-preview`, etc.
- `td-detail`

## Complete Example: Adding "edit" to File Browser

### Step 1: Add to Commands()

```go
// internal/plugins/filebrowser/plugin.go
func (p *Plugin) Commands() []plugin.Command {
    return []plugin.Command{
        {ID: "refresh", Name: "Refresh", Context: "file-browser-tree"},
        {ID: "expand", Name: "Open", Context: "file-browser-tree"},
        {ID: "edit", Name: "Edit", Context: "file-browser-tree"},  // NEW
    }
}
```

### Step 2: Add to bindings.go

```go
// internal/keymap/bindings.go
{Key: "e", Command: "edit", Context: "file-browser-tree"},
```

### Step 3: Handle in Update()

```go
// internal/plugins/filebrowser/plugin.go
func (p *Plugin) Update(msg tea.Msg) (plugin.Plugin, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        if msg.String() == "e" {
            return p, p.openInEditor()
        }
    }
    return p, nil
}
```

## Multiple Contexts (View Modes)

When your plugin has different modes, use different contexts:

```go
func (p *Plugin) FocusContext() string {
    switch p.viewMode {
    case ViewDiff:
        return "git-diff"      // Different bindings active
    case ViewCommit:
        return "git-commit"    // Different bindings active
    default:
        return "git-status"    // Default bindings
    }
}

func (p *Plugin) Commands() []plugin.Command {
    return []plugin.Command{
        // Main view commands
        {ID: "stage-file", Name: "Stage", Context: "git-status"},
        {ID: "show-diff", Name: "Diff", Context: "git-status"},

        // Diff view commands
        {ID: "close-diff", Name: "Close", Context: "git-diff"},
        {ID: "scroll", Name: "Scroll", Context: "git-diff"},

        // Commit view commands
        {ID: "cancel", Name: "Cancel", Context: "git-commit"},
        {ID: "execute-commit", Name: "Commit", Context: "git-commit"},
    }
}
```

## Core Files

| File | Purpose |
|------|---------|
| `internal/plugin/plugin.go` | `Command` struct, `Commands()`, `FocusContext()` interface |
| `internal/keymap/bindings.go` | Default key→command mappings |
| `internal/keymap/registry.go` | Runtime binding lookup, handler registration |
| `internal/app/update.go` | Key routing, `isRootContext()` |
| `internal/app/view.go` | Footer rendering from `Commands()` |

## Common Mistakes

| Symptom | Cause | Fix |
|---------|-------|-----|
| Shortcut doesn't work | Command ID mismatch | Ensure ID in `Commands()` matches `Command` in `bindings.go` |
| Shortcut doesn't work | Context mismatch | Ensure `FocusContext()` returns same context as binding |
| Double footer | Plugin renders own footer | Remove footer rendering from plugin's `View()` |
| Wrong hints shown | `FocusContext()` not updated | Return correct context for current view mode |
| Footer too long | Command names too verbose | Use 1-word names: "Stage" not "Stage file" |
| 'q' quits unexpectedly | Context is root | Add context to non-root list in `isRootContext()` |
| 'q' doesn't quit | Context not root | Add context to root list in `isRootContext()` |

## Checklist for New Shortcuts

- [ ] Command added to `Commands()` with ID, Name, Context
- [ ] `FocusContext()` returns matching context for current view
- [ ] Binding added to `bindings.go` with Key, Command, Context
- [ ] Key handled in `Update()` via `tea.KeyMsg`
- [ ] No duplicate/conflicting keys in same context
- [ ] Command name is short (1-2 words max)
- [ ] Plugin does NOT render its own footer
- [ ] 'q' behavior is correct (check `isRootContext()`)

## Key Format Reference

```go
// Letters (lowercase)
{Key: "j", Command: "cursor-down", Context: "global"}

// Shifted letters (uppercase)
{Key: "G", Command: "cursor-bottom", Context: "global"}

// Control combos
{Key: "ctrl+d", Command: "page-down", Context: "global"}
{Key: "ctrl+c", Command: "quit", Context: "global"}

// Alt combos
{Key: "alt+enter", Command: "execute-commit", Context: "git-commit"}

// Special keys
{Key: "enter", Command: "select", Context: "global"}
{Key: "esc", Command: "back", Context: "global"}
{Key: "tab", Command: "next-plugin", Context: "global"}
{Key: "shift+tab", Command: "prev-plugin", Context: "global"}
{Key: "up", Command: "cursor-up", Context: "global"}
{Key: "down", Command: "cursor-down", Context: "global"}

// Sequences (space-separated, 500ms timeout)
{Key: "g g", Command: "cursor-top", Context: "global"}
```

## Context Precedence

1. Plugin-specific context checked first (e.g., `git-status`)
2. Falls back to `global` context if no match

This means `c` in `git-status` context triggers `commit`, but `c` in `global` context triggers `focus-conversations`.

## Testing

1. Run `sidecar --debug` to see key handling logs
2. Press `?` to verify help overlay shows your bindings
3. Check footer shows your command names
4. Test that keys trigger correct actions
5. Test context switches (enter subview, verify new bindings active)
6. Test 'q' behavior in each context (quit vs back)
