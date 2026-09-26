package irc

import (
	"bufio"
	"context"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/Ploos-AS/Golog/internal/core"
	"github.com/Ploos-AS/Golog/internal/prolog/ichiban"
)

func TestIRCToPrologToIRCSmoke(t *testing.T) {
	engine := ichiban.New()
	if err := engine.Load(`on_privmsg(_Nick, _Target, "!hello", "Hello from smoke").`); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	worker := core.NewWorker(engine, 8)
	go worker.Run(ctx)

	server, clientConn := net.Pipe()
	defer server.Close()
	defer clientConn.Close()
	_ = server.SetDeadline(time.Now().Add(3 * time.Second))

	client := &Client{Nick: "Golog", User: "golog", Real: "Golog test", Channels: []string{"#golog"}}
	done := make(chan error, 1)
	go func() {
		done <- client.Run(ctx, clientConn, worker.Events(), worker.Actions())
	}()

	r := bufio.NewReader(server)
	for _, want := range []string{"CAP LS 302", "NICK Golog", "USER golog 0 * :Golog test"} {
		line, err := r.ReadString('\n')
		if err != nil {
			t.Fatal(err)
		}
		if got := strings.TrimSpace(line); got != want {
			t.Fatalf("registration: got %q want %q", got, want)
		}
	}

	if _, err := server.Write([]byte(":server CAP Golog LS :\r\n")); err != nil {
		t.Fatal(err)
	}
	line, err := r.ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(line); got != "CAP END" {
		t.Fatalf("CAP negotiation: got %q", got)
	}

	if _, err := server.Write([]byte(":server 001 Golog :welcome\r\n")); err != nil {
		t.Fatal(err)
	}
	line, err = r.ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(line); got != "JOIN #golog" {
		t.Fatalf("join: got %q", got)
	}

	if _, err := server.Write([]byte(":alice!u@example PRIVMSG #golog :!hello\r\n")); err != nil {
		t.Fatal(err)
	}
	line, err = r.ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(line); got != "PRIVMSG #golog :Hello from smoke" {
		t.Fatalf("reply: got %q", got)
	}

	cancel()
	_ = server.Close()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("IRC client did not stop after cancellation")
	}
}
