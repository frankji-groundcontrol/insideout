# ai-turn-deadline

`INSIDEOUT_AI_TURN_DEADLINE` is the wall-clock bound on one whole coach turn.
It is a Go duration; blank means the default `12m`.

## 1. Where the `.key` file is

`hosts/<host>/keys/llm/.secrets/ai-turn-deadline.key` (manifest
`llm/coach.yaml`).

## 2. How it is propagated

The value moves through `.env` and `dev.sh` into the coach
(`server/internal/agent/coach.go`). `docker-compose.yml` passes it to the
server, empty when `.env` leaves it unset. It is not sent to the model
provider.

## 3. How to set it

1. Choose a Go duration such as `12m` or `20m`, or leave it blank for `12m`.
2. Run `./scripts/env.sh edit`, choose `INSIDEOUT_AI_TURN_DEADLINE`, and enter
   the duration at the hidden prompt.
3. Initialize/fill the host key store:

   ```bash
   INSIDEOUT_KEYS_HOST='<host>' python3 scripts/keys.py init
   INSIDEOUT_KEYS_HOST='<host>' python3 scripts/keys.py fill
   ```

An unparsable or non-positive value silently keeps the default, so verify the
observed behavior after changing this knob.

## 4. How to test it

Run the shape test, then point a disposable test server at an upstream that
streams slowly and never finishes, with a short deadline such as `30s`.

```bash
cd scripts && python3 -m unittest discover -s tests/keys -t . -p 'test_ai-turn-deadline.py' -v
```

**Expected result:** the turn ends with an error once the deadline passes, and
the concurrency slot and the conversation lock are released for the next turn.

**If it fails:** check the duration syntax, restart the server after editing,
and confirm the upstream really keeps the turn open past the deadline.

## 5. How it is used

Together with [ai-max-concurrent](./ai-max-concurrent.md) and
[ai-queue-wait](./ai-queue-wait.md), this setting stops a trickling upstream
from holding a slot and the conversation lock indefinitely.

## 6. Installation guide

Follow the [coach setup tutorial](../../key-setup-tutorial.md#4-choose-the-coach-mode),
the [coach testing tutorial](../../key-testing-tutorial.md#5-prove-the-coach-configuration),
and the [own-server installation path](../../installation.md#your-own-server).
The variable's row is in the [environment reference](../../environment.md#ai-provider).
