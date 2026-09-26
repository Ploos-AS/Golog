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

func TestParseIRCv3Tags(t *testing.T) {
	m := Parse("@account=alice;time=2026-09-26T12:00:00.123Z;example=hello\\sworld :alice!u@example PRIVMSG #golog :hi")
	if m.Tags["account"] != "alice" {
		t.Fatalf("account tag = %q", m.Tags["account"])
	}
	if m.Tags["time"] != "2026-09-26T12:00:00.123Z" {
		t.Fatalf("time tag = %q", m.Tags["time"])
	}
	if m.Tags["example"] != "hello world" {
		t.Fatalf("escaped tag = %q", m.Tags["example"])
	}
}
