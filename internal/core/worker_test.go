package core

import (
	"context"
	"testing"
	"time"

	"github.com/Ploos-AS/Golog/internal/prolog/ichiban"
)

func TestPrivmsgThroughPrologWorker(t *testing.T) {
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
