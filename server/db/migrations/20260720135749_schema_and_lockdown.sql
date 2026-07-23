-- Create the insideout schema and its shared updated_at trigger.
--
-- No `public`-schema DDL here on purpose. insideout_app's relationship to
-- `public` differs by deployment target and this migration must work on
-- both without editing:
--   - Dedicated instance (bundled docker-compose postgres): insideout_app
--     owns the whole database, and therefore owns `public` too — it can
--     always CREATE there regardless of any REVOKE (ownership bypasses
--     grants against the PUBLIC pseudo-role). A REVOKE here would be a
--     no-op guard against insideout_app itself, not a real boundary.
--   - Shared instance (e.g. a multi-tenant Supabase project used only via
--     its Postgres connection string, other schemas belong to unrelated
--     tenants): insideout_app is schema-scoped, owns only `insideout`,
--     and is never granted CREATE on `public` in the first place — the
--     real boundary is simply that grant never happening. A REVOKE
--     statement here would either fail (insideout_app isn't the owner)
--     or, if it happened to have rights, be a global cross-tenant change.
-- Either way, "never write to public" is enforced by this migration set
-- never targeting it — not by an ineffective/risky REVOKE.
--
-- gen_random_uuid() is native to Postgres 13+ (pg_catalog), so no
-- extension is needed for the UUID PK defaults below.
-- See docs/plans/2026-07-20-go-rewrite/01-database.md §2.
--
-- The insideout schema itself is created by Migrate()'s bootstrap step
-- (internal/store/migrate.go), not here — CREATE SCHEMA needs database-
-- level CREATE privilege, which a schema-scoped role on a shared instance
-- doesn't have even for a redundant IF NOT EXISTS against a schema it
-- already owns. The bootstrap only attempts creation when the schema is
-- actually missing.

CREATE OR REPLACE FUNCTION insideout.set_updated_at()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
  NEW.updated_at := clock_timestamp();
  RETURN NEW;
END;
$$;
