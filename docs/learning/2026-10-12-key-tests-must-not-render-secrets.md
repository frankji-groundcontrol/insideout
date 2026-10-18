# Key test failures must never render secret operands

The per-key suite originally used ordinary `unittest` assertions against raw
credential bytes. When a newline fixture failed, the assertion message
included the complete credential, turning a useful test failure into a secret
disclosure.

Shape helpers now assert on booleans and status messages only. A nested
regression test injects a synthetic secret, forces the assertion to fail, and
checks that neither the value nor its `repr` appears in the failure output.
Single-line keys also normalize one editor-added final newline while rejecting
embedded or repeated newlines.

Apply this whenever a test reads credentials, tokens, DSNs, or PEM material:
assert a derived property and write an opaque failure message. Never pass the
raw value as an assertion operand when the test framework may render it.
