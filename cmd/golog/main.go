package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/Ploos-AS/Golog/internal/core"
	gprolog "github.com/Ploos-AS/Golog/internal/prolog"
	"github.com/Ploos-AS/Golog/internal/prolog/ichiban"
)

func main() {
	engine := ichiban.New()
	if err := run(engine, "rules/hello.pl"); err != nil {
		log.Fatal(err)
	}
}

func run(engine gprolog.Engine, rulePath string) error {
	source, err := os.ReadFile(rulePath)
	if err != nil {
		return fmt.Errorf("read rules: %w", err)
	}
	if err := engine.Load(string(source)); err != nil {
		return fmt.Errorf("load rules: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	worker := core.NewWorker(engine, 8)
	go worker.Run(ctx)

	worker.Events() <- core.Event{Type: core.EventPrivmsg, Nick: "smoke", Target: "#golog", Text: "!hello"}
	select {
	case err := <-worker.Errors():
		return err
	case action := <-worker.Actions():
		if action.Command != "PRIVMSG" || action.Text == "" {
			return fmt.Errorf("M0.2 event/rule smoke test failed: %#v", action)
		}
		fmt.Printf("Golog M0.2: IRC event -> Prolog -> %s ready\n", action.Command)
		return nil
	case <-time.After(time.Second):
		return fmt.Errorf("M0.2 event/rule smoke test timed out")
	}
}
