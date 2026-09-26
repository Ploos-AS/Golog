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

### M0.3 network runtime

- environment-based runtime configuration
- TCP and TLS connections
- TLS 1.2 minimum with server-name verification
- reconnect loop with exponential backoff
- automatic NICK/USER registration
- automatic channel JOIN after server `001`
- signal-aware shutdown
- runtime test for registration and JOIN output

### M0.4 IRC protocol hardening

- IRCv3 `CAP LS 302` negotiation with clean `CAP END`
- nick collision handling (`433`) using a fallback nick
- correct reply target for private messages
- reconnect naturally re-registers and rejoins configured channels
- CR/LF sanitization for registration, PONG, JOIN and PRIVMSG output
- outgoing PRIVMSG lines limited to the IRC 510-byte pre-CRLF limit
- larger scanner buffer for safe handling of extended IRCv3 input
- tests for CAP registration, nick fallback, channel detection and line-injection/length handling

Run against an IRC network:

```sh
GOLOG_IRC_SERVER=irc.libera.chat:6697 \
GOLOG_IRC_TLS=true \
GOLOG_IRC_NICK=GologTest \
GOLOG_IRC_CHANNELS=#golog-test \
go run ./cmd/golog
```

Current demo rule:

```prolog
on_privmsg(_Nick, _Target, "!hello", "Hello from Golog Prolog!").
```

BotAI, BotWeb and PBMP are not required anywhere in this path.

## License

Software is intended to be MIT licensed unless otherwise noted.
