package irc

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"strings"

	"github.com/Ploos-AS/Golog/internal/core"
)

type Client struct {
	Nick     string
	User     string
	Real     string
	Channels []string
}

func (c *Client) Run(ctx context.Context, conn net.Conn, events chan<- core.Event, actions <-chan core.Action) error {
	if err := writeRegistration(conn, c); err != nil {
		return err
	}
	return c.RunRegistered(ctx, conn, events, actions)
}

func (c *Client) RunRegistered(ctx context.Context, conn net.Conn, events chan<- core.Event, actions <-chan core.Action) error {
	errCh := make(chan error, 1)
	go func() {
		s := bufio.NewScanner(conn)
		buf := make([]byte, 0, 4096)
		s.Buffer(buf, 64*1024)
		for s.Scan() {
			m := Parse(s.Text())
			switch m.Command {
			case "001":
				if err := joinChannels(conn, c.Channels); err != nil {
					errCh <- err
					return
				}
			case "433":
				c.Nick = fallbackNick(c.Nick)
				if _, err := fmt.Fprintf(conn, "NICK %s\r\n", sanitizeParam(c.Nick)); err != nil {
					errCh <- err
					return
				}
			case "CAP":
				if len(m.Params) >= 2 && strings.EqualFold(m.Params[1], "LS") {
					if _, err := fmt.Fprint(conn, "CAP END\r\n"); err != nil {
						errCh <- err
						return
					}
				}
			case "PING":
				payload := m.Trail
				if payload == "" && len(m.Params) > 0 {
					payload = m.Params[0]
				}
				if _, err := fmt.Fprintf(conn, "PONG :%s\r\n", sanitizeText(payload)); err != nil {
					errCh <- err
					return
				}
			case "PRIVMSG":
				if len(m.Params) < 1 {
					continue
				}
				nick := NickFromPrefix(m.Prefix)
				target := m.Params[0]
				if strings.EqualFold(target, c.Nick) || !isChannelTarget(target) {
					target = nick
				}
				ev := core.Event{Type: core.EventPrivmsg, Nick: nick, Target: target, Text: m.Trail}
				select {
				case events <- ev:
				case <-ctx.Done():
					return
				}
			}
		}
		if err := s.Err(); err != nil && err != io.EOF {
			errCh <- err
			return
		}
		errCh <- io.EOF
	}()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-errCh:
			return err
		case a := <-actions:
			if strings.EqualFold(a.Command, "PRIVMSG") {
				if _, err := fmt.Fprint(conn, limitPrivmsg(a.Target, a.Text)); err != nil {
					return err
				}
			}
		}
	}
}

func fallbackNick(nick string) string {
	nick = sanitizeParam(nick)
	if nick == "" {
		return "Golog_"
	}
	if len(nick) >= 28 {
		nick = nick[:28]
	}
	return nick + "_"
}
