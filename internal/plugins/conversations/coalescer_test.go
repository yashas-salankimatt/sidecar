package conversations

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestEventCoalescer_SingleEvent(t *testing.T) {
	// Single event should be flushed after window
	var received CoalescedRefreshMsg
	var wg sync.WaitGroup
	wg.Add(1)

	ch := make(chan CoalescedRefreshMsg, 1)
	go func() {
		received = <-ch
		wg.Done()
	}()

	c := NewEventCoalescer(50*time.Millisecond, ch)
	c.Add("session-123")

	wg.Wait()

	if len(received.SessionIDs) != 1 {
		t.Errorf("expected 1 session ID, got %d", len(received.SessionIDs))
	}
	if received.SessionIDs[0] != "session-123" {
		t.Errorf("expected session-123, got %s", received.SessionIDs[0])
	}
	if received.RefreshAll {
		t.Error("expected RefreshAll=false")
	}
}

func TestEventCoalescer_MultipleEvents(t *testing.T) {
	// Multiple events within window should be batched
	var received CoalescedRefreshMsg
	var wg sync.WaitGroup
	wg.Add(1)

	ch := make(chan CoalescedRefreshMsg, 1)
	go func() {
		received = <-ch
		wg.Done()
	}()

	c := NewEventCoalescer(100*time.Millisecond, ch)
	c.Add("session-1")
	c.Add("session-2")
	c.Add("session-3")

	wg.Wait()

	if len(received.SessionIDs) != 3 {
		t.Errorf("expected 3 session IDs, got %d", len(received.SessionIDs))
	}
}

func TestEventCoalescer_DuplicateSessionIDs(t *testing.T) {
	// Duplicate session IDs should be deduplicated
	var received CoalescedRefreshMsg
	var wg sync.WaitGroup
	wg.Add(1)

	ch := make(chan CoalescedRefreshMsg, 1)
	go func() {
		received = <-ch
		wg.Done()
	}()

	c := NewEventCoalescer(50*time.Millisecond, ch)
	c.Add("session-1")
	c.Add("session-1")
	c.Add("session-1")

	wg.Wait()

	if len(received.SessionIDs) != 1 {
		t.Errorf("expected 1 session ID (deduplicated), got %d", len(received.SessionIDs))
	}
}

func TestEventCoalescer_EmptySessionID(t *testing.T) {
	// Empty session ID should trigger RefreshAll
	var received CoalescedRefreshMsg
	var wg sync.WaitGroup
	wg.Add(1)

	ch := make(chan CoalescedRefreshMsg, 1)
	go func() {
		received = <-ch
		wg.Done()
	}()

	c := NewEventCoalescer(50*time.Millisecond, ch)
	c.Add("")

	wg.Wait()

	if !received.RefreshAll {
		t.Error("expected RefreshAll=true for empty session ID")
	}
}

func TestEventCoalescer_TooManyEvents(t *testing.T) {
	// More than maxPendingSessionIDs should trigger RefreshAll
	var received CoalescedRefreshMsg
	var wg sync.WaitGroup
	wg.Add(1)

	ch := make(chan CoalescedRefreshMsg, 1)
	go func() {
		received = <-ch
		wg.Done()
	}()

	c := NewEventCoalescer(50*time.Millisecond, ch)
	for i := 0; i < 15; i++ { // More than maxPendingSessionIDs (10)
		c.Add(fmt.Sprintf("session-%d", i))
	}

	wg.Wait()

	if !received.RefreshAll {
		t.Error("expected RefreshAll=true when exceeding max pending IDs")
	}
}

func TestEventCoalescer_Stop(t *testing.T) {
	// Stop should cancel pending flush
	ch := make(chan CoalescedRefreshMsg, 1)
	c := NewEventCoalescer(100*time.Millisecond, ch)

	c.Add("session-1")
	c.Stop()

	// Wait longer than coalesce window
	time.Sleep(150 * time.Millisecond)

	select {
	case <-ch:
		t.Error("expected no message after Stop()")
	default:
		// Good - no message sent
	}
}

func TestEventCoalescer_TimerReset(t *testing.T) {
	// New events should reset the timer
	var received CoalescedRefreshMsg
	var wg sync.WaitGroup
	wg.Add(1)

	start := time.Now()
	ch := make(chan CoalescedRefreshMsg, 1)
	go func() {
		received = <-ch
		wg.Done()
	}()

	c := NewEventCoalescer(50*time.Millisecond, ch)

	// Add event, wait 30ms, add another
	c.Add("session-1")
	time.Sleep(30 * time.Millisecond)
	c.Add("session-2")

	wg.Wait()
	elapsed := time.Since(start)

	// Should take ~80ms (30ms + 50ms), not ~50ms
	if elapsed < 70*time.Millisecond {
		t.Errorf("expected timer to reset, but elapsed was %v", elapsed)
	}

	if len(received.SessionIDs) != 2 {
		t.Errorf("expected 2 session IDs, got %d", len(received.SessionIDs))
	}
}
