% Golog M1.1 example rule pack: IRC help.
% Enable with GOLOG_RULES=rules/hello.pl,rules/packs/irc-help

on_privmsg(_Nick, _Account, Target, "!irc ping", _Time, _Tags,
           "IRC PING checks whether the connection is alive; servers answer with PONG.").

on_privmsg(_Nick, _Account, Target, "!irc join", _Time, _Tags,
           "IRC JOIN enters a channel, for example: JOIN #golog.").
