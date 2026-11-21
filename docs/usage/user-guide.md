# User guide

From nothing to a signed-in CLI. This page does not set up a server.
Running InsideOut on your own machine is [your own server](installation.md#your-own-server).

## Web

Open the hosted site. Signup is invitation-only. An invitation comes
from the server's administrator. The [README](../../README.md) says how
to ask.

## Install the CLI and the MCP server

macOS or Linux, x86_64 or arm64. Windows is not supported; the web app
works in any browser.

```bash
curl -fsSL https://github.com/frankji-groundcontrol/insideout/releases/latest/download/bootstrap.sh | bash
insideout version
```

The script checks the checksum and, when `gh` is signed in and new
enough, the build provenance. The details are in
[releasing](releasing.md#verifying-a-download).

## Set up

On a terminal, one command asks numbered questions. Under each question, Back and Exit sit side by side, with Forward
beside them. ← goes back, → goes forward, and Q exits a choice. →
does nothing until a number is selected, or until a text field has a
value. While typing, Q is a letter and Ctrl-C exits.
The command checks that the host answers before it asks anything else.
If the host does not answer, it stays there until you fix the address,
go back, or exit.

The first question is the managed service, a service you host, or a
server on this machine. Connecting then asks whether you are logging in
or registering. Login offers a password, an emailed code, or a magic
link. Register asks for an invitation code when the server requires one,
then your email, a display name, and a password. The password is hidden.

The paths are mapped in [setup paths](setup-paths.md). That is the
document we keep up to date.

```bash
insideout setup
insideout whoami
```

The login is saved in a file of mode 0600. The CLI and the MCP server
read it. You do not set environment variables for the hosted service.

Without a terminal, the same door takes flags:

```bash
insideout setup --hosted
insideout login you@example.com
```

No account yet:

```bash
insideout register --invite <invitation-code> you@example.com
```

## MCP

```json
{ "mcpServers": { "insideout": { "command": "insideout-mcp" } } }
```

No token in that snippet. `insideout-mcp` reads the saved login.
Claude Code can also use `claude mcp add insideout -- insideout-mcp`.
More is in [CLI, MCP, and guidance](cli-mcp-and-guidance.md).
