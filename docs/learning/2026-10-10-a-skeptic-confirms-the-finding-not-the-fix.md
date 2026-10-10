# A skeptic confirms the finding, not the fix — reproduce the remedy before planning it

**Learned:** 2026-10-10, multirole review 3.

A verification pass — adversarial skeptic or coordinator reproduction —
validates that the defect is real at the quoted line. It does not validate
the *remedy* the finder proposed, and a confirmed finding with a wrong fix
direction is more dangerous than an unverified one, because the verdict's
 credibility rides along with the fix into the remediation plan.

## Evidence

Review 3's ARCH-01 was CONFIRMED by its skeptic and by the coordinator:
`server/go.mod` demands Go 1.26.0 while `server/Dockerfile` pins
`golang:1.25-alpine`, so every documented image build fails (the in-repo
BUG-005 signature). The reviewer's primary fix, also quoted in the register,
was to "lower the go directive back to 1.25.0 — the highest GoVersion any
dependency declares is 1.25.0". The coordinator's reproduction ran the same
`go list -m -json all` and got a different answer:
`golang.org/x/sys` and `golang.org/x/term` already declare `go 1.26.0`,
so the directive cannot go down without also downgrading those modules. The
remediation plan therefore fixes the Dockerfile base (`golang:1.26-alpine`)
instead — the opposite branch of the reviewer's either/or.
([Register with the correction](../review/2026-10-10-multirole-3/01-findings-register.md).)

## Scope

Applies whenever a review finding (human or agent) ships a proposed fix and
someone plans work from it: skeptics refute *evidence*, dependency claims rot
between the finder's run and yours, and "confirmed" attaches to the defect,
never to the diff.

## When to apply again

Before a remediation plan names a fix direction as settled: re-run the one
command that discriminates between the proposed remedies (`go list -m -json
all`, the version pin grep, the actual image build) and record which branch
survives. If the branch flips, say so in the register — a corrected fix is a
finding about the review process, and hiding it quietly imports the wrong
diff into every downstream wave.
