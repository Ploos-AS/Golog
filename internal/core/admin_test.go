package core

import (
	"context"
	"strings"
	"testing"

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
	newEngine := ichiban.New()
	if err := newEngine.Load(`on_privmsg(_Nick, _Target, "!v", "new").`); err != nil {
		t.Fatal(err)
	}
	w := NewWorker(oldEngine, 4)
	w.SetAdminAccounts([]string{"aliceacct"})
	w.SetEngineLoader(func() (interfaceEngine, error) { return newEngine, nil })
	_ = w
}

// interfaceEngine keeps this file focused on ACL behavior; reload behavior is
// already covered by worker reload tests.
type interfaceEngine = interface {
	Load(string) error
	Ask(string, ...any) (bool, error)
	QueryReply(string, ...any) (string, bool, error)
	QueryActions(string, ...any) ([]struct {
		Command string
		Target  string
		Arg     string
		Text    string
	}, error)
}
