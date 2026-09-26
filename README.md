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

```prolog
on_timer(Name, FiredAt, Command, Target, Arg, Text).
```

Enable a periodic timer at runtime:

```sh
GOLOG_TIMER_NAME=heartbeat \
GOLOG_TIMER_INTERVAL=60s \
go run ./cmd/golog
```

The scheduler also supports one-shot delayed timers through its internal `After` API, alongside recurring `Every` timers and cancellation/replacement by timer name.

### M0.9 persistent state

Golog has a small Go-owned persistent state store. The default file is `data/state.json`; override it with `GOLOG_STATE`.

State-aware Prolog hooks receive a deterministic escaped snapshot:

```prolog
on_privmsg_state(Nick, Account, Target, Text, ServerTime, Tags, State,
                 Command, ActionTarget, Arg, ActionText).

on_timer_state(Name, FiredAt, State, Command, Target, Arg, Text).
```

Two internal commands mutate state and are never sent to IRC:

- `STATE_SET` — `Target` is the key and `Text` is the value.
- `STATE_DELETE` — `Target` is the key.

The JSON file is written through a temporary file and rename, and runtime state files are ignored by git.

### M0.10 safe Prolog hot reload

Send `SIGHUP` to the running Golog process to reload the configured `.pl` rule file without disconnecting from IRC:

```sh
kill -HUP <golog-pid>
```

Reload is transactional:

1. the rule file is read into a fresh `ichiban/prolog` interpreter,
2. the candidate interpreter must load successfully,
3. the worker swaps engines only between serialized events,
4. the existing IRC connection, scheduler and persistent state remain untouched.

If the new rule file is invalid, Golog logs the reload error and continues using the previous working rules. Tests verify both rule replacement and state preservation across an engine swap.

### M0.11 operator/admin layer

Privileged IRC administration is handled by the Go core, not by normal Prolog rules. Configure one or more allowed IRC account names:

```sh
GOLOG_ADMIN_ACCOUNTS=aliceacct,bobacct
```

Access is based on authenticated IRC account data (`account-tag` / account state), never on nickname alone. Admin accounts can use:

```text
!admin status
!admin reload
!admin join #channel
!admin part #channel [reason]
!admin state [key]
!admin help
```

`!admin reload` uses the same validated rule loader as SIGHUP. Invalid rules are rejected before the active engine is replaced. Admin commands are intercepted before the Prolog rule pipeline, so ordinary rule output cannot impersonate the operator layer.

### M0.12 observability

Golog uses structured JSON logging through Go `slog`. Runtime events such as startup, IRC connect/disconnect, rule reload and observability HTTP startup are emitted as structured records.

An optional dependency-free HTTP endpoint can be enabled with:

```sh
GOLOG_HTTP_ADDR=127.0.0.1:8080
```

Endpoints:

- `/healthz` — process liveness.
- `/readyz` — HTTP 200 only while the IRC runtime is connected/ready; otherwise HTTP 503.
- `/metrics` — Prometheus text exposition format.

Metrics include IRC connectivity, readiness, event/action counters, reconnects, reload successes/failures, state persistence errors, worker event-queue depth/capacity and process uptime. The HTTP listener is disabled by default, so Golog remains a standalone IRC bot with no mandatory web service.

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
