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
	gprolog "github.com/Ploos-AS/Golog/internal/prolog"
	"github.com/Ploos-AS/Golog/internal/prolog/ichiban"
	"github.com/Ploos-AS/Golog/internal/state"
)

func main() {
	cfg, err := config.FromEnv()
	if err != nil {
		log.Fatal(err)
	}

	engine, err := loadEngine(cfg.RulePath)
	if err != nil {
		log.Fatal(err)
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

	hup := make(chan os.Signal, 1)
	signal.Notify(hup, syscall.SIGHUP)
	defer signal.Stop(hup)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-hup:
				candidate, err := loadEngine(cfg.RulePath)
				if err != nil {
					log.Printf("rule reload rejected; keeping previous rules: %v", err)
					continue
				}
				if err := worker.Reload(ctx, candidate); err != nil {
					if ctx.Err() == nil {
						log.Printf("rule reload failed; keeping previous rules: %v", err)
					}
					continue
				}
				log.Printf("Prolog rules reloaded from %s", cfg.RulePath)
			}
		}
	}()

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

	fmt.Printf("Golog M0.10: connecting to %s (TLS=%t, SASL=%t, timer=%t, state=%s)\n", cfg.Server, cfg.TLS, cfg.SASLUser != "", cfg.TimerInterval > 0, cfg.StatePath)
	if err := runtime.Run(ctx); err != nil && err != context.Canceled {
		log.Fatal(err)
	}
}

func loadEngine(rulePath string) (gprolog.Engine, error) {
	source, err := os.ReadFile(rulePath)
	if err != nil {
		return nil, fmt.Errorf("read rules: %w", err)
	}
	engine := ichiban.New()
	if err := engine.Load(string(source)); err != nil {
		return nil, fmt.Errorf("load rules: %w", err)
	}
	return engine, nil
}
