package irc

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"log/slog"
	"net"
	"strings"
	"time"

	"github.com/Ploos-AS/Golog/internal/config"
	"github.com/Ploos-AS/Golog/internal/core"
	"github.com/Ploos-AS/Golog/internal/observability"
)

type Runtime struct {
	Config  config.Config
	Client  *Client
	Events  chan<- core.Event
	Actions <-chan core.Action
	Dial    func(context.Context, string, bool) (net.Conn, error)
	Metrics *observability.Metrics
}

func (r *Runtime) Run(ctx context.Context) error {
	if r.Client == nil {
		r.Client = &Client{
			Nick: r.Config.Nick, User: r.Config.User, Real: r.Config.Real,
			Channels: r.Config.Channels, SASLUser: r.Config.SASLUser, SASLPass: r.Config.SASLPass,
		}
	}
	if r.Dial == nil {
		r.Dial = dial
	}

	backoff := r.Config.ReconnectInitial
	if backoff <= 0 {
		backoff = time.Second
	}
	maxBackoff := r.Config.ReconnectMax
	if maxBackoff <= 0 {
		maxBackoff = 30 * time.Second
	}

	first := true
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		conn, err := r.Dial(ctx, r.Config.Server, r.Config.TLS)
		if err == nil {
			if r.Metrics != nil {
				r.Metrics.SetConnected(true)
				r.Metrics.SetReady(true)
			}
			slog.Info("IRC connected", "server", r.Config.Server, "tls", r.Config.TLS)
			err = r.runConnection(ctx, conn)
			_ = conn.Close()
		}
		if r.Metrics != nil {
			r.Metrics.SetConnected(false)
			r.Metrics.SetReady(false)
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err != nil && !errors.Is(err, io.EOF) {
			slog.Warn("IRC connection ended", "error", err, "server", r.Config.Server)
		}
		if !first && r.Metrics != nil {
			r.Metrics.IncReconnects()
		}
		first = false

		t := time.NewTimer(backoff)
		select {
		case <-ctx.Done():
			t.Stop()
			return ctx.Err()
		case <-t.C:
		}
		backoff *= 2
		if backoff > maxBackoff {
			backoff = maxBackoff
		}
	}
}

func (r *Runtime) runConnection(ctx context.Context, conn net.Conn) error {
	if err := writeRegistration(conn, r.Client); err != nil {
		return err
	}
	return r.Client.RunRegistered(ctx, conn, r.Events, r.Actions)
}

func dial(ctx context.Context, address string, useTLS bool) (net.Conn, error) {
	d := &net.Dialer{}
	if !useTLS {
		return d.DialContext(ctx, "tcp", address)
	}
	host := address
	if h, _, err := net.SplitHostPort(address); err == nil {
		host = h
	}
	return tls.DialWithDialer(d, "tcp", address, &tls.Config{ServerName: host, MinVersion: tls.VersionTLS12})
}

func writeRegistration(conn net.Conn, c *Client) error {
	if c.Nick == "" {
		c.Nick = "Golog"
	}
	if c.User == "" {
		c.User = "golog"
	}
	if c.Real == "" {
		c.Real = "Golog IRC bot"
	}
	_, err := conn.Write([]byte("CAP LS 302\r\nNICK " + sanitizeParam(c.Nick) + "\r\nUSER " + sanitizeParam(c.User) + " 0 * :" + sanitizeText(c.Real) + "\r\n"))
	return err
}

func joinChannels(conn net.Conn, channels []string) error {
	for _, channel := range channels {
		channel = strings.TrimSpace(channel)
		if channel == "" {
			continue
		}
		if _, err := conn.Write([]byte("JOIN " + sanitizeParam(channel) + "\r\n")); err != nil {
			return err
		}
	}
	return nil
}
