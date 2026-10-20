-- Coach-conversation retention and account erasure (rv4-DATA-06).
-- The reaper (ReapStaleAIRuns) runs with no app.user_id. Mirror
-- 20260721160000: an unset current_user_id() is the trusted system context,
-- so the purge can delete idle conversations. Messages follow by cascade.
-- Runs as insideout_owner (the migrate runner's role guard).

DO $$
BEGIN
  IF current_user <> 'insideout_owner' THEN
    RAISE EXCEPTION 'must run as insideout_owner, not %', current_user;
  END IF;
END $$;

DROP POLICY IF EXISTS agent_conversations_all ON insideout.agent_conversations;
CREATE POLICY agent_conversations_all ON insideout.agent_conversations
  FOR ALL
  USING (insideout.current_user_id() IS NULL OR user_id = insideout.current_user_id())
  WITH CHECK (insideout.current_user_id() IS NULL OR user_id = insideout.current_user_id());

DROP POLICY IF EXISTS agent_messages_all ON insideout.agent_messages;
CREATE POLICY agent_messages_all ON insideout.agent_messages
  FOR ALL
  USING (
    insideout.current_user_id() IS NULL
    OR insideout._conversation_owner(insideout.agent_messages.conversation_id) = insideout.current_user_id()
  )
  WITH CHECK (
    insideout.current_user_id() IS NULL
    OR insideout._conversation_owner(insideout.agent_messages.conversation_id) = insideout.current_user_id()
  );

-- Ideas have no DELETE policy, so a DELETE from insideout_app matches nothing.
-- The author may delete their own ideas, including after they have left the
-- workspace: erasure removes private text of every status, and a removed
-- member is no longer covered by a membership check. Admins do not gain a
-- path to delete someone else's idea.
DROP POLICY IF EXISTS ideas_delete ON insideout.ideas;
CREATE POLICY ideas_delete ON insideout.ideas
  FOR DELETE USING (author_id = insideout.current_user_id());

-- creator_id is pinned against insideout_app (20261018220000) and email is
-- not in the columns that role may update (20261018140000). These two
-- functions run as insideout_owner, which has BYPASSRLS, so the pins let
-- them through. Each one refuses a caller whose app.user_id is not the
-- account being erased.
CREATE OR REPLACE FUNCTION insideout.reassign_workspace_creator(p_workspace_id uuid, p_from uuid)
RETURNS boolean
LANGUAGE plpgsql SECURITY DEFINER SET search_path = insideout, pg_catalog
AS $$
DECLARE
  v_to uuid;
BEGIN
  IF insideout.current_user_id() IS DISTINCT FROM p_from THEN
    RAISE EXCEPTION 'reassign_workspace_creator: actor is not the creator being replaced'
      USING ERRCODE = 'insufficient_privilege';
  END IF;
  IF NOT EXISTS (
    SELECT 1 FROM insideout.workspaces
    WHERE id = p_workspace_id AND creator_id = p_from
  ) THEN
    RAISE EXCEPTION 'reassign_workspace_creator: workspace creator does not match'
      USING ERRCODE = 'insufficient_privilege';
  END IF;
  SELECT m.user_id INTO v_to
  FROM insideout.workspace_memberships m
  WHERE m.workspace_id = p_workspace_id AND m.user_id <> p_from
  ORDER BY (m.role = 'admin') DESC, m.created_at ASC, m.user_id ASC
  LIMIT 1;
  IF v_to IS NULL THEN
    RETURN false;
  END IF;
  UPDATE insideout.workspaces
  SET creator_id = v_to
  WHERE id = p_workspace_id AND creator_id = p_from;
  RETURN true;
END;
$$;

CREATE OR REPLACE FUNCTION insideout.tombstone_user(p_user_id uuid)
RETURNS void
LANGUAGE plpgsql SECURITY DEFINER SET search_path = insideout, pg_catalog
AS $$
BEGIN
  IF insideout.current_user_id() IS DISTINCT FROM p_user_id THEN
    RAISE EXCEPTION 'tombstone_user: actor is not the account'
      USING ERRCODE = 'insufficient_privilege';
  END IF;
  UPDATE insideout.users
  SET email = 'erased:' || p_user_id::text,
      username = 'erased-' || p_user_id::text,
      password_hash = '!',
      avatar_url = NULL,
      bio = '',
      keywords = '{}',
      meta = '{}',
      role = 'user',
      system_role = 'member'
  WHERE id = p_user_id;
  IF NOT FOUND THEN
    RAISE EXCEPTION 'tombstone_user: no such account'
      USING ERRCODE = 'no_data_found';
  END IF;
END;
$$;

REVOKE ALL ON FUNCTION insideout.reassign_workspace_creator(uuid, uuid) FROM PUBLIC;
REVOKE ALL ON FUNCTION insideout.tombstone_user(uuid) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION insideout.reassign_workspace_creator(uuid, uuid) TO insideout_app;
GRANT EXECUTE ON FUNCTION insideout.tombstone_user(uuid) TO insideout_app;
