package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/Ploos-AS/Golog/internal/config"
	"github.com/Ploos-AS/Golog/internal/core"
	"github.com/Ploos-AS/Golog/internal/irc"
	"github.com/Ploos-AS/Golog/internal/observability"
	gprolog "github.com/Ploos-AS/Golog/internal/prolog"
	"github.com/Ploos-AS/Golog/internal/prolog/ichiban"
	"github.com/Ploos-AS/Golog/internal/rules"
	"github.com/Ploos-AS/Golog/internal/state"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	if err := run(); err != nil {
		slog.Error("Golog stopped", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.FromEnv()
	if err != nil {
		return err
	}

	engine, loadedRules, registry, err := loadEngine(cfg.RulePaths)
	if err != nil {
		return err
	}

	store, err := state.Open(cfg.StatePath)
	if err != nil {
		return fmt.Errorf("open state: %w", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	metrics := observability.NewMetrics()
	worker := core.NewWorker(engine, 64)
	worker.SetStateStore(store)
	worker.SetAdminAccounts(cfg.AdminAccounts)
	worker.SetEngineLoader(func() (gprolog.Engine, error) {
		candidate, _, _, err := loadEngine(cfg.RulePaths)
		return candidate, err
	})
	worker.SetMetrics(metrics)
	go worker.Run(ctx)

	if cfg.HTTPAddr != "" {
		go func() {
			if err := observability.ServeHTTP(ctx, cfg.HTTPAddr, metrics); err != nil && err != context.Canceled {
				slog.Error("observability HTTP stopped", "error", err)
			}
		}()
	}

	hup := make(chan os.Signal, 1)
	signal.Notify(hup, syscall.SIGHUP)
	defer signal.Stop(hup)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-hup:
				candidate, files, candidateRegistry, err := loadEngine(cfg.RulePaths)
				if err != nil {
					metrics.IncReloadFailures()
					slog.Warn("rule reload rejected; keeping previous rules", "error", err)
					continue
				}
				if err := worker.Reload(ctx, candidate); err != nil {
					if ctx.Err() == nil {
						metrics.IncReloadFailures()
						slog.Warn("rule reload failed; keeping previous rules", "error", err)
					}
					continue
				}
				registry = candidateRegistry
				slog.Info("Prolog rule packs reloaded",
					"files", files,
					"file_count", len(files),
					"packs", registry.IDs(),
					"roles", registry.Roles(),
					"capabilities", registry.Capabilities(),
				)
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
		Metrics: metrics,
	}

	slog.Info("Golog M1.2 starting",
		"server", cfg.Server,
		"tls", cfg.TLS,
		"sasl", cfg.SASLUser != "",
		"timer", cfg.TimerInterval > 0,
		"state", cfg.StatePath,
		"admins", len(cfg.AdminAccounts),
		"http", cfg.HTTPAddr,
		"rule_files", loadedRules,
		"rule_file_count", len(loadedRules),
		"packs", registry.IDs(),
		"pack_count", registry.Len(),
		"roles", registry.Roles(),
		"commands", registry.Commands(),
		"capabilities", registry.Capabilities(),
	)
	if err := runtime.Run(ctx); err != nil && err != context.Canceled {
		return err
	}
	return nil
}

func loadEngine(rulePaths []string) (gprolog.Engine, []string, *rules.Registry, error) {
	registry, err := rules.DiscoverRegistry(rulePaths)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("discover rule packs: %w", err)
	}
	engine, files, err := rules.Load(rulePaths, func() gprolog.Engine { return ichiban.New() })
	if err != nil {
		return nil, nil, nil, err
	}
	return engine, files, registry, nil
}
