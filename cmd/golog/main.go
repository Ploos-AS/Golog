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
	"github.com/Ploos-AS/Golog/internal/state"
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

	store, err := state.Open(cfg.StatePath)
	if err != nil {
		log.Fatalf("open state: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	worker := core.NewWorker(engine, 64)
	worker.SetStateStore(store)
	go worker.Run(ctx)

	if cfg.TimerInterval > 0 {
		scheduler := core.NewScheduler(worker.Events())
		scheduler.Every(ctx, cfg.TimerName, cfg.TimerInterval)
	}

	runtime := &irc.Runtime{
		Config: cfg,
		Client: &irc.Client{
			Nick:     cfg.Nick,
			User:     cfg.User,
			Real:     cfg.Real,
			Channels: cfg.Channels,
			SASLUser: cfg.SASLUser,
			SASLPass: cfg.SASLPass,
		},
		Events:  worker.Events(),
		Actions: worker.Actions(),
	}

	fmt.Printf("Golog M0.9: connecting to %s (TLS=%t, SASL=%t, timer=%t, state=%s)\n", cfg.Server, cfg.TLS, cfg.SASLUser != "", cfg.TimerInterval > 0, cfg.StatePath)
	if err := runtime.Run(ctx); err != nil && err != context.Canceled {
		log.Fatal(err)
	}
}
