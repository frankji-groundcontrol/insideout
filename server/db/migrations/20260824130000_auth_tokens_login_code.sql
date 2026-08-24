-- Code login: reuse auth_tokens for emailed 6-digit sign-in codes. A third
-- kind widens the CHECK (the 120000 migration already ran locally, so this
-- is an ALTER, not a rewrite), and a miss counter bounds brute force on a
-- low-entropy code — verify misses increment it and retire the row at five.
ALTER TABLE insideout.auth_tokens DROP CONSTRAINT auth_tokens_kind_check;
ALTER TABLE insideout.auth_tokens
  ADD CONSTRAINT auth_tokens_kind_check
  CHECK (kind IN ('magic_link', 'password_reset', 'login_code'));

ALTER TABLE insideout.auth_tokens ADD COLUMN attempts int NOT NULL DEFAULT 0;
