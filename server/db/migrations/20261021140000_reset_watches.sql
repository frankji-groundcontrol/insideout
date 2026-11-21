-- A watch id for a password reset. The CLI that asked for the link polls
-- this row. The email token stays in auth_tokens; this id is not that
-- token. A request for an unknown address still gets a row, with no
-- token_hash, so the status never becomes completed and the command
-- does not reveal which addresses are registered. No RLS: the API is
-- the only caller, same posture as insideout.auth_tokens.
CREATE TABLE insideout.reset_watches (
  id uuid PRIMARY KEY,
  token_hash text,
  expires_at timestamptz NOT NULL,
  completed_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX ON insideout.reset_watches (token_hash) WHERE token_hash IS NOT NULL;

DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'insideout_app') THEN
    GRANT SELECT, INSERT, UPDATE, DELETE ON insideout.reset_watches TO insideout_app;
  END IF;
END $$;
