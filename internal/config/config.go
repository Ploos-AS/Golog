package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Server           string
	TLS              bool
	Nick             string
	User             string
	Real             string
	Channels         []string
	ReconnectInitial time.Duration
	ReconnectMax     time.Duration
	RulePaths        []string
	StatePath        string
	SASLUser         string
	SASLPass         string
	TimerName        string
	TimerInterval    time.Duration
	AdminAccounts    []string
	HTTPAddr         string
}

func FromEnv() (Config, error) {
	cfg := Config{
		Server:           getenv("GOLOG_IRC_SERVER", "irc.libera.chat:6697"),
		Nick:             getenv("GOLOG_IRC_NICK", "Golog"),
		User:             getenv("GOLOG_IRC_USER", "golog"),
		Real:             getenv("GOLOG_IRC_REAL", "Golog IRC bot"),
		StatePath:        getenv("GOLOG_STATE", "data/state.json"),
		ReconnectInitial: time.Second,
		ReconnectMax:     30 * time.Second,
		SASLUser:         os.Getenv("GOLOG_IRC_SASL_USER"),
		SASLPass:         os.Getenv("GOLOG_IRC_SASL_PASS"),
		TimerName:        getenv("GOLOG_TIMER_NAME", "heartbeat"),
		HTTPAddr:         strings.TrimSpace(os.Getenv("GOLOG_HTTP_ADDR")),
	}

	tlsValue := getenv("GOLOG_IRC_TLS", "true")
	tlsEnabled, err := strconv.ParseBool(tlsValue)
	if err != nil {
		return Config{}, fmt.Errorf("GOLOG_IRC_TLS: %w", err)
	}
	cfg.TLS = tlsEnabled

	cfg.Channels = splitCSV(os.Getenv("GOLOG_IRC_CHANNELS"))
	cfg.AdminAccounts = splitCSV(os.Getenv("GOLOG_ADMIN_ACCOUNTS"))
	cfg.RulePaths = splitCSV(getenv("GOLOG_RULES", "rules/hello.pl"))

	if raw := strings.TrimSpace(os.Getenv("GOLOG_TIMER_INTERVAL")); raw != "" {
		interval, err := time.ParseDuration(raw)
		if err != nil {
			return Config{}, fmt.Errorf("GOLOG_TIMER_INTERVAL: %w", err)
		}
		if interval <= 0 {
			return Config{}, fmt.Errorf("GOLOG_TIMER_INTERVAL must be greater than zero")
		}
		cfg.TimerInterval = interval
	}

	if cfg.Server == "" {
		return Config{}, fmt.Errorf("GOLOG_IRC_SERVER must not be empty")
	}
	if len(cfg.RulePaths) == 0 {
		return Config{}, fmt.Errorf("GOLOG_RULES must name at least one rule file or directory")
	}
	if cfg.StatePath == "" {
		return Config{}, fmt.Errorf("GOLOG_STATE must not be empty")
	}
	if (cfg.SASLUser == "") != (cfg.SASLPass == "") {
		return Config{}, fmt.Errorf("GOLOG_IRC_SASL_USER and GOLOG_IRC_SASL_PASS must be set together")
	}
	if cfg.SASLUser != "" && !cfg.TLS {
		return Config{}, fmt.Errorf("SASL PLAIN requires GOLOG_IRC_TLS=true")
	}
	return cfg, nil
}

func splitCSV(raw string) []string {
	var out []string
	for _, value := range strings.Split(raw, ",") {
		value = strings.TrimSpace(value)
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
