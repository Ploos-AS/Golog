# Golog

Golog is a full-featured IRC bot written in Go with an embedded Prolog rule and scripting engine.

## Design rules

- Golog is a complete IRC bot on its own.
- Prolog is a first-class rule/scripting layer, not an external service.
- `github.com/ichiban/prolog` is the default embedded Prolog backend.
- Prolog execution is serialized through a worker/event queue initially.
- BotAI, BotWeb and PBMP are optional integrations; Golog must not depend on them.

## M0

M0 establishes the standalone Go runtime, embedded Prolog engine, event model and first rule-driven IRC behavior.

### M0.1 bootstrap

- Go module and `cmd/golog`
- internal Prolog engine abstraction
- ichiban/prolog backend
- load/query smoke test
- initial `rules/hello.pl`

### M0.2 IRC event pipeline

- normalized IRC event/action model
- serialized Prolog worker
- minimal IRC message parser
- minimal IRC client core with NICK/USER, PING/PONG and PRIVMSG handling
- `PRIVMSG -> Prolog on_privmsg/4 -> PRIVMSG` path
- unit tests for IRC parsing and the Prolog worker

Current demo rule:

```prolog
on_privmsg(_Nick, _Target, "!hello", "Hello from Golog Prolog!").
```

BotAI, BotWeb and PBMP are not required anywhere in this path.

## License

Software is intended to be MIT licensed unless otherwise noted.
