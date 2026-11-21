---
name: insideout-server
description: Run InsideOut's server on this machine, or work on the Flutter client. Use only when the user asked to self-host, run locally, or develop the app. Uses insideout setup --local. Does not describe the developer key files.
---

# InsideOut — server on this machine

Open this only after the person asked to run the server or develop the
client. Someone who wants the hosted service uses
[insideout-setup](../insideout-setup/SKILL.md).

## Rules

- Never read `.env` or `.env.local`. Never print a token, password, or
  database URL.
- Do not invent an email or a password.
- A fresh database is invitation-only. `insideout seed` is only for a
  disposable local database.

## Server

From a checkout of the repository, on a terminal:

```bash
insideout setup
```

Choice 3 is the server. It then asks for bundled Docker Postgres or a
URL the person already has.

Without a terminal:

```bash
insideout setup --local --postgres bundled
# or, with the URL on stdin and never echoed:
insideout setup --local --postgres own
```

A fresh URL needs the two roles. Add `--create-roles` when the connecting
role may create them. Otherwise run the SQL the command prints.

Then point the CLI at this server and sign in:

```bash
export INSIDEOUT_API=http://127.0.0.1:8080/api/v1
insideout whoami
```

The full reference is
[installation — your own server](../../docs/usage/installation.md#your-own-server).
The wizard paths are
[setup paths](../../docs/usage/setup-paths.md).
Day-to-day commands are
[local development](../../docs/usage/local-development.md).

## Client

After the server is up:

```bash
cd client && flutter run -d chrome --web-port=5173 --dart-define=API_BASE=http://127.0.0.1:8080/api/v1
```

Keep port 5173. Invite and magic-link URLs use it.
