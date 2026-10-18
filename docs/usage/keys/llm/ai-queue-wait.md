# ai-queue-wait

`INSIDEOUT_AI_QUEUE_WAIT` is the maximum time a coach turn waits for a
concurrency slot. It is a Go duration; blank means the catalog default `15s`.

## 1. Where the `.key` file is

`hosts/<host>/keys/llm/.secrets/ai-queue-wait.key` (manifest
`llm/coach.yaml`).

## 2. How it is propagated

The value moves through `.env` and `dev.sh` into coach admission. It controls
local queueing and is not sent to the model provider.

## 3. How to set it

1. Choose a Go duration such as `15s` or `30s`, or leave it blank for `15s`.
2. Run `./scripts/env.sh edit`, choose `INSIDEOUT_AI_QUEUE_WAIT`, and enter the
duration at the hidden prompt.
3. Initialize/fill the host key store:

   ```bash
   INSIDEOUT_KEYS_HOST='<host>' python3 scripts/keys.py init
   INSIDEOUT_KEYS_HOST='<host>' python3 scripts/keys.py fill
   ```

Garbage values silently use the default, so verify the observed behavior after
changing this knob.

## 4. How to test it

Run the shape test, then set a disposable test server to one concurrency slot
and issue a second coach turn while the first is held open. Use a short queue
wait so the second request reaches the admission decision quickly.

```bash
cd scripts && python3 -m unittest discover -s tests/keys -t . -p 'test_ai-queue-wait.py' -v
```

**Expected result:** a turn that cannot obtain a slot before the configured
wait ends receives the normal 503 retry response; a free slot admits normally.

**If it fails:** check the duration syntax with `./scripts/env.sh check server`,
restart the server after editing, and confirm the first turn really holds a
slot. A provider error after admission is a separate key or endpoint issue.

## 5. How it is used

Together with [ai-max-concurrent](ai-max-concurrent.md), this setting bounds
local backpressure before a provider call is attempted.

## 6. Installation guide

Follow the [coach setup tutorial](../../key-setup-tutorial.md#4-choose-the-coach-mode),
the [coach testing tutorial](../../key-testing-tutorial.md#5-prove-the-coach-configuration),
and the [own-server installation path](../../installation.md#your-own-server).
