# Verifying a tty-only tool, and guards that fail closed

Derived from the session that built `scripts/env.sh edit` and `propagate`. The
through-line: **a check that cannot fail in the direction of the bug is not a
check.** Ask of any guard — *what would this look like if the thing it guards
were broken?* If the answer is "the same", the guard proves nothing.

Closed failures — the system says no, or says nothing, instead of saying
"broken" — are the right default for security, and exactly why they need a
different kind of verification.

## 1. Split logic from rendering, then test the logic

curses needs a tty, so a TUI cannot be exercised in a test run. Everything the
screen *decides* therefore lives in [`scripts/env_catalog.py`](../../scripts/env_catalog.py)
and [`env_tui.py`](../../scripts/env_tui.py) only draws. `env.sh edit --list`
renders the same rows without a terminal, and that is the surface
[`test_env_catalog.py`](../../scripts/test_env_catalog.py) and
[`test_env_writes.py`](../../scripts/test_env_writes.py) assert on.

This is necessary, not sufficient. In the reference implementation this split
produced 29 green assertions alongside a UI that crashed the first time Enter
was pressed: writing a full-width line at row `h-1` fills the bottom-right
cell, which curses rejects with `addnwstr() returned ERR`. No test could see
it, because the crash lived in the layer tests cannot reach.

## 2. Drive the real binary in tmux, not a raw pty

`tmux capture-pane -p` returns the *rendered cell grid*. A pty gives you the
escape stream, and naive escape-stripping reports phantom misalignment, because
curses redraws incrementally with cursor jumps.

```bash
tmux new-session -d -s envprobe -x 120 -y 30 -c "$PWD"
tmux send-keys -t envprobe:0 './scripts/env.sh edit' Enter; sleep 3
tmux capture-pane -p -t envprobe:0            # the actual screen
tmux send-keys -t envprobe:0 Enter            # the keystroke that crashed the reference
```

