# Setup paths

The map of `insideout setup` on a terminal, with no flags. This is the
document we maintain. When a path changes, change it here first.

Each question has a step number. The wizard prints that number on the question line, in bold, with a
dim dot: `1.1.2 · Email`. Say the number when you walk a path, for
example "1.1.4". The blocks below are the screen for that step, without
the button row. The developer key files are not part of this map. The
flag forms (`--hosted`, `--local`, `--check`) are the same work without
a terminal, and they are listed in [installation](installation.md).

## Buttons

These sit under every numbered question. They are not steps.

```text
[ ← Back ]    [ Q Exit ]    [ → Forward ]
```

| Key | Action |
| --- | --- |
| A number | Selects that row. It does not advance. |
| → or Enter | Forward. Dim until a number is selected, or until a text field has a value. |
| ← | Back to the previous step. On step 1, the wizard stays on step 1. |
| Q | Exit, on a choice. |
| Ctrl-C | Exit, while typing. Q itself is a letter in a text field. |

A text field that has a default starts with that value filled in, so →
accepts it. Clearing the field blocks →.

## 1. What are you setting up?

```text
InsideOut — interactive setup

1 · What are you setting up?
  1) Remote default: the managed service
  2) Connect to a service you host
  3) Run a server on this machine
```

Nothing is selected until a number is pressed. The row you select
chooses the branch: row 1 continues at 1.1, row 2 at 1.2, row 3 at 1.3.

### 1.1 Remote default

#### 1.1.1 Check the managed host

The wizard runs a health check. It does not ask for an address. Reaching
the host is not enough: `/healthz` must report ok, which means the
database answered, and the API must answer.

```text
1.1.1 · Checking the default service
  ✓ address  default
  ✓ health   /healthz ok
  ✓ api      authentication required
Healthy.
```

A failure stays on 1.1.1 and names the check that failed:

```text
  ✗ health   /healthz answered 503: database unreachable
Not healthy. Fix this before the next step.

1.1.1 · The service is not healthy.
  1) Try again
```

→ on Try again runs the health check again. ← returns to step 1.
Step 1.1.2 is next, and only after `Healthy.`

#### 1.1.2 Do you already have an account?

```text
1.1.2 · Do you already have an account?
  1) Log in
  2) Register
```

Row 1 continues at 1.1.3. Row 2 continues at 1.1.6. ← returns to 1.1.1.

#### 1.1.3 How do you want to sign in?

Only after 1.1.2 row 1.

```text
1.1.3 · How do you want to sign in?
  1) Password
  2) Emailed login code
  3) Magic link
  4) Forgot password
```

← returns to 1.1.2. Row 4 continues at 1.1.3.1.

##### 1.1.3.1 Email

The address the reset link should go to. The wizard then asks you to
confirm that address before anything is sent, and prints the email
status: accepted for delivery, or logged on the server and not delivered
(no mail provider is configured). ← returns to 1.1.3.

`insideout forgot-password <email>` is the same request outside the
wizard. It keeps the watch id and waits. When the link is used in the
browser, the command notices, drops the saved login, and tells you to
sign in again. It does not learn the new password. Ctrl-C stops the
wait; `insideout reset-password` still finishes a link you paste.

##### 1.1.3.2 Reset link

The email shows the reset code on its own line, and the same code is
inside the link. Paste either one. Ctrl-C exits this field. The code
works once, for 60 minutes. Using it here or on the web page uses it up.
← returns to 1.1.3.1.

##### 1.1.3.3 New password

Hidden, at least 8 characters. When it is accepted, the wizard signs in
with it and continues at 1.1.7. Every other session for that account
is signed out. ← returns to 1.1.3.2.

#### 1.1.4 Email

```text
1.1.4 · Email
```

For an emailed code or a magic link, the next question is confirmation:
`Send email to <address>?` Yes sends it and prints the email status.
No sends nothing and stays on 1.1.4. A password sign-in skips that
question. ← returns to 1.1.3.

#### 1.1.5 Secret

The prompt matches the row chosen at 1.1.3. The password is hidden.
The code and the magic link are sent after 1.1.4.

```text
1.1.5 · Password
```

or `Login code (check your email)`, or `Magic link (paste the link from your email)`.

A wrong secret stays on 1.1.4. ← returns to 1.1.4.
Success continues at 1.1.7.

#### 1.1.6 Register

Only after 1.1.2 row 2. The server's signup mode decides which fields
appear.

| Signup | What comes next |
| --- | --- |
| Closed | Says to ask the server's administrator, and returns to 1.1.2. |
| Invitation-only | 1.1.6.1, then 1.1.6.2. |
| Open | Skips 1.1.6.1 and starts at 1.1.6.2. |

##### 1.1.6.1 Invitation code

```text
1.1.6.1 · Invitation code
```

← returns to 1.1.2.

##### 1.1.6.2 Email

```text
1.1.6.2 · Email
```

← returns to 1.1.6.1, or to 1.1.2 when signup is open.

##### 1.1.6.3 Display name

```text
1.1.6.3 · Display name
```

Blank uses the part of the email before @. ← returns to 1.1.6.2.

##### 1.1.6.4 Password

```text
1.1.6.4 · Password
```

Hidden. A rejected registration stays here. ← returns to 1.1.6.3.
Success continues at 1.1.7.

#### 1.1.7 MCP snippet

After a successful sign-in from 1.1.5 or 1.1.6.

```text
1.1.7 · Show the MCP registration snippet?
  1) Yes
  2) No
```

Yes prints the MCP JSON. The token is not printed. The login is saved
in a file of mode 0600. Then: `Verify: insideout whoami`.

### 1.2 Connect to a service you host

#### 1.2.1 API base URL

```text
1.2.1 · API base URL
  default
```

The field starts as the word `default`. → keeps the default service and
does not print its address. Clear it and type an address to point at a
server you host. ← returns to step 1.

#### 1.2.2 Check that host

```text
1.2.2 · Checking the default service
  ✓ address  default
  ✓ health   /healthz ok
  ✓ api      authentication required
Healthy.
```

An address you typed is shown as you typed it. The default is not.

A failure stays on 1.2.1. Type a new address and →, or ← to step 1.
`Healthy.` continues at 1.1.2, and from there the steps are 1.1.2
through 1.1.7. ← from 1.1.2 returns to 1.2.1 on this branch.

### 1.3 Run a server on this machine

Needs a checkout of this repository.

#### 1.3.1 PostgreSQL

```text
1.3.1 · PostgreSQL for this server
  1) Bundled Postgres in Docker (127.0.0.1:5442)
  2) A Postgres URL I already have
```

Row 1 skips 1.3.2 and goes to 1.3.3. Row 2 continues at 1.3.2.
← returns to step 1.

#### 1.3.2 Database URL

Only after 1.3.1 row 2. The value is hidden. ← returns to 1.3.1.
Ctrl-C exits. An empty value does not move forward.

```text
1.3.2 · Database URL
  •••
```

#### 1.3.3 Check the database

```text
1.3.3 · Checking the database …
Reachable.
```

A failure returns to 1.3.1. Fix the URL or Docker, then try again.

#### 1.3.4 Start the API

```text
1.3.4 · Start the API now?
  1) Yes
  2) No
```

← returns to 1.3.1. No prints the start command. Yes starts the API in
the foreground. This path does not sign you in. After the API is up, set
`INSIDEOUT_API=http://127.0.0.1:8080/api/v1` and run `insideout setup`
again, taking 1.2 with that address.

A fresh database is invitation-only. On a disposable local database,
`insideout seed` then promoting `demo@insideout.local` is the short
sign-in. `seed` must not run against a shared database.
