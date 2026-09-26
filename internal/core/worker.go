package core

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

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
	switch ev.Type {
	case EventPrivmsg:
		return w.handlePrivmsg(ctx, ev)
	case EventTimer:
		return w.handleTimer(ctx, ev)
	case EventJoin:
		return w.fire(`catch(on_join(?, ?, ?, ?), _, fail).`, ev.Nick, ev.Account, ev.Target, eventTime(ev.Time))
	case EventPart:
		return w.fire(`catch(on_part(?, ?, ?, ?), _, fail).`, ev.Nick, ev.Target, ev.Text, eventTime(ev.Time))
	case EventQuit:
		return w.fire(`catch(on_quit(?, ?, ?), _, fail).`, ev.Nick, ev.Text, eventTime(ev.Time))
	case EventNick:
		return w.fire(`catch(on_nick(?, ?, ?), _, fail).`, ev.Nick, ev.Target, eventTime(ev.Time))
	case EventAccount:
		return w.fire(`catch(on_account(?, ?, ?), _, fail).`, ev.Nick, ev.Account, eventTime(ev.Time))
	default:
		return nil
	}
}

func (w *Worker) handlePrivmsg(ctx context.Context, ev Event) error {
	timestamp := eventTime(ev.Time)
	tags := flattenTags(ev.Tags)

	actions, err := w.engine.QueryActions(
		`catch(on_privmsg_action(?, ?, ?, ?, ?, ?, Command, Target, Arg, Text), _, fail).`,
		ev.Nick, ev.Account, ev.Target, ev.Text, timestamp, tags,
	)
	if err != nil {
		return fmt.Errorf("prolog on_privmsg_action/10: %w", err)
	}
	if len(actions) > 0 {
		for _, a := range actions {
			if err := w.emit(ctx, Action{Command: strings.ToUpper(a.Command), Target: a.Target, Arg: a.Arg, Text: a.Text}); err != nil {
				return err
			}
		}
		return nil
	}

	reply, ok, err := w.engine.QueryReply(
		`catch(on_privmsg(?, ?, ?, ?, ?, ?, Reply), _, fail).`,
		ev.Nick, ev.Account, ev.Target, ev.Text, timestamp, tags,
	)
	if err != nil {
		return fmt.Errorf("prolog on_privmsg/7: %w", err)
	}
	if !ok {
		reply, ok, err = w.engine.QueryReply(`catch(on_privmsg(?, ?, ?, Reply), _, fail).`, ev.Nick, ev.Target, ev.Text)
		if err != nil {
			return fmt.Errorf("prolog on_privmsg/4: %w", err)
		}
	}
	if !ok || reply == "" {
		return nil
	}
	return w.emit(ctx, Action{Command: "PRIVMSG", Target: ev.Target, Text: reply})
}

func (w *Worker) handleTimer(ctx context.Context, ev Event) error {
	actions, err := w.engine.QueryActions(
		`catch(on_timer(?, ?, Command, Target, Arg, Text), _, fail).`,
		ev.Name, eventTime(ev.Time),
	)
	if err != nil {
		return fmt.Errorf("prolog on_timer/6: %w", err)
	}
	for _, a := range actions {
		if err := w.emit(ctx, Action{Command: strings.ToUpper(a.Command), Target: a.Target, Arg: a.Arg, Text: a.Text}); err != nil {
			return err
		}
	}
	return nil
}

func (w *Worker) emit(ctx context.Context, action Action) error {
	select {
	case w.out <- action:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (w *Worker) fire(query string, args ...any) error {
	_, err := w.engine.Ask(query, args...)
	if err != nil {
		return fmt.Errorf("prolog event hook: %w", err)
	}
	return nil
}

func eventTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339Nano)
}

func flattenTags(tags map[string]string) string {
	if len(tags) == 0 {
		return ""
	}
	keys := make([]string, 0, len(tags))
	for k := range tags {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, k+"="+tags[k])
	}
	return strings.Join(parts, ";")
}
