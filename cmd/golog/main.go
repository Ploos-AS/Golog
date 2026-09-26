package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/Ploos-AS/Golog/internal/config"
	"github.com/Ploos-AS/Golog/internal/core"
	"github.com/Ploos-AS/Golog/internal/irc"
	"github.com/Ploos-AS/Golog/internal/prolog/ichiban"
)

func main() {
	cfg, err := config.FromEnv()
	if err != nil {
		log.Fatal(err)
	}

	source, err := os.ReadFile(cfg.RulePath)
	if err != nil {
		log.Fatalf("read rules: %v", err)
	}

	engine := ichiban.New()
	if err := engine.Load(string(source)); err != nil {
		log.Fatalf("load rules: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	worker := core.NewWorker(engine, 64)
	go worker.Run(ctx)

	runtime := &irc.Runtime{
		Config: cfg,
		Client: &irc.Client{
			Nick:     cfg.Nick,
			User:     cfg.User,
			Real:     cfg.Real,
			Channels: cfg.Channels,
		},
		Events:  worker.Events(),
		Actions: worker.Actions(),
	}

	fmt.Printf("Golog M0.3: connecting to %s (TLS=%t)\n", cfg.Server, cfg.TLS)
	if err := runtime.Run(ctx); err != nil && err != context.Canceled {
		log.Fatal(err)
	}
}
