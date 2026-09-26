package irc

import (
	"reflect"
	"strings"
	"testing"

	"github.com/Ploos-AS/Golog/internal/core"
)

func TestFallbackNick(t *testing.T) {
	if got := fallbackNick("Golog"); got != "Golog_" {
		t.Fatalf("got %q", got)
	}
}

func TestChannelTarget(t *testing.T) {
	for _, target := range []string{"#golog", "&local", "+modeless", "!safe"} {
		if !isChannelTarget(target) {
			t.Fatalf("expected channel target: %q", target)
		}
	}
	if isChannelTarget("SomeNick") {
		t.Fatal("nick must not be treated as channel")
	}
}

func TestPrivmsgIsSanitizedAndLimited(t *testing.T) {
	line := limitPrivmsg("#golog\r\nOPER bad", strings.Repeat("x", 1000)+"\r\nQUIT")
	if strings.Contains(line, "\r\nOPER") || strings.Contains(line, "\r\nQUIT") {
		t.Fatalf("line injection survived: %q", line)
	}
	wire := strings.TrimSuffix(line, "\r\n")
	if len(wire) > maxIRCLine {
		t.Fatalf("line too long: %d", len(wire))
	}
}

func TestFormatActions(t *testing.T) {
	tests := []struct {
		a    core.Action
		want string
	}{
		{core.Action{Command: "NOTICE", Target: "alice", Text: "hello"}, "NOTICE alice :hello\r\n"},
		{core.Action{Command: "JOIN", Target: "#golog"}, "JOIN #golog\r\n"},
		{core.Action{Command: "PART", Target: "#golog", Text: "bye"}, "PART #golog :bye\r\n"},
		{core.Action{Command: "TOPIC", Target: "#golog", Text: "New topic"}, "TOPIC #golog :New topic\r\n"},
		{core.Action{Command: "MODE", Target: "#golog", Arg: "+v alice"}, "MODE #golog +v alice\r\n"},
		{core.Action{Command: "KICK", Target: "#golog", Arg: "alice", Text: "reason"}, "KICK #golog alice :reason\r\n"},
	}
	for _, tt := range tests {
		got, ok := formatAction(tt.a)
		if !ok {
			t.Fatalf("action unexpectedly rejected: %#v", tt.a)
		}
		if got != tt.want {
			t.Fatalf("for %#v got %q want %q", tt.a, got, tt.want)
		}
	}

	if _, ok := formatAction(core.Action{Command: "OPER", Target: "root", Text: "secret"}); ok {
		t.Fatal("unsupported raw command must be rejected")
	}
}

func TestDesiredCapabilities(t *testing.T) {
	advertised := map[string]bool{
		"message-tags": true,
		"server-time":  true,
		"account-tag":  true,
		"sasl":         true,
		"echo-message": true,
	}
	got := desiredCapabilities(advertised, false)
	want := []string{"account-tag", "message-tags", "server-time"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("without SASL got %#v want %#v", got, want)
	}
	got = desiredCapabilities(advertised, true)
	want = []string{"account-tag", "message-tags", "sasl", "server-time"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("with SASL got %#v want %#v", got, want)
	}
}
