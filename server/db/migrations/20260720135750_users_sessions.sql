CREATE TABLE insideout.users (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  email text NOT NULL UNIQUE,
  password_hash text NOT NULL,
  username text NOT NULL,
  avatar_url text,
  bio text NOT NULL DEFAULT '',
  keywords text[] NOT NULL DEFAULT '{}',
  role text NOT NULL DEFAULT 'user' CHECK (role IN ('user', 'admin')),
  email_verified_at timestamptz,
  meta jsonb NOT NULL DEFAULT '{}',
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TRIGGER set_updated_at
  BEFORE UPDATE ON insideout.users
  FOR EACH ROW EXECUTE FUNCTION insideout.set_updated_at();

-- Refresh-token store. token_hash is sha256 of the opaque refresh token;
-- rotated on every use (old row gets revoked_at), never reused.
-- 刷新令牌存储。token_hash 是刷新令牌的 sha256；每次使用即轮换（旧行写 revoked_at），绝不重用。
CREATE TABLE insideout.sessions (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id uuid NOT NULL REFERENCES insideout.users(id) ON DELETE CASCADE,
  token_hash text NOT NULL UNIQUE,
  expires_at timestamptz NOT NULL,
  revoked_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX ON insideout.sessions (user_id);
CREATE INDEX ON insideout.sessions (expires_at) WHERE revoked_at IS NULL;
