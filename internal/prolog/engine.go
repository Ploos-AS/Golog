package prolog

// Engine is the internal abstraction for the rule engine.
// Golog uses ichiban/prolog by default, but the core must not depend
// directly on one concrete Prolog implementation.
type Engine interface {
	Load(source string) error
	Ask(query string, args ...any) (bool, error)
	QueryReply(query string, args ...any) (string, bool, error)
}
