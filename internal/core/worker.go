package core

import (
	"context"
	"fmt"

	gprolog "github.com/Ploos-AS/Golog/internal/prolog"
)

// Worker serializes rule-engine access through a single goroutine.
type Worker struct {
	engine gprolog.Engine
	in     chan Event
	out    chan Action
	err    chan error
}

func NewWorker(engine gprolog.Engine, queueSize int) *Worker {
	if queueSize < 1 {
		queueSize = 1
	}
	return &Worker{
		engine: engine,
		in:     make(chan Event, queueSize),
		out:    make(chan Action, queueSize),
		err:    make(chan error, queueSize),
	}
}

func (w *Worker) Events() chan<- Event { return w.in }
func (w *Worker) Actions() <-chan Action { return w.out }
func (w *Worker) Errors() <-chan error { return w.err }

func (w *Worker) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case ev := <-w.in:
			if err := w.handle(ctx, ev); err != nil {
				select {
				case w.err <- err:
				case <-ctx.Done():
					return
				}
			}
		}
	}
}

func (w *Worker) handle(ctx context.Context, ev Event) error {
	if ev.Type != EventPrivmsg {
		return nil
	}

	reply, ok, err := w.engine.QueryReply(`on_privmsg(?, ?, ?, Reply).`, ev.Nick, ev.Target, ev.Text)
	if err != nil {
		return fmt.Errorf("prolog on_privmsg: %w", err)
	}
	if !ok || reply == "" {
		return nil
	}

	action := Action{Command: "PRIVMSG", Target: ev.Target, Text: reply}
	select {
	case w.out <- action:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
