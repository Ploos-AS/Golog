package core

import (
	"context"
	"testing"
	"time"
)

func TestSchedulerAfter(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	events := make(chan Event, 2)
	s := NewScheduler(events)
	s.After(ctx, "once", 10*time.Millisecond)

	select {
	case ev := <-events:
		if ev.Type != EventTimer || ev.Name != "once" || ev.Time.IsZero() {
			t.Fatalf("unexpected event: %#v", ev)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for delayed timer")
	}
}

func TestSchedulerEvery(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	events := make(chan Event, 4)
	s := NewScheduler(events)
	s.Every(ctx, "tick", 10*time.Millisecond)

	for i := 0; i < 2; i++ {
		select {
		case ev := <-events:
			if ev.Type != EventTimer || ev.Name != "tick" {
				t.Fatalf("unexpected event: %#v", ev)
			}
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for periodic timer")
		}
	}
}
