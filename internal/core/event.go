package core

import "time"

// EventType identifies an event entering the rule engine.
type EventType string

const (
	EventPrivmsg EventType = "privmsg"
)

// Event is the normalized input passed from IRC into the Prolog worker.
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
