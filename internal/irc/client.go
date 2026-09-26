package irc

import (
	"bufio"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"sort"
	"strings"
	"time"

	"github.com/Ploos-AS/Golog/internal/core"
)

type Client struct {
	Nick     string
	User     string
	Real     string
	Channels []string
	SASLUser string
	SASLPass string
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
		advertised := map[string]bool{}
		requested := map[string]bool{}
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
				if len(m.Params) < 2 {
					continue
				}
				sub := strings.ToUpper(m.Params[1])
				switch sub {
				case "LS":
					for _, cap := range strings.Fields(m.Trail) {
						name, _, _ := strings.Cut(cap, "=")
						advertised[name] = true
					}
					continued := len(m.Params) >= 3 && m.Params[2] == "*"
					if !continued {
						caps := desiredCapabilities(advertised, c.SASLUser != "")
						if len(caps) == 0 {
							if _, err := fmt.Fprint(conn, "CAP END\r\n"); err != nil {
								errCh <- err
								return
							}
						} else {
							for _, cap := range caps {
								requested[cap] = true
							}
							if _, err := fmt.Fprintf(conn, "CAP REQ :%s\r\n", strings.Join(caps, " ")); err != nil {
								errCh <- err
								return
							}
						}
					}
				case "ACK":
					if requested["sasl"] {
						if strings.Contains(" "+m.Trail+" ", " sasl ") {
							if _, err := fmt.Fprint(conn, "AUTHENTICATE PLAIN\r\n"); err != nil {
								errCh <- err
								return
							}
						} else {
							errCh <- fmt.Errorf("server did not ACK requested SASL capability")
							return
						}
					} else if _, err := fmt.Fprint(conn, "CAP END\r\n"); err != nil {
						errCh <- err
						return
					}
				case "NAK":
					if requested["sasl"] {
						errCh <- fmt.Errorf("server rejected requested SASL capability")
						return
					}
					if _, err := fmt.Fprint(conn, "CAP END\r\n"); err != nil {
						errCh <- err
						return
					}
				}
			case "AUTHENTICATE":
				if c.SASLUser != "" && (m.Trail == "+" || (len(m.Params) > 0 && m.Params[0] == "+")) {
					payload := base64.StdEncoding.EncodeToString([]byte("\x00" + c.SASLUser + "\x00" + c.SASLPass))
					if _, err := fmt.Fprintf(conn, "AUTHENTICATE %s\r\n", payload); err != nil {
						errCh <- err
						return
					}
				}
			case "903":
				if _, err := fmt.Fprint(conn, "CAP END\r\n"); err != nil {
					errCh <- err
					return
				}
			case "904", "905", "906", "907":
				errCh <- fmt.Errorf("SASL authentication failed (%s)", m.Command)
				return
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
				ev := core.Event{Type: core.EventPrivmsg, Nick: nick, Target: target, Text: m.Trail, Tags: m.Tags, Account: m.Tags["account"]}
				if raw := m.Tags["time"]; raw != "" {
					if ts, err := time.Parse(time.RFC3339Nano, raw); err == nil {
						ev.Time = ts
					}
				}
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

func desiredCapabilities(advertised map[string]bool, wantSASL bool) []string {
	wanted := []string{"account-notify", "account-tag", "extended-join", "message-tags", "server-time"}
	if wantSASL {
		wanted = append(wanted, "sasl")
	}
	var out []string
	for _, cap := range wanted {
		if advertised[cap] {
			out = append(out, cap)
		}
	}
	sort.Strings(out)
	return out
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
