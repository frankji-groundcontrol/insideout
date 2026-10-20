# Wave 4b — integrated locally 2026-10-20

Four ACCEPTED clusters were cherry-picked onto local `main`, in worker order,
with no conflicts. Each landed patch matches its worker patch. The prose
commit was rejected by the verifier because a docs edit has no red test;
the coordinator read that diff and cherry-picked it. Nothing in this wave
is pushed, tagged, or deployed. `origin/main` remains `e6a3d14`.

## What landed

| Cluster | Worker SHAs | Landed SHAs |
| --- | --- | --- |
| `w4b-ci-fetch` | `04bbc8b4253bce21d72287414db85a2346161245`, `5448d02bc2f1a9517e244f3dfd29c0063addba38` | `d91630c998058ab88425cec72b19bd2a7be60099`, `938e9a186d6c5996176586186a5d01133aaa5fd5` |
| `w4b-register-scrub` | `49efd435b9c6687f84a06fccf8a269871753b72b`, `27b4a9ce0567aed138029c56a8dd0f4e822d04bb` | `8310e342c9888311e157da01e7e86c298ff0da6e`, `044ba1c5d499a539cc81691e17d0cde03ef5100b` |
| `w4b-agent-wiring` | `be7ae64249c2db4aedad04cfb913d30e1724e654`, `47b9330dbbe375c7ce5f8bfb81711896692a3485` | `78deb04337abbdf9c6f0e72d784599eba287fcd8`, `0cfe1e7e35e643f4f2604fe0f575e2cf626681eb` |
| `w4b-test-strict` | `f81acbbc4b63c4b4d18b77bd53de855b697ac188`, `a65744d597cc897099d359b6f68e80b735b16eac` | `6c0299e0e6c64ca3898cf5c674c1389d8f79d4c2`, `460c33cfdbd46a70e43d826c94f3e500d00c4e38` |

The code tip after those eight commits is `460c33cfdbd46a70e43d826c94f3e500d00c4e38`.
The prose tip on top is `4e16d1b`.

- `w4b-ci-fetch` sets `fetch-depth: 0` on the env-suites checkout and adds the gate that requires it (`rv4-FR-F45`).
- `w4b-register-scrub` replaces the quoted deploy id in the findings register with `<deploy-id>` and deletes the `QUOTES_EVIDENCE` allowlist (`rv4-PUB-12`).
- `w4b-agent-wiring` passes the conversation's user messages into the `record_fact` evidence check (`rv4-FR-F94` / `rv4-FR-F86`).
- `w4b-test-strict` calls `testlock.RequireLocalURL` before the remaining audited test opens, and `scripts/dev.sh` follows (`rv4-FR-F195`).

## Prose

`w4b-prose` worker commit `f8600ccd4400d3cda6e045222c959551b7f7e7a5` was REJECTED by the verifier: there was no red commit. The coordinator read the five-file diff (deadline exemptions, the strict-test coverage sentence, and the Responses retry) and cherry-picked it as `4e16d1b`. `check_docs_truth.py`, `check_docs_truth_rv4.py`, and `check-redaction.sh` exit 0 on that tip.

## Checks

- `python3 -I scripts/tests/check_ci_gates.py`: exit 0, `ci gates ok: 10 checks`.
- `python3 -I scripts/tests/check_redaction_addresses.py`: exit 0, 16 tests OK.
- `sh scripts/check-redaction.sh`: exit 0.
- `gofmt -l` on the Go files these commits touched: no output.
- `go test -count=1 -run '^TestRvGuardsStrictModeRefusesARemoteDatabase$'` in `server` for `./internal/store/`, `./internal/api/`, and `./cmd/insideout/`: ok (1.141s, 0.487s, 0.646s). Those tests do not need a database.
- `DATABASE_URL` was unset in the parent checkout. It was not loaded from `.env`. The integrator did not re-run `TestRecordFact_WholeFourCharacterAnswerIsStored`. The verifier had already run it on a private Postgres 18 at `127.0.0.1:55442`: red at `be7ae64`, green at `47b9330`.
