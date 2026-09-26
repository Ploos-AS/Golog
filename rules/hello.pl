% Golog M0 rules.

hello(golog).

respond(hello, "Hello from Golog Prolog!").

% Legacy M0.2 contract. Kept for backward compatibility.
on_privmsg(_Nick, _Target, "!hello", "Hello from Golog Prolog!").

% M0.6 rich IRCv3-aware contract:
% on_privmsg(Nick, Account, Target, Text, ServerTime, Tags, Reply).
% Tags is a stable semicolon-separated key=value string.
on_privmsg(Nick, Account, _Target, "!context", ServerTime, Tags, Reply) :-
    format(atom(Reply), "nick=~w account=~w time=~w tags=~w", [Nick, Account, ServerTime, Tags]).

% Optional lifecycle hooks (examples can be added by deployments):
% on_join(Nick, Account, Channel, ServerTime).
% on_part(Nick, Channel, Reason, ServerTime).
% on_quit(Nick, Reason, ServerTime).
% on_nick(OldNick, NewNick, ServerTime).
% on_account(Nick, Account, ServerTime).
