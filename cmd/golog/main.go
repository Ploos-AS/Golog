package main

import (
	"fmt"
	"log"
	"os"

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

	ok, err := engine.Ask(`hello(?).`, "golog")
	if err != nil {
		return fmt.Errorf("query rules: %w", err)
	}
	if !ok {
		return fmt.Errorf("M0.1 Prolog smoke test failed")
	}

	fmt.Println("Golog M0.1: embedded Prolog engine ready")
	return nil
}
