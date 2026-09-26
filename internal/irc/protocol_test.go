package irc

import (
	"reflect"
	"strings"
	"testing"
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