What this pass actually confirmed, none of which a unit test could see: the
prompt opens at `h-1` without an `ERR`; a typed secret echoes as `******` and
the raw value never appears in the cell grid; `Escape` cancels in well under a
second (curses' `ESCDELAY` defaults to 1000 ms — `curses.set_escdelay(25)`);
the header keeps counting the *whole* environment while `/` filters the list
(summarising the filtered view reports `0 yours`, which reads as data loss);
and after a write the cursor follows the row to its new sorted position instead
of landing on an unrelated variable.

Use `PPage`/`Down` for deterministic navigation in a probe — counting `Down`
presses from an assumed cursor position is how a TUI navigation loop overran
its target and clobbered a real value in the reference session.

## 3. Exercise every guard against a failing input

Three guards in the reference were each wrong in a way that reading them did
not reveal. Two more nearly shipped here:

| Guard | Looked right | Actually did |
| --- | --- | --- |
| `git check-ignore "$dir/.env"` from the root | refuses to write secrets to a tracked path | consults the wrong `.gitignore` once a component is its own repository. Run `( cd "$dir" && git check-ignore -q .env )`. |
| `n=$(grep -c x f \|\| echo 0)` | counts matches, defaulting to 0 | `grep -c` prints `0` **and** exits 1, so `\|\| echo` appends a second line and the numeric compare breaks |
| `[[ "$v" == *"<"* ]]` | catches `<your-value-here>` | rejects legitimate values containing `<`. Match named markers only. |
| "every declared variable has a consumer", by word-boundary grep | catches a variable that outlived its consumer | **passed a deliberately planted phantom**, because Go's SQL contains `FOR KEY SHARE`. Match the *consumption syntax* (`"NAME"`, `process.env.NAME`, `${NAME…}`), not the bare token. |
| a doc comment illustrating the schema convention | explains the format | `NAME` + `=` inside a comment parses as a declaration, inventing a variable in the catalog. State such rules in prose. |
| `( set -a; . "$f" ) ; declare -p NAME` | reports what the file sets | reports the **ambient** environment too — `declare -p` cannot tell a file assignment from an inherited export, so the gate passed on a zero-byte file whenever the operator's shell had the variable. Clear the names first, and emit an explicit `unset` for each one the file omits. |
| a checksum of the SOURCE stamped into a generated copy | proves the copy is current | proves only that the *source* has not moved. A copy edited in place still matched. A second checksum over the copy's own payload is what detects tampering. |
| `if placeholder or default: return value` in a masker | skeleton values are public | `placeholder` is a *substring* test on the live value, so a real credential containing `change_me` was rendered in the clear. **A content test must never override a name test.** |

Method: for each guard, run it once against a passing input and once against a
failing one, and assert the exit code both times. In this pass that meant four
deliberately corrupted `.env.example` variants (phantom name, dead variable,
optional-marked-required, inline comment after a value), nine `check` inputs
(empty, 31-char secret, 32-char secret, placeholder DSN, bundled-pg with and
without the password, mismatched password, invalid duration, missing file), and
the propagate refusals (hand-written target, unknown component, unignored path).

Everything below the third row of that table was caught *only* by a later
adversarial review that ran the failing input. All of it had looked correct and
all of it was green.

## 3a. A writer must be able to read back what it wrote

Two defects of the same shape survived the first verification pass because
every test used a *tame* value:

- both writers emitted bare `KEY=value`, so a value containing a space produced
  a file the tool's own loader then rejected — and a space is legal in a
  Postgres password, which the prompt happily accepts;
- both resolved a duplicated key to the **first** occurrence while bash,
  compose and c12 all take the **last**, so editing a key that had been
  overridden at the bottom of the file wrote a value nothing used and reported
  it saved.

**Method:** round-trip an awkward value set through writer → reader — space,
single quote, double quote, `$`, backtick, trailing space, `=`, and a DSN full
of `/:@&` — and assert equality, not absence of error. Where two
implementations write the same format (here bash and Python), diff their output
for the same inputs rather than testing each alone.

## 3b. Error paths leak what success paths mask

`check` masked every value it printed — and then bash's own diagnostic, on the
path taken when `.env` is malformed, echoed the offending **line** to the
terminal. The one command you run when the file is broken was the one that
printed its contents.

**Method:** run the leak grep against the *failure* modes too, not just the
happy path. Plant a distinctive token, break the file, and grep every stream —
stdout and stderr — for it. The same pass found the whole DSN being handed to
`pg_isready` as an argv element, where `/proc/<pid>/cmdline` makes it readable
by any local process.

## 3c. Optional terminal capabilities are not optional to guard

`curs_set` and `use_default_colors` raise on `TERM=vt100/vt220/ansi`, and
`initscr` never returns on `TERM=dumb`. The tool promised in a docstring that a
limited terminal would "degrade instead of abort" while three unguarded calls
did the opposite.

**Method:** drive the binary under each degenerate `TERM` in a pty and assert
the process *exits*, with a message naming the non-interactive alternative.
Also drive at the minimum viable size (40×14 here): `curses`' `n` argument
counts characters, not columns, so bilingual text silently overruns a clamp
that looks correct.

## 4. Stamp provenance wherever a later layer can override an earlier one

`app/.env` and the exported root `.env` are two layers over the same keys, and
which one wins depends on the launch path. A generated copy therefore carries
the root file's checksum, `check <component>` compares it, and `dev.sh` gates
every launch on that check — so drift *blocks startup and names the fix*
instead of being invisible. Generalisation: **when a later layer can override
an earlier one, stamp the provenance and verify it**, or the override is
undetectable by construction.

Determine the direction empirically rather than assuming it. Here the loader's
actual rule (c12 assigns a dotenv key only when `process.env` has none) is the
opposite of the reference implementation's, which changes the *story* of the
failure — a bistable file rather than a silent later-wins override — while
leaving the fix identical.

## 5. Back up anything the test writes

Two real values were clobbered while testing the reference, recoverable only
because the file had been copied first.

- `cp .env /tmp/env.bak` before running any tool that writes it, and diff it
  back afterwards (`cmp -s` — assert byte-identical, do not eyeball).
- Move, never delete, a file a tool refuses to overwrite.
- Restore through the system's own mechanism where one exists.

## 6. Then have someone hostile try to break it

Everything above was done, and green, before an adversarial review of the same
work raised 30 claims across four independent lenses (guards, leaks, catalog
logic, docs), each verified by a second pass whose instruction was to *refute*
it and which had to paste a reproduction. 28 survived, including both gate
failures and all three leaks described here.

Two things made that pass productive rather than noisy, and are worth copying:

- **Lenses, not repetition.** Four reviewers each blind to the others' angle
  found four disjoint defect classes. Four reviewers asked the same question
  would have found one.
- **Verification that must reproduce.** "Default to refuted unless you ran it"
  killed six plausible-sounding claims and forced evidence for the rest — and
  the corrections the verifiers attached (which half of a claim was real, which
  was covered by design) were as useful as the findings.

Give reviewers the *design intent* to assert against, and forbid repo writes:
one still ran a command that regenerated a real artifact, which briefly made an
unrelated check flap. Point them at a temp copy.

## Checklist

- [ ] Does a failing case make this check fail? Run one, assert the exit code.
- [ ] Is the tty-only layer separated from the logic, and driven once in tmux?
- [ ] Can a later layer silently override an earlier one? Stamp and verify — the
      source *and* the copy.
- [ ] Does a name-based decision stay a name-based decision, or does some later
      branch re-inspect the value?
- [ ] Can the writer read back what it writes? Round-trip an awkward value set.
- [ ] Does the *error* path leak what the success path masks?
- [ ] Is there an end-to-end probe for each credential, not just a presence check?
- [ ] Backed up anything the test writes; verified the restore byte-for-byte.
- [ ] Has anyone hostile, with the design intent in hand, tried to break it?

## Related

- [SETENV.md](../SETENV.md) — the operating manual these guards protect.
- [Verify against a real database](2026-07-21-verify-against-real-database.md)
  — the same argument for DB-dependent work.
- [Live-exercise streaming endpoints](2026-07-21-live-exercise-streaming-endpoints.md)
  — the same argument for SSE.
