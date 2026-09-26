package core

import (
	"context"
	"sync"
	"time"
)

// Scheduler feeds timer events into the same serialized worker queue used by IRC.
// It supports periodic timers and one-shot delayed timers.
type Scheduler struct {
	events chan<- Event
	mu     sync.Mutex
	cancel map[string]context.CancelFunc
}

func NewScheduler(events chan<- Event) *Scheduler {
	return &Scheduler{events: events, cancel: make(map[string]context.CancelFunc)}
}

func (s *Scheduler) Every(ctx context.Context, name string, interval time.Duration) {
	if name == "" || interval <= 0 {
		return
	}
	child, cancel := context.WithCancel(ctx)
	s.replace(name, cancel)
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-child.Done():
				return
			case now := <-ticker.C:
				if !s.emit(child, name, now) {
					return
				}
			}
		}
	}()
}

func (s *Scheduler) After(ctx context.Context, name string, delay time.Duration) {
	if name == "" || delay < 0 {
		return
	}
	child, cancel := context.WithCancel(ctx)
	s.replace(name, cancel)
	go func() {
		timer := time.NewTimer(delay)
		defer timer.Stop()
		select {
		case <-child.Done():
			return
		case now := <-timer.C:
			s.emit(child, name, now)
			s.Cancel(name)
		}
	}()
}

func (s *Scheduler) Cancel(name string) {
	s.mu.Lock()
	cancel := s.cancel[name]
	delete(s.cancel, name)
	s.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

func (s *Scheduler) replace(name string, cancel context.CancelFunc) {
	s.mu.Lock()
	old := s.cancel[name]
	s.cancel[name] = cancel
	s.mu.Unlock()
	if old != nil {
		old()
	}
}

func (s *Scheduler) emit(ctx context.Context, name string, now time.Time) bool {
	select {
	case s.events <- Event{Type: EventTimer, Name: name, Time: now}:
		return true
	case <-ctx.Done():
		return false
	}
}
