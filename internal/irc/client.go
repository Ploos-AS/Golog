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
	Nick string
	User string
	Real string
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
		for s.Scan() {
			m := Parse(s.Text())
			switch m.Command {
			case "PING":
				payload := m.Trail
				if payload == "" && len(m.Params) > 0 {
					payload = m.Params[0]
				}
				_, err := fmt.Fprintf(conn, "PONG :%s\r\n", payload)
				if err != nil {
					errCh <- err
					return
				}
			case "PRIVMSG":
				if len(m.Params) < 1 {
					continue
				}
				ev := core.Event{Type: core.EventPrivmsg, Nick: NickFromPrefix(m.Prefix), Target: m.Params[0], Text: m.Trail}
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
				if _, err := fmt.Fprintf(conn, "PRIVMSG %s :%s\r\n", a.Target, a.Text); err != nil {
					return err
				}
			}
		}
	}
}
