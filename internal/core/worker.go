package core

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	gprolog "github.com/Ploos-AS/Golog/internal/prolog"
	gstate "github.com/Ploos-AS/Golog/internal/state"
)

type reloadRequest struct {
	engine gprolog.Engine
	done   chan error
}

type EngineLoader func() (gprolog.Engine, error)

// Worker serializes rule-engine access through a single goroutine.
type Worker struct {
	engine        gprolog.Engine
	state         gstate.Store
	in            chan Event
	out           chan Action
	err           chan error
	reload        chan reloadRequest
	adminAccounts map[string]struct{}
	engineLoader  EngineLoader
}

func NewWorker(engine gprolog.Engine, queueSize int) *Worker {
	if queueSize < 1 {
		queueSize = 1
	}
	return &Worker{
		engine:        engine,
		in:            make(chan Event, queueSize),
		out:           make(chan Action, queueSize),
		err:           make(chan error, queueSize),
		reload:        make(chan reloadRequest),
		adminAccounts: map[string]struct{}{},
	}
}

// SetStateStore enables persistent state for state-aware Prolog hooks.
func (w *Worker) SetStateStore(store gstate.Store) { w.state = store }

// SetAdminAccounts enables account-authenticated operator commands. IRC account
// names are matched case-insensitively; nicknames alone never grant access.
func (w *Worker) SetAdminAccounts(accounts []string) {
	w.adminAccounts = make(map[string]struct{}, len(accounts))
	for _, account := range accounts {
		account = strings.ToLower(strings.TrimSpace(account))
		if account != "" {
			w.adminAccounts[account] = struct{}{}
		}
	}
}

// SetEngineLoader supplies the validated engine factory used by !admin reload.
func (w *Worker) SetEngineLoader(loader EngineLoader) { w.engineLoader = loader }

func (w *Worker) Events() chan<- Event { return w.in }
func (w *Worker) Actions() <-chan Action { return w.out }
func (w *Worker) Errors() <-chan error { return w.err }

