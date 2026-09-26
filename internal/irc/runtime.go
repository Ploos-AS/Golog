package irc

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"time"

	"github.com/Ploos-AS/Golog/internal/config"
	"github.com/Ploos-AS/Golog/internal/core"
)

type Runtime struct {
	Config  config.Config
	Client  *Client
	Events  chan<- core.Event
	Actions <-chan core.Action
	Dial    func(context.Context, string, bool) (net.Conn, error)
}

func (r *Runtime) Run(ctx context.Context) error {
	if r.Client == nil {
		r.Client = &Client{Nick: r.Config.Nick, User: r.Config.User, Real: r.Config.Real}
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

	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		conn, err := r.Dial(ctx, r.Config.Server, r.Config.TLS)
		if err == nil {
			err = r.runConnection(ctx, conn)
			_ = conn.Close()
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err != nil && !errors.Is(err, io.EOF) {
			fmt.Printf("golog: IRC connection ended: %v\n", err)
		}

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
	if err := joinChannels(conn, r.Config.Channels); err != nil {
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
	_, err := fmt.Fprintf(conn, "NICK %s\r\nUSER %s 0 * :%s\r\n", c.Nick, c.User, c.Real)
	return err
}

func joinChannels(conn net.Conn, channels []string) error {
	for _, channel := range channels {
		channel = strings.TrimSpace(channel)
		if channel == "" {
			continue
		}
		if _, err := fmt.Fprintf(conn, "JOIN %s\r\n", channel); err != nil {
			return err
		}
	}
	return nil
}
