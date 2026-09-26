package irc

import (
	"bufio"
	"net"
	"strings"
	"testing"
)

func TestRegistrationAndJoin(t *testing.T) {
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	done := make(chan error, 1)
	go func() {
		c := &Client{Nick: "Golog", User: "golog", Real: "Golog IRC bot"}
		if err := writeRegistration(client, c); err != nil {
			done <- err
			return
		}
		done <- joinChannels(client, []string{"#golog", "#bots"})
	}()

	r := bufio.NewReader(server)
	var got []string
	for i := 0; i < 4; i++ {
		line, err := r.ReadString('\n')
		if err != nil {
			t.Fatal(err)
		}
		got = append(got, strings.TrimSpace(line))
	}

	want := []string{
		"NICK Golog",
		"USER golog 0 * :Golog IRC bot",
		"JOIN #golog",
		"JOIN #bots",
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("line %d: got %q want %q", i, got[i], want[i])
		}
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}
