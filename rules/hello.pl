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

% M0.7 multi-action contract:
% on_privmsg_action(Nick, Account, Target, Text, ServerTime, Tags,
%                   Command, ActionTarget, Arg, ActionText).
% Every matching solution becomes one safe allowlisted IRC action.
on_privmsg_action(Nick, _Account, Target, "!multi", _ServerTime, _Tags,
                  "NOTICE", Nick, "", "private notice from Prolog").
on_privmsg_action(_Nick, _Account, Target, "!multi", _ServerTime, _Tags,
                  "PRIVMSG", Target, "", "channel reply from Prolog").

% M0.8 scheduled action contract:
% on_timer(Name, FiredAt, Command, Target, Arg, Text).
% Enable a periodic timer with GOLOG_TIMER_INTERVAL and GOLOG_TIMER_NAME.
% Example deployment rule:
% on_timer("heartbeat", FiredAt, "NOTICE", "#golog", "", Text) :-
%     format(atom(Text), "Golog timer fired at ~w", [FiredAt]).

% Optional lifecycle hooks (examples can be added by deployments):
% on_join(Nick, Account, Channel, ServerTime).
% on_part(Nick, Channel, Reason, ServerTime).
% on_quit(Nick, Reason, ServerTime).
% on_nick(OldNick, NewNick, ServerTime).
% on_account(Nick, Account, ServerTime).
