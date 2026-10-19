-- rv4 whole-change review (rv4-FR-F38, F25): 20261018120000 added
-- proposal_items.applied_at with no data step. Its comment reads "NULL = not
-- applied yet (every pre-migration row)", which is wrong for the items the old
-- build had already applied: their roadmap nodes exist, applied_at is NULL, and
-- the first accept with apply after the upgrade built every one of them again.
-- A forward migration, because that file already ran on every database that
-- took the wave-2 build. Runs as insideout_owner, which RLS does not apply to,
-- so it sees every workspace.
--
-- An item is stamped only when its node demonstrably exists:
--   * its proposal carries an accepted decision (nothing in a pending or a
--     rejected proposal was applied by this proposal), and
--   * its project holds a node whose description is 'proposed: ' || title, the
--     text the apply wrote on every node it built (a node made by hand has its
--     own description), and
--   * each such node is credited to one item. The items of a project that share
--     a title rank, the already stamped first (they have used their node), then
--     by when their proposal was decided, and only the first N are stamped, N
--     being the number of marked nodes with that description. Two proposals
--     with one title and one node get one stamp, never two.
-- The stamp is the decision time, the nearest honest instant there is. A re-run
-- changes nothing: a stamped item keeps its rank, nothing new qualifies.
--
-- What it cannot see stays NULL, and a repeat accept builds that node once
-- more, as it did before this file: nodes the old web feed created itself after
-- an accept (they carry no mark; a node made by hand looks the same) and nodes
-- deleted since. Stamping those too would also skip items an owner accepted
-- without apply, a worse mistake than one repeated node.
WITH ranked AS (
  SELECT i.id, u.project_id, d.decided_at, 'proposed: ' || i.title AS mark,
         row_number() OVER (
           PARTITION BY u.project_id, 'proposed: ' || i.title
           ORDER BY (i.applied_at IS NULL), d.decided_at, i.position, i.created_at, i.id
         ) AS credit
  FROM insideout.proposal_items i
  JOIN insideout.project_updates u ON u.id = i.update_id
  LEFT JOIN insideout.proposal_decisions d ON d.update_id = i.update_id
  WHERE i.action = 'add_node'
    AND (i.applied_at IS NOT NULL OR d.decision = 'accepted')
), made AS (
  SELECT project_id, description AS mark, count(*) AS nodes
  FROM insideout.roadmap_nodes
  WHERE description LIKE 'proposed: %'
  GROUP BY project_id, description
)
UPDATE insideout.proposal_items p
SET applied_at = r.decided_at
FROM ranked r
JOIN made m ON m.project_id = r.project_id AND m.mark = r.mark
WHERE p.id = r.id AND p.applied_at IS NULL AND r.credit <= m.nodes;
