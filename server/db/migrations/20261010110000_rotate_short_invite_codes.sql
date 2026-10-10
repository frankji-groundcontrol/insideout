-- Workspace invite codes were 6-digit numbers (a 10^6 keyspace, and the only
-- credential on the unthrottled join endpoint) until 2026-07-27, when new
-- codes became 128-bit hex. Rows created before that kept their short code.
-- Replace them with the current 32-hex shape; longer codes are untouched, so
-- a second run changes nothing. The updated_at trigger touches the rotated
-- rows, which is accurate: their code did change. Admins read the new code in
-- the workspace; an outstanding short code stops working.
UPDATE insideout.workspaces
SET code = replace(gen_random_uuid()::text, '-', '')
WHERE length(code) < 32;
