# 2026-10-10 — A fake copies the page's misreading of the contract

## Context

The PRD page's **Build MVP** button called `POST /prds/{id}/build` and
navigated to an empty roadmap, because the endpoint had become draft-only and
the write lives on `/build/confirm`
([BUG-022](../issues/2026-10-10-bug-022-build-mvp-wrote-nothing.md)). The
page test passed throughout: its fake answered `/build` with a 409
`replace_conflict` on the first call and a success on the retry, which is the
page author's picture of the contract, written once and used on both sides of
the test. The client had no request builder for `/build/confirm` at all, and
no test was able to notice.

## Lesson

A hand-written fake of an endpoint is only as right as the reading of the
contract it was written from. When the page and its fake share one reader,
the test checks the reader's consistency, not the server's behavior.

- Derive the fake from the handler, not from the page: read the route, the
  response shape and the error codes in `server/internal/api`, and write down
  which verb owns the write.
- A flow whose job is to write needs one live call before it is called done:
  run it against the real server and look at the rows (or the screen the user
  lands on). Here, one tap in a browser would have shown the empty roadmap.
- When a server verb is split or retargeted, grep the clients for the old verb
  in the same change. The CLI and the MCP tools were updated in the split; the
  Flutter client was not, and nothing connected the two.

## When to apply

- Any client whose test fake invents a response shape or an error code.
- Any server change that moves a side effect from one endpoint to another.
- Same family as [BUG-015](../issues/2026-09-03-bug-015-signup-mode-wrong-method.md),
  where `httptest` pinned the client's consistency and the real API answered 405.