// Reload swaps in an already validated Prolog engine. The swap is serialized
// with normal event processing so no query can observe a half-reloaded engine.
func (w *Worker) Reload(ctx context.Context, engine gprolog.Engine) error {
	if engine == nil {
		return fmt.Errorf("reload engine must not be nil")
	}
	req := reloadRequest{engine: engine, done: make(chan error, 1)}
	select {
	case w.reload <- req:
	case <-ctx.Done():
		return ctx.Err()
	}
	select {
	case err := <-req.done:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (w *Worker) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case req := <-w.reload:
			w.engine = req.engine
			req.done <- nil
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
	if handled, err := w.handleAdmin(ctx, ev); handled || err != nil {
		return err
	}

	timestamp := eventTime(ev.Time)
	tags := flattenTags(ev.Tags)

	if w.state != nil {
		actions, err := w.engine.QueryActions(
			`catch(on_privmsg_state(?, ?, ?, ?, ?, ?, ?, Command, Target, Arg, Text), _, fail).`,
			ev.Nick, ev.Account, ev.Target, ev.Text, timestamp, tags, gstate.Flatten(w.state.Snapshot()),
		)
		if err != nil {
			return fmt.Errorf("prolog on_privmsg_state/11: %w", err)
		}
		if len(actions) > 0 {
			return w.applyRuleActions(ctx, actions)
		}
	}

	actions, err := w.engine.QueryActions(
		`catch(on_privmsg_action(?, ?, ?, ?, ?, ?, Command, Target, Arg, Text), _, fail).`,
		ev.Nick, ev.Account, ev.Target, ev.Text, timestamp, tags,
	)
	if err != nil {
		return fmt.Errorf("prolog on_privmsg_action/10: %w", err)
	}
	if len(actions) > 0 {
		return w.applyRuleActions(ctx, actions)
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

func (w *Worker) handleAdmin(ctx context.Context, ev Event) (bool, error) {
	fields := strings.Fields(ev.Text)
	if len(fields) == 0 || !strings.EqualFold(fields[0], "!admin") {
		return false, nil
	}
	if _, ok := w.adminAccounts[strings.ToLower(ev.Account)]; !ok || ev.Account == "" {
		return true, w.emit(ctx, Action{Command: "NOTICE", Target: ev.Nick, Text: "admin access denied: authenticated IRC account required"})
	}
	if len(fields) == 1 || strings.EqualFold(fields[1], "help") {
		return true, w.emit(ctx, Action{Command: "NOTICE", Target: ev.Nick, Text: "admin: status | reload | join <#channel> | part <#channel> [reason] | state [key]"})
	}

	switch strings.ToLower(fields[1]) {
	case "status":
		stateCount := 0
		if w.state != nil {
			stateCount = len(w.state.Snapshot())
		}
		return true, w.emit(ctx, Action{Command: "NOTICE", Target: ev.Nick, Text: fmt.Sprintf("Golog admin: account=%s state_keys=%d rules=active", ev.Account, stateCount)})
	case "reload":
		if w.engineLoader == nil {
			return true, w.emit(ctx, Action{Command: "NOTICE", Target: ev.Nick, Text: "reload unavailable"})
		}
		candidate, err := w.engineLoader()
		if err != nil {
			return true, w.emit(ctx, Action{Command: "NOTICE", Target: ev.Nick, Text: "reload rejected: " + err.Error()})
		}
		w.engine = candidate
		return true, w.emit(ctx, Action{Command: "NOTICE", Target: ev.Nick, Text: "Prolog rules reloaded"})
	case "join":
		if len(fields) != 3 {
			return true, w.emit(ctx, Action{Command: "NOTICE", Target: ev.Nick, Text: "usage: !admin join <#channel>"})
		}
		return true, w.emit(ctx, Action{Command: "JOIN", Target: fields[2]})
	case "part":
		if len(fields) < 3 {
			return true, w.emit(ctx, Action{Command: "NOTICE", Target: ev.Nick, Text: "usage: !admin part <#channel> [reason]"})
		}
		reason := ""
		if len(fields) > 3 {
			reason = strings.Join(fields[3:], " ")
		}
		return true, w.emit(ctx, Action{Command: "PART", Target: fields[2], Text: reason})
	case "state":
		if w.state == nil {
			return true, w.emit(ctx, Action{Command: "NOTICE", Target: ev.Nick, Text: "state unavailable"})
		}
		if len(fields) == 2 {
			return true, w.emit(ctx, Action{Command: "NOTICE", Target: ev.Nick, Text: gstate.Flatten(w.state.Snapshot())})
		}
		key := strings.Join(fields[2:], " ")
		value, ok := w.state.Get(key)
		if !ok {
			return true, w.emit(ctx, Action{Command: "NOTICE", Target: ev.Nick, Text: "state key not found: " + key})
		}
		return true, w.emit(ctx, Action{Command: "NOTICE", Target: ev.Nick, Text: key + "=" + value})
	default:
		return true, w.emit(ctx, Action{Command: "NOTICE", Target: ev.Nick, Text: "unknown admin command"})
	}
}

func (w *Worker) handleTimer(ctx context.Context, ev Event) error {
	if w.state != nil {
		actions, err := w.engine.QueryActions(
			`catch(on_timer_state(?, ?, ?, Command, Target, Arg, Text), _, fail).`,
			ev.Name, eventTime(ev.Time), gstate.Flatten(w.state.Snapshot()),
		)
		if err != nil {
			return fmt.Errorf("prolog on_timer_state/7: %w", err)
		}
		if len(actions) > 0 {
			return w.applyRuleActions(ctx, actions)
		}
	}

	actions, err := w.engine.QueryActions(
		`catch(on_timer(?, ?, Command, Target, Arg, Text), _, fail).`,
		ev.Name, eventTime(ev.Time),
	)
	if err != nil {
		return fmt.Errorf("prolog on_timer/6: %w", err)
	}
	return w.applyRuleActions(ctx, actions)
}

func (w *Worker) applyRuleActions(ctx context.Context, actions []gprolog.RuleAction) error {
	for _, a := range actions {
		command := strings.ToUpper(a.Command)
		switch command {
		case "STATE_SET":
			if w.state == nil {
				continue
			}
			if err := w.state.Set(a.Target, a.Text); err != nil {
				return fmt.Errorf("state set %q: %w", a.Target, err)
			}
		case "STATE_DELETE":
			if w.state == nil {
				continue
			}
			if err := w.state.Delete(a.Target); err != nil {
				return fmt.Errorf("state delete %q: %w", a.Target, err)
			}
		default:
			if err := w.emit(ctx, Action{Command: command, Target: a.Target, Arg: a.Arg, Text: a.Text}); err != nil {
				return err
			}
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
