# Golog rule packs

A rule pack is a directory containing one or more `.pl` files. Golog walks pack directories recursively and loads all Prolog files in deterministic lexicographic order.

Use numeric prefixes when ordering matters:

```text
rules/packs/my-pack/
  pack.json
  10-facts.pl
  20-commands.pl
  30-actions.pl
```

`pack.json` is optional for legacy/plain rule directories, but recommended for reusable packs. M1.2 understands:

```json
{
  "id": "my-pack",
  "version": "0.1.0",
  "description": "Example Golog rule pack.",
  "roles": ["example-expert"],
  "commands": ["!example"],
  "capabilities": ["expert.example"],
  "depends": []
}
```

Rules:

- `id` is a stable lowercase pack identifier.
- `version` uses a semver-like `x.y.z` form.
- `roles` describes expert/persona roles exposed by the pack.
- `commands` documents user-facing IRC commands implemented by the pack.
- `capabilities` are stable machine-readable feature identifiers for discovery by Golog and optional integrations.
- `depends` lists pack IDs that must also be loaded.
- duplicate pack IDs, malformed manifests and missing dependencies reject startup/reload.

Enable one or more files/directories with the comma-separated `GOLOG_RULES` setting:

```sh
GOLOG_RULES=rules/hello.pl,rules/packs/irc-help,rules/packs/amiga
```

All enabled packs share one embedded Prolog interpreter and the same event/action contracts. Packs must not assume BotAI, BotWeb or PBMP is installed. Those integrations remain optional.
