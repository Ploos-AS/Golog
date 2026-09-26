package core

import (
	"context"
	"strings"
	"testing"

	gprolog "github.com/Ploos-AS/Golog/internal/prolog"
	"github.com/Ploos-AS/Golog/internal/prolog/ichiban"
)

func TestAdminRequiresAuthenticatedAllowedAccount(t *testing.T) {
	engine := ichiban.New()
	w := NewWorker(engine, 4)
	w.SetAdminAccounts([]string{"aliceacct"})
	ctx := context.Background()

	if err := w.handle(ctx, Event{Type: EventPrivmsg, Nick: "alice", Text: "!admin status"}); err != nil {
		t.Fatal(err)
	}
	select {
	case action := <-w.Actions():
		if action.Command != "NOTICE" || action.Target != "alice" || !strings.Contains(action.Text, "access denied") {
			t.Fatalf("unexpected denied action: %#v", action)
		}
	default:
		t.Fatal("expected access denied notice")
	}
}

func TestAdminJoinForAllowedAccount(t *testing.T) {
	engine := ichiban.New()
	w := NewWorker(engine, 4)
	w.SetAdminAccounts([]string{"AliceAcct"})
	ctx := context.Background()

	if err := w.handle(ctx, Event{Type: EventPrivmsg, Nick: "alice", Account: "aliceacct", Text: "!admin join #ops"}); err != nil {
		t.Fatal(err)
	}
	select {
	case action := <-w.Actions():
		if action.Command != "JOIN" || action.Target != "#ops" {
			t.Fatalf("unexpected admin action: %#v", action)
		}
	default:
		t.Fatal("expected JOIN action")
	}
}

func TestAdminReloadUsesValidatedLoader(t *testing.T) {
	oldEngine := ichiban.New()
	if err := oldEngine.Load(`on_privmsg(_Nick, _Target, "!v", "old").`); err != nil {
		t.Fatal(err)
	}
	newEngine := ichiban.New()
	if err := newEngine.Load(`on_privmsg(_Nick, _Target, "!v", "new").`); err != nil {
		t.Fatal(err)
	}

	w := NewWorker(oldEngine, 4)
	w.SetAdminAccounts([]string{"aliceacct"})
	w.SetEngineLoader(func() (gprolog.Engine, error) { return newEngine, nil })
	ctx := context.Background()

	if err := w.handle(ctx, Event{Type: EventPrivmsg, Nick: "alice", Account: "aliceacct", Text: "!admin reload"}); err != nil {
		t.Fatal(err)
	}
	select {
	case action := <-w.Actions():
		if !strings.Contains(action.Text, "reloaded") {
			t.Fatalf("unexpected reload notice: %#v", action)
		}
	default:
		t.Fatal("expected reload notice")
	}

	if err := w.handle(ctx, Event{Type: EventPrivmsg, Nick: "alice", Target: "#golog", Text: "!v"}); err != nil {
		t.Fatal(err)
	}
	select {
	case action := <-w.Actions():
		if action.Text != "new" {
			t.Fatalf("new engine not active: %#v", action)
		}
	default:
		t.Fatal("expected reply from reloaded engine")
	}
}
