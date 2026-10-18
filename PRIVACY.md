# Privacy notice

InsideOut is open-source software (MPL-2.0) and also runs as a hosted,
multi-user service: the deployment whose address ships as `DefaultBase` in
[server/internal/apiclient/default_base.go](server/internal/apiclient/default_base.go),
operated by the repository owner. This notice says what the software stores
about you, which outside services it can pass your data to, and how long it
keeps it. It was written on 2026-10-19 from the code in this repository. It
states facts; it names no legal basis and claims no compliance with a
particular law. If you run your own deployment you are the operator: the
services below are the ones you configure, and you are the contact.

## What is stored

Everything is kept in one PostgreSQL schema (`insideout`) behind row-level
security. The operator runs the database and can technically read all of it.

| Data | What exactly |
| --- | --- |
| Account | E-mail address, username, a salted argon2id hash of your password (never the password), and the avatar address, bio and keywords you choose to add. Also your role, when the address was verified, and created and updated times. |
| Sign-in state | Hashes of refresh tokens and of e-mailed one-time tokens (sign-in link, sign-in code, password reset), each with an expiry. For an invitation: the invited e-mail address, a hash of the code, and who issued and who redeemed it. |
| Workspaces | The name, the members and their roles. |
| Your work | Ideas, projects, project updates, PRDs with their revisions and commits, roadmap nodes and evidence, and proposal decisions: the text you or the coach wrote, each marked with the author's account. |
| Coach conversations | Every message you send and every reply, the coach's tool-call payloads and a rolling summary of the conversation. A run record per message holds your prompt and the reply. Per-call usage telemetry holds the event type, model name, token counts and latency; its columns for request and response text exist, and the current code leaves them empty. |
| Server log | One line per request: method, path without the query string, status, duration, and your account id when you are signed in. No IP address, no request body. |

Held only in server memory, never in the database: while you have a project
open, your display name and pointer position are visible to the other members
viewing it, and are dropped when you leave; and sign-in throttling keeps your
IP address and the e-mail address you typed for the length of the throttle
window. The web server (nginx) and the hosting platform keep their own
standard access logs, which include IP addresses; this repository sets no
retention for them.

## Who can see it inside the service

- Members of a workspace see its projects and PRDs.
- An idea is visible to its author and to the admins of its workspace, not to
  the other members.
- A coach conversation is visible to its author only.
- Deployment administrators can issue and revoke invitations, and see every
  invitation with its invited address, expiry and status.
- Nothing is public. Apart from the health check, the account entry routes
  (sign-in, registration, password reset, sign-in links and codes) and
  GitHub's webhook, every route needs a signed-in session; there is no share
  link for projects, ideas or PRDs.

## Who else receives data

The software passes your data to an outside service only when the operator has
configured that service. The code can reach these:

| Recipient | What it receives | When |
| --- | --- | --- |
| A model provider chosen by the operator (the default endpoint is Anthropic's API; any service that speaks the Anthropic Messages or OpenAI Responses wire format works) | For a coach turn: the conversation so far, your latest message, the PRD title and section text, the recorded facts and the reviewer's findings. For the review pass: the PRD title, section text and recorded facts. For the roadmap planner: the PRD title and section text, or a project title with one roadmap node's title and description. Your e-mail address and username are not added to these requests. | Only when `INSIDEOUT_LLM_API_KEY` is set. Without it the coach runs offline from templates and nothing leaves the server. |
| Resend, a mail provider | The recipient address, subject and body of sign-in links and codes, password-reset and invitation mails. | Only when `INSIDEOUT_MAIL_API_KEY` is set. Without it no mail is sent, and the server logs the recipient and subject (the body too, if the operator opted in for development). |
| GitHub (`api.github.com`) | The repository you bind to a project, and requests made with the operator's GitHub credential; GitHub also calls the server's webhook. What it returns (commits, activity) is stored as project updates. | Only for a project bound to a repository. |
| Google-operated hosts (`fonts.googleapis.com`, `fonts.gstatic.com`, `www.gstatic.com`) | Your IP address, browser details and the page's origin, when the web client loads its fonts and the CanvasKit renderer. | On every page load of the web client. Bundling both with the build is row 3.11 of the [remediation plan](docs/review/2026-10-18-multirole-4/40-remediation-plan.md); until that ships, this holds. |

The variables are described in [docs/usage/environment.md](docs/usage/environment.md).
The client and the server contain no analytics or advertising code. The
server sets two HttpOnly cookies, `insideout_access` and `insideout_refresh`,
only to keep you signed in; the web client also keeps its sign-in tokens and
your language and theme choice in the browser's local storage.

## Retention

Today, as built:

- **Your account and everything you create** are kept until you delete the
  item. There is no automatic purge and no expiry for accounts, ideas,
  projects, PRDs or roadmaps.
- **Coach conversations** (messages and summaries) have no automatic purge and
  no retention window. They go when the PRD or the workspace they belong to is
  deleted.
- **Run records** (your prompt and the reply) and **usage telemetry** are
  deleted 30 days after they were created, by a routine in the server that
  runs every five minutes.
- **Sign-in sessions, one-time tokens and invitations** are deleted 30 days
  after they expire.
- **Logs and backups** follow the hosting platform and the operator's backup
  rotation; database backups hold everything above until rotated.

## Deleting your data

You can edit your profile, delete your ideas, projects and project updates,
leave a workspace, and delete a workspace you created, which deletes
everything in it, conversations included. A single PRD can be exported as
Markdown from its export page.

There is no self-service account deletion and no export of a whole account.
Ask at the contact below; this notice promises neither a route nor a response
time. Which retention windows and which erasure rule to adopt is an open
decision, recorded in
[docs/issues/2026-10-18-rv4-retention-erasure.md](docs/issues/2026-10-18-rv4-retention-erasure.md);
this notice changes when it is made.

## Contact

<!-- Operator: replace <privacy-contact> with a monitored address or a form
URL before announcing a deployment. Never a personal address in this file. -->

Questions and requests about your data: `<privacy-contact>`.

General questions that need no personal data can go to the
[issue tracker](https://github.com/frankji-groundcontrol/insideout/issues),
which is public: do not post personal data there.
