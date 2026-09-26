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
	RulePath         string
}

func FromEnv() (Config, error) {
	cfg := Config{
		Server:           getenv("GOLOG_IRC_SERVER", "irc.libera.chat:6697"),
		Nick:             getenv("GOLOG_IRC_NICK", "Golog"),
		User:             getenv("GOLOG_IRC_USER", "golog"),
		Real:             getenv("GOLOG_IRC_REAL", "Golog IRC bot"),
		RulePath:         getenv("GOLOG_RULES", "rules/hello.pl"),
		ReconnectInitial: time.Second,
		ReconnectMax:     30 * time.Second,
	}

	tlsValue := getenv("GOLOG_IRC_TLS", "true")
	tlsEnabled, err := strconv.ParseBool(tlsValue)
	if err != nil {
		return Config{}, fmt.Errorf("GOLOG_IRC_TLS: %w", err)
	}
	cfg.TLS = tlsEnabled

	if raw := strings.TrimSpace(os.Getenv("GOLOG_IRC_CHANNELS")); raw != "" {
		for _, ch := range strings.Split(raw, ",") {
			ch = strings.TrimSpace(ch)
			if ch != "" {
				cfg.Channels = append(cfg.Channels, ch)
			}
		}
	}

	if cfg.Server == "" {
		return Config{}, fmt.Errorf("GOLOG_IRC_SERVER must not be empty")
	}
	return cfg, nil
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
