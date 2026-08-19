---
name: a2h
description: >-
  Notify a human through the a2h CLI (agent-to-human). Use when the user
  invokes /a2h, asks to be messaged when work finishes, wants a Discord
  ping, or otherwise needs an out-of-band human notification.
disable-model-invocation: true
---

# a2h

Message the human with the `a2h` CLI. Do not POST to Discord or any other
provider yourself. Methods have different payload shapes; let the CLI
discriminate.

## Workflow

1. Confirm `a2h` is on `PATH`. If it is missing, tell the human to install
   it from this repo (`npm install && npm run build && npm link`).
2. Resolve the default method:
   `a2h default --json`
3. Read that method's limits and payload shape:
   `a2h info --json`
   If they passed a method name, use `a2h info <method> --json`.
4. Send when the requested event happens (task done, blocked, needs a
   decision). Do not send a preview, and do not send internal chain-of-thought.
   `a2h send --json "<message>"`
5. If send fails because the message is over the content limit, split into
   multiple `a2h send` calls at or under that limit.

## Configuration

Credentials live in `~/.agenttohuman` on the human's machine. If a method is
not configured, ask them to run:

```bash
a2h init discord-webhook
```

Non-interactive:

```bash
a2h init discord-webhook --webhook-url "$DISCORD_WEBHOOK_URL"
```

Do not invent webhook URLs. Do not write secrets into the repo.

## Message style

Write as a human-facing Discord message: short, specific, useful. Say what
finished or what you need. Include a path, PR, or error only when it helps
the human act.
