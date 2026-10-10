-- TEST-01 (review-3): the pinned "old data" for the upgrade-replay lane
-- (upgrade-replay.sh). This is the row shape the hosted database had when
-- the deployed migration set stopped at 20261006120000: a member, a removed
-- member (no membership rows) who still authors ideas/PRD/updates — the rows
-- the 20261010120000/130000 policy rewrites must still handle — and two
-- workspace invite codes: one legacy 6-digit code from before 2026-07-27
-- (the exact row the 20261010110000 rotation exists for) next to one
-- already-modern 32-hex code that must not move.
--
-- Fixed UUIDs so the lane's assertions stay literal; load once on a fresh
-- schema only (the replay script refuses a non-empty one). Passwords are
-- placeholders: these accounts exist for migrations to trip over, not to
-- log in as.

INSERT INTO insideout.users (id, email, password_hash, username) VALUES
  ('00000000-0000-4000-8000-0000000000a1', 'seed-owner@example.test',
   'seed-not-a-login', 'seed-owner'),
  ('00000000-0000-4000-8000-0000000000b2', 'seed-removed@example.test',
   'seed-not-a-login', 'seed-removed');

INSERT INTO insideout.workspaces (id, title, code, creator_id) VALUES
  -- 6-digit invite code: the pre-2026-07-27 shape the rotation must meet.
  ('00000000-0000-4000-8000-0000000001a1', 'Seed Legacy Workspace', '448172',
   '00000000-0000-4000-8000-0000000000a1'),
  -- Already the 128-bit hex shape; the rotation must leave it byte-identical.
  ('00000000-0000-4000-8000-0000000001b2', 'Seed Modern Workspace',
   '9f8e7d6c5b4a3021f0e1d2c3b4a59687',
   '00000000-0000-4000-8000-0000000000a1');

INSERT INTO insideout.workspace_memberships (workspace_id, user_id, role) VALUES
  ('00000000-0000-4000-8000-0000000001a1', '00000000-0000-4000-8000-0000000000a1',
   'admin');
-- seed-removed deliberately gets NO membership row: they were in the
-- workspace once (their ideas/PRD/update rows prove it) and were removed.

INSERT INTO insideout.projects (id, workspace_id, title, created_by) VALUES
  ('00000000-0000-4000-8000-0000000002a1', '00000000-0000-4000-8000-0000000001a1',
   'Seed Project', '00000000-0000-4000-8000-0000000000a1');

INSERT INTO insideout.ideas (id, workspace_id, author_id, title, content) VALUES
  ('00000000-0000-4000-8000-0000000003a1', '00000000-0000-4000-8000-0000000001a1',
   '00000000-0000-4000-8000-0000000000a1', 'Seed idea by a member',
   'a note-taking app for teams'),
  ('00000000-0000-4000-8000-0000000003b2', '00000000-0000-4000-8000-0000000001a1',
   '00000000-0000-4000-8000-0000000000b2', 'Seed idea by a removed member',
   'authored before removal');

INSERT INTO insideout.prds (id, workspace_id, idea_id, author_id, title, sections) VALUES
  ('00000000-0000-4000-8000-0000000004a1', '00000000-0000-4000-8000-0000000001a1',
   '00000000-0000-4000-8000-0000000003a1', '00000000-0000-4000-8000-0000000000a1',
   'Seed PRD by a member', '{"background": "seed"}'),
  ('00000000-0000-4000-8000-0000000004b2', '00000000-0000-4000-8000-0000000001a1',
   NULL, '00000000-0000-4000-8000-0000000000b2',
   'Seed PRD by a removed member', '{"background": "seed"}');

INSERT INTO insideout.project_updates (id, project_id, author_id, kind, content) VALUES
  ('00000000-0000-4000-8000-0000000005a1', '00000000-0000-4000-8000-0000000002a1',
   '00000000-0000-4000-8000-0000000000a1', 'progress', 'seed progress by a member'),
  ('00000000-0000-4000-8000-0000000005b2', '00000000-0000-4000-8000-0000000002a1',
   '00000000-0000-4000-8000-0000000000b2', 'note', 'seed note by a removed member');
