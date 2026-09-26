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
)

// Event is the normalized input passed from IRC into the Prolog worker.
// Fields are reused by event type: Target is normally a channel or new nick,
// Text carries message/reason data, and Account carries IRCv3 account state.
type Event struct {
	Type    EventType
	Nick    string
	Target  string
	Text    string
	Account string
	Time    time.Time
	Tags    map[string]string
}

// Action is an operation produced by the rule engine.
type Action struct {
	Command string
	Target  string
	Text    string
}
