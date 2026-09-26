package core

import "time"

// EventType identifies an event entering the rule engine.
type EventType string

const (
	EventPrivmsg EventType = "privmsg"
	EventJoin    EventType = "join"
	EventPart    EventType = "part"
	EventQuit    EventType = "quit"
	EventNick    EventType = "nick"
	EventAccount EventType = "account"
	EventTimer   EventType = "timer"
)

// Event is the normalized input passed from IRC or the scheduler into the Prolog worker.
// Fields are reused by event type: Target is normally a channel or new nick,
// Text carries message/reason data, Account carries IRCv3 account state, and
// Name identifies a scheduled timer event.
type Event struct {
	Type    EventType
	Nick    string
	Target  string
	Text    string
	Account string
	Name    string
	Time    time.Time
	Tags    map[string]string
}

// Action is an operation produced by the rule engine.
// Arg is used for commands that need one extra structured parameter, such as
// MODE modes/arguments or the nick being kicked by KICK.
type Action struct {
	Command string
	Target  string
	Arg     string
	Text    string
}
