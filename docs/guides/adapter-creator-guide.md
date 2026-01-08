# Adapter Creator Guide

This guide describes how to add a new AI session adapter to Sidecar.

## Overview

Adapters live in `internal/adapter/<name>` and implement the `adapter.Adapter` interface:

```go
// internal/adapter/adapter.go

type Adapter interface {
	ID() string
	Name() string
	Icon() string
	Detect(projectRoot string) (bool, error)
	Capabilities() CapabilitySet
	Sessions(projectRoot string) ([]Session, error)
	Messages(sessionID string) ([]Message, error)
	Usage(sessionID string) (*UsageStats, error)
	Watch(projectRoot string) (<-chan Event, error)
}
```

Sidecar discovers adapters via `adapter.RegisterFactory` and `adapter.DetectAdapters`.

## File Layout

```
internal/adapter/<name>/
  adapter.go       // main implementation
  types.go         // JSONL payload types
  watcher.go       // fsnotify watcher (if supported)
  register.go      // init() registers factory
  *_test.go        // unit tests + fixture parsing
```

## Required Fields

When building sessions, populate adapter identity:

```go
adapter.Session{
	ID:          meta.SessionID,
	Name:        shortID(meta.SessionID),
	Slug:        meta.SessionID, // optional: short display slug if you have one
	AdapterID:   "<your-id>",
	AdapterName: "<Your Name>",
	AdapterIcon: a.Icon(),
	// ... timestamps, tokens, counts
}
```

These are used for badges, filtering, and resume commands in the conversations UI.

## Step-by-step

### 1) Define adapter constants

```go
const (
	adapterID   = "my-adapter"
	adapterName = "My Adapter"
)
```

Pick a stable `adapterID` (it becomes part of persisted UI state like filters).

### 2) Define adapter icon

Choose a unique single-character icon for your adapter:

```go
func (a *Adapter) Icon() string { return "◆" }
```

Icon guidelines:
- Use non-emoji Unicode symbols (◆ ▶ ★ ◇ ▲ ■ etc.)
- Avoid emojis for terminal compatibility
- Pick something visually distinct from existing adapters
- Icon appears in conversation list badges

Existing icons:

| Adapter     | Icon |
|-------------|------|
| claude-code | ◆    |
| codex       | ▶    |
| gemini-cli  | ★    |
| cursor-cli  | ▌    |
| warp        | ⚡   |
| opencode    | ◇    |

### 3) Implement Detect

Detect should return `true` only when sessions for `projectRoot` exist. Prefer:
- `filepath.Abs` + `filepath.Rel` for stable path matching
- `filepath.EvalSymlinks` to avoid false negatives
- graceful handling when data directories do not exist

### 4) Implement Sessions

Parse all session files, extract:
- `SessionID`
- `FirstMsg` and `LastMsg`
- `MsgCount` (user + assistant messages)
- `TotalTokens` (if available)

Sort by `UpdatedAt` descending.

### 5) Implement Messages

Return ordered `adapter.Message` values with:
- `Role`: user or assistant
- `Content`: concatenated content blocks
- `ToolUses`: tool calls and outputs
- `ThinkingBlocks`: reasoning summaries (if present)
- `TokenUsage`: map token_count events to the next assistant message
- `Model`: from your session metadata

### 6) Implement Usage

Aggregate per-message token usage, and optionally fall back to totals from a session summary record.

### 7) Implement Watch (optional but recommended)

Use `fsnotify` and:
- add watchers for nested directories (fsnotify is non-recursive)
- debounce rapid writes
- map file events to `adapter.Event` types

### 8) Register the adapter

Add a `register.go` with an init hook:

```go
package myadapter

import "github.com/marcus/sidecar/internal/adapter"

func init() {
	adapter.RegisterFactory(func() adapter.Adapter {
		return New()
	})
}
```

And ensure the package is imported (blank import) in `cmd/sidecar/main.go`:

```go
import (
	_ "github.com/marcus/sidecar/internal/adapter/myadapter"
)
```

## UI Integration Notes

- Conversations view shows adapter badges using `AdapterIcon` + abbreviation.
- `resumeCommand()` is adapter-specific; add a mapping in `internal/plugins/conversations/view.go` if your tool supports resuming sessions.
- `modelShortName()` should be extended if your models are non-Claude.

## Conversation Flow View (Primary)

The conversations plugin has two view modes:

1. **Conversation Flow** (default) - Content-focused chat thread like Claude Code web UI
2. **Turn View** - Metadata-focused aggregated turns (accessible via `v` shortcut)

Conversation flow is the primary view. To support it well, adapters should populate `ContentBlocks` on messages.

### ContentBlocks Structure

```go
type ContentBlock struct {
    Type       string // "text", "tool_use", "tool_result", "thinking"
    Text       string // For text/thinking blocks
    ToolUseID  string // For tool_use and tool_result linking
    ToolName   string // For tool_use
    ToolInput  string // For tool_use (JSON string)
    ToolOutput string // For tool_result
    IsError    bool   // For tool_result errors
    TokenCount int    // For thinking blocks
}
```

### Populating ContentBlocks

When implementing `Messages()`, parse your source format into ContentBlocks:

