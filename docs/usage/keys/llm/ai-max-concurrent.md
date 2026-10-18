# ai-max-concurrent

`INSIDEOUT_AI_MAX_CONCURRENT` is the maximum number of provider-backed coach
turns allowed in flight. Blank means the catalog default `4`.

## 1. Where the `.key` file is

`hosts/<host>/keys/llm/.secrets/ai-max-concurrent.key` (manifest
`llm/coach.yaml`).

## 2. How it is propagated

The value moves through `.env` and `dev.sh` into the coach dispatcher. It is a
runtime backpressure setting; the provider itself never sees it.

## 3. How to set it

1. Choose a positive integer that the provider quota and server size can
   sustain, or leave the value blank to use `4`.
2. Run `./scripts/env.sh edit`, choose `INSIDEOUT_AI_MAX_CONCURRENT`, and enter
   the integer (the prompt does not echo it).
3. Initialize/fill the host key store:

   ```bash
   INSIDEOUT_KEYS_HOST='<host>' python3 scripts/keys.py init
   INSIDEOUT_KEYS_HOST='<host>' python3 scripts/keys.py fill
   ```

Zero, negative, and non-numeric values fall back to the default; do not use a
shape-invalid value to signal "unlimited".

## 4. How to test it

Run the shape test, then use a disposable account to issue more simultaneous
coach SSE turns than the configured limit. Keep the server log local and do
not copy transcript or provider data into a report.

```bash
cd scripts && python3 -m unittest discover -s tests/keys -t . -p 'test_ai-max-concurrent.py' -v
```

**Expected result:** no more than the configured number of turns occupy the
provider dispatcher at once; extra turns wait for a slot or receive the normal
queue response.

**If it fails:** run `./scripts/env.sh check server`, confirm the value is a
positive integer, and restart the one server process that owns the database.
A provider 429 is a quota signal, not proof that this setting was applied.

## 5. How it is used

The coach semaphore limits concurrent model calls so one burst of sessions
cannot exhaust the [llm-api-key](llm-api-key.md) quota.

## 6. Installation guide

Follow the [coach setup tutorial](../../key-setup-tutorial.md#4-choose-the-coach-mode),
the [coach testing tutorial](../../key-testing-tutorial.md#5-prove-the-coach-configuration),
and the [own-server installation path](../../installation.md#your-own-server).
