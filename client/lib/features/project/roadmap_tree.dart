import '../../api/models.dart';

/// How many nodes sit beneath [node] in [all] — what the server's cascade
/// (roadmap_nodes.parent_id is ON DELETE CASCADE) deletes along with it.
int roadmapDescendantCount(List<RoadmapNode> all, RoadmapNode node) {
  var count = 0;
  for (final child in all.where((n) => n.parentId == node.id)) {
    count += 1 + roadmapDescendantCount(all, child);
  }
  return count;
}
