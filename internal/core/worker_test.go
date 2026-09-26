package core

import (
	"context"
	"testing"
	"time"

	"github.com/Ploos-AS/Golog/internal/prolog/ichiban"
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
	if err := engine.Load(`on_privmsg("alice", "aliceacct", "#golog", "!who", "2026-09-26T12:00:00Z", Tags, Reply) :- atom_length(Tags, N), N > 0, Reply = "rich-ok".`); err != nil {
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

func TestFlattenTagsStable(t *testing.T) {
	got := flattenTags(map[string]string{"time": "t", "account": "a"})
	if got != "account=a;time=t" {
		t.Fatalf("got %q", got)
	}
}