```go
func buildContentBlocks(rawBlocks []YourContentBlock) []adapter.ContentBlock {
    var blocks []adapter.ContentBlock
    for _, b := range rawBlocks {
        switch b.Type {
        case "text":
            blocks = append(blocks, adapter.ContentBlock{
                Type: "text",
                Text: b.Text,
            })
        case "tool_use":
            inputJSON, _ := json.Marshal(b.Input)
            blocks = append(blocks, adapter.ContentBlock{
                Type:      "tool_use",
                ToolUseID: b.ID,
                ToolName:  b.Name,
                ToolInput: string(inputJSON),
            })
        case "tool_result":
            blocks = append(blocks, adapter.ContentBlock{
                Type:       "tool_result",
                ToolUseID:  b.ToolUseID,
                ToolOutput: b.Content,
                IsError:    b.IsError,
            })
        case "thinking":
            blocks = append(blocks, adapter.ContentBlock{
                Type:       "thinking",
                Text:       b.Thinking,
                TokenCount: b.TokenCount,
            })
        }
    }
    return blocks
}
```

### Tool Result Linking

The conversation view uses a two-pass approach to link tool calls with their results:

1. **First pass**: Collect all `tool_result` blocks by `ToolUseID`
2. **Second pass**: When rendering `tool_use` blocks, look up and inline the result

This means adapters should:
- Always populate `ToolUseID` on both `tool_use` and `tool_result` blocks
- Use consistent IDs that match between the tool call and its result
- Include `tool_result` blocks even though they're not rendered separately

### Example: Tool Use Flow

```
Assistant message with ContentBlocks:
  [0] type=text, text="I'll read the file..."
  [1] type=tool_use, id="tu_123", name="Read", input={"path": "foo.go"}

User message with ContentBlocks:
  [0] type=tool_result, tool_use_id="tu_123", output="package main..."
```

The conversation view will render this as:
```
[14:30] assistant
    I'll read the file...
    ▶ Read foo.go
      package main...
```

### Fallback Rendering

If `ContentBlocks` is empty, the view falls back to:
1. `Message.Content` string (rendered as markdown)
2. `Message.ToolUses` array (legacy format)
3. `Message.ThinkingBlocks` array (legacy format)

Populating `ContentBlocks` provides the richest display but isn't required.

### View Mode Behavior

| Feature | Conversation Flow | Turn View |
|---------|------------------|-----------|
| Shows actual message content | ✓ | Preview only |
| Inline tool results | ✓ | Count only |
| Expandable thinking | ✓ | Token count |
| Message-level cursor | ✓ | Turn-level |
| Keyboard shortcut | Default | `v` to toggle |

Adapters don't need to do anything special for turn view—it's computed from messages automatically.

## Testing Checklist

- Detect() matches both absolute and relative project roots
- Sessions() includes AdapterIcon from Icon()
- Sessions() sorts by UpdatedAt
- Messages() attaches tool uses and token usage correctly
- Messages() populates ContentBlocks with proper types and linking
- ContentBlocks tool_use and tool_result share matching ToolUseID
- Usage() matches message totals
- Watch() emits create/write events (if supported)
- Conversation flow view renders messages with inline tool results

## Performance Best Practices

The `Sessions()` method is called frequently (on every watch event). Poorly optimized adapters can cause CPU spikes during active AI sessions.

### Cache within Sessions()

Avoid parsing the same data multiple times within a single `Sessions()` call:

```go
// BAD: Parses messages twice per session
func (a *Adapter) Sessions(projectRoot string) ([]adapter.Session, error) {
    for _, entry := range entries {
        msgCount := a.countMessages(path)      // parses all messages
        firstMsg := a.getFirstUserMessage(path) // parses again!
    }
}

// GOOD: Parse once, extract multiple values
func (a *Adapter) Sessions(projectRoot string) ([]adapter.Session, error) {
    for _, entry := range entries {
        messages, _ := a.parseMessages(path)
        msgCount := len(messages)
        firstMsg := extractFirstUserMessage(messages)
    }
}
```

### Pre-compile Regexes

Regex compilation is expensive. For any regex used in rendering or message parsing, compile once at package level:

```go
// BAD: Compiles on every call
func stripTags(s string) string {
    re := regexp.MustCompile(`<[^>]+>`)
    return re.ReplaceAllString(s, "")
}

// GOOD: Compile once at package level
var tagRegex = regexp.MustCompile(`<[^>]+>`)

func stripTags(s string) string {
    return tagRegex.ReplaceAllString(s, "")
}
```

### Watch Event Efficiency

When implementing `Watch()`:
1. Include `SessionID` in emitted events so consumers can do targeted refreshes
2. Use adequate debounce (100-200ms) to coalesce rapid writes
3. Consider file modification time checks to avoid spurious events

```go
// Emit events with SessionID for targeted refresh
events <- adapter.Event{
    Type:      adapter.EventMessageAdded,
    SessionID: sessionID,  // enables smart refresh
}
```

### Avoid Blocking I/O in Hot Paths

The `Messages()` method may be called frequently when a session is selected:
- Consider caching parsed messages with TTL or invalidation on watch events
- Use read-only SQLite mode (`?mode=ro`) to avoid lock contention
- Avoid repeated directory scans; cache session-to-path mappings

## Minimal Skeleton

```go
package myadapter

type Adapter struct {
	// data dir, indexes, etc.
}

func New() *Adapter { /* ... */ }
func (a *Adapter) ID() string { return adapterID }
func (a *Adapter) Name() string { return adapterName }
func (a *Adapter) Icon() string { return "●" } // choose unique icon
func (a *Adapter) Detect(projectRoot string) (bool, error) { /* ... */ }
func (a *Adapter) Capabilities() adapter.CapabilitySet { /* ... */ }
func (a *Adapter) Sessions(projectRoot string) ([]adapter.Session, error) { /* ... */ }
func (a *Adapter) Messages(sessionID string) ([]adapter.Message, error) { /* ... */ }
func (a *Adapter) Usage(sessionID string) (*adapter.UsageStats, error) { /* ... */ }
func (a *Adapter) Watch(projectRoot string) (<-chan adapter.Event, error) { /* ... */ }
```
