# agenttohuman (`a2h`)

CLI for **agent-to-human** notifications. An agent sends a message; a human
gets it through a configured method. Methods are a horizontal, extendable
layer. Discord webhook is implemented first.

- Binary: `a2h` (Go)
- Skill: `/a2h`
- Local data: `~/.agenttohuman`
- Prebuilt targets: macOS arm64, Linux amd64, Linux arm64

## Install

```bash
curl -fsSL https://raw.githubusercontent.com/TyrellD1/agenttohuman/main/install.sh | bash
```

That downloads the latest GitHub Release into `~/.local/bin/a2h`. Override the
directory with `A2H_INSTALL_DIR`.

From source (needs Go 1.22+):

```bash
git clone https://github.com/TyrellD1/agenttohuman.git
cd agenttohuman
go install ./cmd/a2h
```

Copy the skill to your personal Cursor skills if you want `/a2h` in every
project:

```bash
mkdir -p ~/.cursor/skills/a2h
curl -fsSL https://raw.githubusercontent.com/TyrellD1/agenttohuman/main/.cursor/skills/a2h/SKILL.md \
  -o ~/.cursor/skills/a2h/SKILL.md
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
# with images shown inline (png, jpeg, gif, webp, avif; repeatable):
a2h send "New dashboard vs old" --image ./after.png --image ./before.png
# with any other files (downloads in Discord):
a2h send "Build files" --file ./report.html --file ./notes.md
```

Images and files share one limit: 10 per message, 20 MiB each. Text is
optional when you attach something. Several images show as a gallery under
the message. `--image` rejects formats Discord would only offer as a download
(HEIC, BMP, TIFF, SVG); convert them first, or send them with `--file`.

```bash
# convert an iPhone photo on macOS
sips -s format png IMG_0001.HEIC --out IMG_0001.png
```

Agents should prefer `--json` and the `/a2h` skill:

```bash
a2h default --json
a2h info --json
a2h send --json "The long job finished."
```

## Update

```bash
a2h update --check
a2h update
```

`a2h update` replaces the current `a2h` binary with the latest GitHub Release
after verifying `SHA256SUMS`. It only works for the prebuilt platforms above,
and only if this process is an `a2h` binary (not a wrapper script).

## Commands

| Command | Purpose |
| --- | --- |
| `a2h send <message> [--image <path> ...] [--file <path> ...]` | Send via the default method (`-m` to override; `-i`/`--image` and `-f`/`--file` repeatable) |
| `a2h init [method]` | Save credentials for a method |
| `a2h methods` | List methods |
| `a2h info [method]` | Limits and payload shape |
| `a2h default [method]` | Get or set the default method |
| `a2h status` | Config path, default, configured methods |
| `a2h update` | Install the latest GitHub Release over this binary |
| `a2h version` | Print the version |

Global flags: `--json`, `-h` / `--help`, `-V` / `--version`.

## Config

Stored at `~/.agenttohuman/config.json` (override with `AGENTTOHUMAN_HOME`).
The directory is created `0700` and the file `0600`. Do not commit it.

On first init, that method becomes the default. Later `a2h send` uses the
default unless `--method` is passed.

## Adding a method

See `AGENTS.md`. Implement `Method` in `internal/methods/` and keep `a2h info`
metadata accurate (character limits, payload shape).

## License

MIT
