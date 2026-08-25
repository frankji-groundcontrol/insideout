-- Invitation-only signup + first admin (qwenimage pattern):
--   users.system_role    — admin gate for invitation + settings management
--   invitations          — admin-minted, email-bound, single-use codes
--                          (sha256 hash at rest, same posture as auth_tokens)
--   settings.signup_mode — invitation_only | open | closed, seeded
--                          fail-closed: an unconfigured deployment must not
--                          drift open.
ALTER TABLE insideout.users ADD COLUMN system_role text NOT NULL DEFAULT 'member'
  CHECK (system_role IN ('member', 'admin'));

CREATE TABLE insideout.invitations (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  email text NOT NULL,
  code_hash text NOT NULL UNIQUE,
  expires_at timestamptz NOT NULL,
  used_at timestamptz,
  used_by uuid REFERENCES insideout.users(id) ON DELETE SET NULL,
  revoked_at timestamptz,
  created_by uuid REFERENCES insideout.users(id) ON DELETE SET NULL,
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX ON insideout.invitations (email);

CREATE TABLE insideout.settings (
  key text PRIMARY KEY,
  value jsonb NOT NULL,
  updated_at timestamptz NOT NULL DEFAULT now()
);

INSERT INTO insideout.settings (key, value) VALUES
  ('signup_mode', '"invitation_only"');
