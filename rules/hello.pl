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

% M0.9 persistent state contracts:
% on_privmsg_state(Nick, Account, Target, Text, ServerTime, Tags, State,
%                  Command, ActionTarget, Arg, ActionText).
% on_timer_state(Name, FiredAt, State, Command, Target, Arg, Text).
%
% State is a deterministic escaped key=value snapshot. STATE_SET and
% STATE_DELETE are internal commands consumed by Go and never sent to IRC.
% Example deployment rules:
% on_privmsg_state(_Nick, _Account, _Target, "!remember", _Time, _Tags, _State,
%                  "STATE_SET", "favorite", "", "amiga").
% on_privmsg_state(_Nick, _Account, Target, "!state", _Time, _Tags, State,
%                  "PRIVMSG", Target, "", State).

% Optional lifecycle hooks (examples can be added by deployments):
% on_join(Nick, Account, Channel, ServerTime).
% on_part(Nick, Channel, Reason, ServerTime).
% on_quit(Nick, Reason, ServerTime).
% on_nick(OldNick, NewNick, ServerTime).
% on_account(Nick, Account, ServerTime).
