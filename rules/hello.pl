% Golog M0 rules.

hello(golog).

respond(hello, "Hello from Golog Prolog!").

% M0.2: IRC PRIVMSG -> Prolog -> IRC reply.
on_privmsg(_Nick, _Target, "!hello", "Hello from Golog Prolog!").
