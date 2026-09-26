# Golog rule packs

A rule pack is a directory containing one or more `.pl` files. Golog walks pack directories recursively and loads all Prolog files in deterministic lexicographic order.

Use numeric prefixes when ordering matters:

```text
rules/packs/my-pack/
  10-facts.pl
  20-commands.pl
  30-actions.pl
```

Enable one or more files/directories with the comma-separated `GOLOG_RULES` setting:

```sh
GOLOG_RULES=rules/hello.pl,rules/packs/irc-help,rules/packs/amiga
```

All enabled packs share one embedded Prolog interpreter and the same event/action contracts. Packs must not assume BotAI, BotWeb or PBMP is installed. Those integrations remain optional.
