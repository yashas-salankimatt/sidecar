package conversations

import (
	"sync"
	"time"
)

const (
	defaultCoalesceWindow = 250 * time.Millisecond
	maxPendingSessionIDs  = 10 // Above this, trigger full refresh
)

// EventCoalescer batches rapid watch events into single refreshes.
// When events arrive faster than the coalesce window, they are
// accumulated and a single refresh is triggered after the window closes.
type EventCoalescer struct {
	mu             sync.Mutex
	pendingIDs     map[string]struct{} // SessionIDs to refresh
	refreshAll     bool                // true if we need full refresh (empty ID received)
	timer          *time.Timer
	coalesceWindow time.Duration
	msgChan        chan<- CoalescedRefreshMsg // channel to send messages
}

// NewEventCoalescer creates a coalescer with the given window duration.
// msgChan receives CoalescedRefreshMsg when the coalesce window closes.
func NewEventCoalescer(window time.Duration, msgChan chan<- CoalescedRefreshMsg) *EventCoalescer {
	if window == 0 {
		window = defaultCoalesceWindow
	}
	return &EventCoalescer{
		pendingIDs:     make(map[string]struct{}),
		coalesceWindow: window,
		msgChan:        msgChan,
	}
}

// Add queues a sessionID for refresh. Empty string triggers full refresh.
// Resets the coalesce timer on each call.
func (c *EventCoalescer) Add(sessionID string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if sessionID == "" {
		c.refreshAll = true
	} else {
		c.pendingIDs[sessionID] = struct{}{}
	}

	// Reset timer - we wait for a quiet period
	if c.timer != nil {
		c.timer.Stop()
	}
	c.timer = time.AfterFunc(c.coalesceWindow, c.flush)
}

// flush sends the coalesced refresh message and resets state.
// Called by timer when coalesce window closes.
func (c *EventCoalescer) flush() {
	c.mu.Lock()

	// Collect pending IDs
	sessionIDs := make([]string, 0, len(c.pendingIDs))
	for id := range c.pendingIDs {
		sessionIDs = append(sessionIDs, id)
	}

	refreshAll := c.refreshAll || len(sessionIDs) > maxPendingSessionIDs

	// Reset state
	c.pendingIDs = make(map[string]struct{})
	c.refreshAll = false
	c.timer = nil

	c.mu.Unlock()

	// Send message outside lock
	if c.msgChan != nil {
		select {
		case c.msgChan <- CoalescedRefreshMsg{
			SessionIDs: sessionIDs,
			RefreshAll: refreshAll,
		}:
		default:
			// Channel full, drop message (next event will trigger refresh)
		}
	}
}

// Stop cancels any pending flush. Call when plugin is shutting down.
func (c *EventCoalescer) Stop() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.timer != nil {
		c.timer.Stop()
		c.timer = nil
	}
}

// CoalescedRefreshMsg is sent when the coalesce window closes.
type CoalescedRefreshMsg struct {
	SessionIDs []string // Specific sessions to refresh (if not RefreshAll)
	RefreshAll bool     // If true, ignore SessionIDs and do full refresh
}
