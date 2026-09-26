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
- outgoing messages limited to the IRC 510-byte pre-CRLF limit
- larger scanner buffer for safe handling of extended IRCv3 input

### M0.5 IRCv3 capabilities

- parses IRCv3 message tags, including tag escaping
- requests only capabilities actually advertised by the server
- supports `message-tags`, `server-time`, `account-tag`, `account-notify` and `extended-join`
- optional SASL PLAIN over TLS
- exposes account, server time and tags on normalized events

### M0.6 Prolog IRC event API

- rich IRCv3-aware `on_privmsg/7`
- backward-compatible fallback to `on_privmsg/4`
- lifecycle hooks for JOIN, PART, QUIT, NICK and ACCOUNT

```prolog
on_join(Nick, Account, Channel, ServerTime).
on_part(Nick, Channel, Reason, ServerTime).
on_quit(Nick, Reason, ServerTime).
on_nick(OldNick, NewNick, ServerTime).
on_account(Nick, Account, ServerTime).
```

### M0.7 Prolog action API

A PRIVMSG can produce zero, one or many IRC actions using:

```prolog
on_privmsg_action(Nick, Account, Target, Text, ServerTime, Tags,
                  Command, ActionTarget, Arg, ActionText).
```

Supported commands are deliberately allowlisted: `PRIVMSG`, `NOTICE`, `JOIN`, `PART`, `TOPIC`, `MODE`, and `KICK`. Unsupported commands are ignored; Prolog cannot emit arbitrary raw IRC lines.

### M0.8 scheduler and timer rules

The internal scheduler emits timer events onto the same serialized event queue used by IRC, so timer rules never bypass the Prolog worker.

Periodic timer rule:

```prolog
on_timer(Name, FiredAt, Command, Target, Arg, Text).
```

Example:

```prolog
on_timer("heartbeat", FiredAt, "NOTICE", "#golog", "", Text) :-
    format(atom(Text), "Golog timer fired at ~w", [FiredAt]).
```

Enable a periodic timer at runtime:

```sh
GOLOG_TIMER_NAME=heartbeat \
GOLOG_TIMER_INTERVAL=60s \
go run ./cmd/golog
```

The scheduler also supports one-shot delayed timers through its internal `After` API, alongside recurring `Every` timers and cancellation/replacement by timer name.

## Running

```sh
GOLOG_IRC_SERVER=irc.libera.chat:6697 \
GOLOG_IRC_TLS=true \
GOLOG_IRC_NICK=GologTest \
GOLOG_IRC_CHANNELS=#golog-test \
go run ./cmd/golog
```

Optional SASL over TLS:

```sh
GOLOG_IRC_SASL_USER=GologTest \
GOLOG_IRC_SASL_PASS='secret' \
go run ./cmd/golog
```

BotAI, BotWeb and PBMP are not required anywhere in this path.

## License

Software is MIT licensed unless otherwise noted.
