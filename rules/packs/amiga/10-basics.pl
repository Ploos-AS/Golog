% Golog M1.1 example rule pack: Amiga basics.
% Enable with GOLOG_RULES=rules/hello.pl,rules/packs/amiga

on_privmsg(_Nick, _Account, Target, "!amiga arexx", _Time, _Tags,
           "ARexx is the Rexx-based scripting and application automation system used by AmigaOS.").

on_privmsg(_Nick, _Account, Target, "!amiga assign", _Time, _Tags,
           "AmigaOS assigns map logical names such as SYS: or BBS: to paths or devices.").
