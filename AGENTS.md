# Agent notes

## Git workflow (required)

- Commit often. Make a small, focused commit as soon as a logical slice is working (scaffold, one method, CLI command, skill, tests).
- Push at the end of every turn. Before you finish a response, `git push` to `origin` so the remote matches local work.
- Never commit secrets: webhook URLs, tokens, `.env` files, or anything under `~/.agenttohuman`.
- Do not skip hooks, force-push to `main`, or rewrite published history unless the user explicitly asks.

## What this is

`a2h` (agent-to-human) is a CLI an agent uses to message a human. Delivery methods are a horizontal, extendable layer. Discord webhook is the first implemented method.

- CLI binary: `a2h` (Go)
- Agent skill: `/a2h` (`.cursor/skills/a2h/SKILL.md`)
- Local data (webhooks, default method): `~/.agenttohuman`
- Install script: `install.sh` (GitHub Releases)
- Release targets: `darwin/arm64`, `linux/amd64`, `linux/arm64`

## Adding a method

1. Add a type under `internal/methods/` that implements `Method` (`ID`, `Name`, `Description`, `Metadata`, `Configured`, `Init`, `Send`).
2. Register it in `internal/methods/registry.go`.
3. Put limits and payload shape in `Metadata` so `a2h info` stays accurate for agents.
4. Store credentials only under `~/.agenttohuman`, never in the repo.
5. Cover validation and config updates with a test.

## Conventions

- Go 1.22+, standard library only.
- Config directory is overridable with `AGENTTOHUMAN_HOME` (used by tests).
- Prefer `--json` on CLI commands that agents will parse.
- Embed version at release time via ldflags on `internal/version.Version`.
