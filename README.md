# agenttohuman (`a2h`)

CLI for **agent-to-human** notifications. An agent sends a message; a human
gets it through a configured method. Methods are a horizontal, extendable
layer. Discord webhook is implemented first.

- Binary: `a2h`
- Skill: `/a2h`
- Local data: `~/.agenttohuman`

## Install

```bash
git clone https://github.com/TyrellD1/agenttohuman.git
cd agenttohuman
npm install
npm run build
npm link
```

Requires Node 18+.

Copy the skill to your personal Cursor skills if you want `/a2h` in every
project:

```bash
mkdir -p ~/.cursor/skills/a2h
cp .cursor/skills/a2h/SKILL.md ~/.cursor/skills/a2h/SKILL.md
```

## Quick start

1. Create a Discord channel webhook (Channel Settings → Integrations → Webhooks).
2. Save it as the default method:

```bash
a2h init discord-webhook
```

Or non-interactive:

```bash
a2h init discord-webhook --webhook-url "https://discord.com/api/webhooks/..."
```

3. Send:

```bash
a2h send "The long job finished."
```

Agents should prefer `--json` and the `/a2h` skill:

```bash
a2h default --json
a2h info --json
a2h send --json "The long job finished."
```

## Commands

| Command | Purpose |
| --- | --- |
| `a2h send <message>` | Send via the default method (`-m` to override) |
| `a2h init [method]` | Save credentials for a method |
| `a2h methods` | List methods |
| `a2h info [method]` | Limits and payload shape |
| `a2h default [method]` | Get or set the default method |
| `a2h status` | Config path, default, configured methods |

Global flags: `--json`, `-h` / `--help`, `-V` / `--version`.

## Config

Stored at `~/.agenttohuman/config.json` (override with `AGENTTOHUMAN_HOME`).
The directory is created `0700` and the file `0600`. Do not commit it.

On first init, that method becomes the default. Later `a2h send` uses the
default unless `--method` is passed.

## Adding a method

See `AGENTS.md`. Register a `MethodDefinition` in `src/methods/` and keep
`a2h info` metadata accurate (character limits, payload shape).

## License

MIT
