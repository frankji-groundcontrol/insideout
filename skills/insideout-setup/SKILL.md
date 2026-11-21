---
name: insideout-setup
description: Install InsideOut for someone who wants to use the hosted service. Use when the user says install, set up, or get started with InsideOut and did not ask to run the server. Downloads the prebuilt binaries, runs the numbered insideout setup, and registers MCP. Does not install Go, Flutter, or Docker, and does not create a .env.
---

# InsideOut setup — hosted service

You are installing the CLI and the MCP server against the hosted API.
The person does not need a database or environment variables.

If they asked to run InsideOut on their own machine, or to work on the
client, stop and use [insideout-server](../insideout-server/SKILL.md).

Supported platforms: macOS and Ubuntu, x86_64 or arm64. On anything
else, say Windows and other systems are unsupported and point at the
web app.

## Rules

- Never read `.env` or `.env.local`. Never print a token, password, or
  database URL.
- Do not invent an email or a password. The person types those.
- Do not set `INSIDEOUT_API` on this path.
- Do not install Go, Flutter, Docker, or a key store.

## Steps

1. Install the binaries if `insideout version` fails:

```bash
curl -fsSL https://github.com/frankji-groundcontrol/insideout/releases/latest/download/bootstrap.sh | bash
insideout version
```

2. On a terminal, run `insideout setup`. It prints numbered choices.
   Enter accepts the hosted service. Then `insideout whoami`.

   Without a terminal:

```bash
insideout setup --hosted
insideout login you@example.com
```

   The password is read from stdin, not from the command line.

3. Register MCP. No token in the snippet:

```json
{ "mcpServers": { "insideout": { "command": "insideout-mcp" } } }
```

The walk-through for a person is
[the user guide](../../docs/usage/user-guide.md). The wizard paths are
[setup paths](../../docs/usage/setup-paths.md).
