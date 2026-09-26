package irc

import "testing"

func TestParsePrivmsg(t *testing.T) {
	m := Parse(":alice!u@example PRIVMSG #golog :!hello\r\n")
	if m.Command != "PRIVMSG" {
		t.Fatalf("command = %q", m.Command)
	}
	if got := NickFromPrefix(m.Prefix); got != "alice" {
		t.Fatalf("nick = %q", got)
	}
	if len(m.Params) != 1 || m.Params[0] != "#golog" {
		t.Fatalf("params = %#v", m.Params)
	}
	if m.Trail != "!hello" {
		t.Fatalf("trail = %q", m.Trail)
	}
}
