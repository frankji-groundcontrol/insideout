# Deleting an account

A signed-in person deletes their own account with `DELETE /api/v1/me`. The
body is JSON, `{"password":"..."}`, the current password. The route refuses an
agent token. There is no whole-account export, and the privacy notice promises
no response time. From a terminal, `insideout account delete` sends that
request: it reads the current password from the hidden prompt (never from a
flag) and then removes the saved login. MCP has no tool for it.

On the web, the profile page ends with a danger block: Delete my account asks
for the current password, sends this request, signs the browser out and returns
to the landing page.

What goes: that person's coach conversations and messages, run records, ideas
of every status, sessions, one-time tokens and workspace memberships. The
account row stays. Its address, username, password hash and profile text are
overwritten with placeholders that are not the old address, and the hash does
not verify.

What stays: other members' PRDs, revisions, commits and projects, and the same
kinds of shared rows this person wrote. A workspace this person created that
still has another member stays, and one remaining member becomes its creator.
A workspace with no other member is deleted with everything in it.

The notice and the contact placeholder are in [PRIVACY.md](../../PRIVACY.md).
The decision is in
[the retention record](../issues/2026-10-18-rv4-retention-erasure.md).
