package core

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/Ploos-AS/Golog/internal/prolog/ichiban"
	"github.com/Ploos-AS/Golog/internal/state"
)

func TestPrivmsgThroughLegacyPrologWorker(t *testing.T) {
	engine := ichiban.New()
	if err := engine.Load(`on_privmsg(_Nick, _Target, "!hello", "Hello from Golog Prolog!").`); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	w := NewWorker(engine, 4)
	go w.Run(ctx)

	w.Events() <- Event{Type: EventPrivmsg, Nick: "tester", Target: "#golog", Text: "!hello"}

	select {
	case err := <-w.Errors():
		t.Fatal(err)
	case action := <-w.Actions():
		if action.Command != "PRIVMSG" || action.Target != "#golog" || action.Text != "Hello from Golog Prolog!" {
			t.Fatalf("unexpected action: %#v", action)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for Prolog action")
	}
}

func TestRichPrivmsgHook(t *testing.T) {
	engine := ichiban.New()
	if err := engine.Load(`on_privmsg("alice", "aliceacct", "#golog", "!who", "2026-09-26T12:00:00Z", Tags, Reply) :- string_length(Tags, N), N > 0, Reply = "rich-ok".`); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	w := NewWorker(engine, 4)
	go w.Run(ctx)

	w.Events() <- Event{
		Type:    EventPrivmsg,
		Nick:    "alice",
		Account: "aliceacct",
		Target:  "#golog",
		Text:    "!who",
		Time:    time.Date(2026, 9, 26, 12, 0, 0, 0, time.UTC),
		Tags:    map[string]string{"account": "aliceacct", "time": "2026-09-26T12:00:00Z"},
	}

	select {
	case err := <-w.Errors():
		t.Fatal(err)
	case action := <-w.Actions():
		if action.Text != "rich-ok" {
			t.Fatalf("unexpected action: %#v", action)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for rich Prolog action")
	}
}

func TestPrivmsgActionHookReturnsMultipleActions(t *testing.T) {
	engine := ichiban.New()
	if err := engine.Load(`
		on_privmsg_action(_Nick, _Account, "#golog", "!ops", _Time, _Tags, "NOTICE", "alice", "", "first").
		on_privmsg_action(_Nick, _Account, "#golog", "!ops", _Time, _Tags, "MODE", "#golog", "+v alice", "").
	`); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	w := NewWorker(engine, 4)
	go w.Run(ctx)
	w.Events() <- Event{Type: EventPrivmsg, Nick: "alice", Target: "#golog", Text: "!ops"}

	var got []Action
	deadline := time.After(time.Second)
	for len(got) < 2 {
		select {
		case err := <-w.Errors():
			t.Fatal(err)
		case a := <-w.Actions():
			got = append(got, a)
		case <-deadline:
			t.Fatalf("timed out waiting for actions: %#v", got)
		}
	}
	if got[0].Command != "NOTICE" || got[0].Target != "alice" || got[0].Text != "first" {
		t.Fatalf("unexpected first action: %#v", got[0])
	}
	if got[1].Command != "MODE" || got[1].Target != "#golog" || got[1].Arg != "+v alice" {
		t.Fatalf("unexpected second action: %#v", got[1])
	}
}

func TestTimerHookProducesAction(t *testing.T) {
	engine := ichiban.New()
	if err := engine.Load(`on_timer("heartbeat", _When, "NOTICE", "#golog", "", "tick").`); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	w := NewWorker(engine, 4)
	go w.Run(ctx)
	w.Events() <- Event{Type: EventTimer, Name: "heartbeat", Time: time.Date(2026, 9, 26, 12, 0, 0, 0, time.UTC)}

	select {
	case err := <-w.Errors():
		t.Fatal(err)
	case action := <-w.Actions():
		if action.Command != "NOTICE" || action.Target != "#golog" || action.Text != "tick" {
			t.Fatalf("unexpected timer action: %#v", action)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for timer action")
	}
}

func TestPersistentStateActionsAndSnapshot(t *testing.T) {
	engine := ichiban.New()
	if err := engine.Load(`
		on_privmsg_state(_Nick, _Account, _Target, "!remember", _Time, _Tags, _State,
		                 "STATE_SET", "favorite", "", "amiga").
		on_privmsg_state(_Nick, _Account, Target, "!state", _Time, _Tags, State,
		                 "PRIVMSG", Target, "", State).
	`); err != nil {
		t.Fatal(err)
	}

	store, err := state.Open(filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	w := NewWorker(engine, 4)
	w.SetStateStore(store)
	ctx := context.Background()

	if err := w.handle(ctx, Event{Type: EventPrivmsg, Nick: "alice", Target: "#golog", Text: "!remember"}); err != nil {
		t.Fatal(err)
	}
	if got, ok := store.Get("favorite"); !ok || got != "amiga" {
		t.Fatalf("persistent state = %q, %v", got, ok)
	}

	if err := w.handle(ctx, Event{Type: EventPrivmsg, Nick: "alice", Target: "#golog", Text: "!state"}); err != nil {
		t.Fatal(err)
	}
	select {
	case action := <-w.Actions():
		if action.Text != "favorite=amiga" {
			t.Fatalf("state snapshot reply = %q", action.Text)
		}
	default:
		t.Fatal("expected state-aware action")
	}
}

func TestReloadSwapsRulesAndPreservesState(t *testing.T) {
	oldEngine := ichiban.New()
	if err := oldEngine.Load(`on_privmsg(_Nick, _Target, "!version", "old").`); err != nil {
		t.Fatal(err)
	}
	newEngine := ichiban.New()
	if err := newEngine.Load(`on_privmsg(_Nick, _Target, "!version", "new").`); err != nil {
		t.Fatal(err)
	}

	store, err := state.Open(filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Set("favorite", "amiga"); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	w := NewWorker(oldEngine, 4)
	w.SetStateStore(store)
	go w.Run(ctx)

	w.Events() <- Event{Type: EventPrivmsg, Nick: "alice", Target: "#golog", Text: "!version"}
	select {
	case a := <-w.Actions():
		if a.Text != "old" {
			t.Fatalf("before reload = %q", a.Text)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out before reload")
	}

	if err := w.Reload(ctx, newEngine); err != nil {
		t.Fatal(err)
	}
	w.Events() <- Event{Type: EventPrivmsg, Nick: "alice", Target: "#golog", Text: "!version"}
	select {
	case a := <-w.Actions():
		if a.Text != "new" {
			t.Fatalf("after reload = %q", a.Text)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out after reload")
	}

	if got, ok := store.Get("favorite"); !ok || got != "amiga" {
		t.Fatalf("state lost across reload: %q, %v", got, ok)
	}
}

func TestFlattenTagsStable(t *testing.T) {
	got := flattenTags(map[string]string{"time": "t", "account": "a"})
	if got != "account=a;time=t" {
		t.Fatalf("got %q", got)
	}
}
