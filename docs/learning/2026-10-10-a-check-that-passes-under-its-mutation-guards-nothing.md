# 2026-10-10 — A check that still passes when you break the thing guards nothing

## Context

The nginx re-resolution check ([BUG-027](../issues/2026-10-10-bug-027-nginx-resolved-the-api-upstream-once.md))
had a last assertion: the second chunk of a streamed response arrives at least a
second after the first, "so it still streams unbuffered". Before trusting it, the
template was mutated with `proxy_buffering on;`. The check still passed.

The first chunk was riding along with the response headers, which nginx forwards
at once even when it buffers. Holding the first chunk back by half a second made
no difference either: nginx 1.27.5 forwards a slow, small stream as it arrives
whether `proxy_buffering` is on or off. The behaviour the check was named for is
not observable with a stream like this.

## Lesson

An assertion earns its place only if some break of the code it names turns it
red. Mutate the code under test once, by hand, for each assertion; when one
survives, do not make the test cleverer until it happens to fail. Find out
whether the behaviour can be observed at all, and if not, rename the check to
what it really proves (here: a chunked response still goes through the variable
`proxy_pass` chunk by chunk) and say in the record what it does not guard.

The same pass found that the other two mutations (no `resolver`, no `Host`
header) were caught, which is what makes the first result believable.

## When to apply

- Any regression check added with a fix, before it is committed as proof.
- A comment or check name that promises more than its assertion measures
  ("unbuffered", "atomic", "no copies").
- Configuration tests, where a directive can be present in the file and still
  have no effect on the thing measured.
