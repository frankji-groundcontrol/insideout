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

/// The nodes [all] starts from: those with no parent in the list. In a
/// subtree read that is the opened node, whose parent lies outside it.
List<RoadmapNode> roadmapRoots(List<RoadmapNode> all) {
  final ids = {for (final n in all) n.id};
  return all.where((n) => !ids.contains(n.parentId)).toList()
    ..sort((a, b) => a.position.compareTo(b.position));
}

/// The caller's work tree within [all]: every node [userId] owns or that
/// sits below one they own. Its roots are their topmost owned nodes.
List<RoadmapNode> roadmapMine(List<RoadmapNode> all, String userId) {
  final byId = {for (final n in all) n.id: n};
  bool mine(RoadmapNode? n) =>
      n != null && (n.ownerId == userId || mine(byId[n.parentId]));
  return all.where(mine).toList();
}

/// Whether the caller may write [node]'s parent: then [node] lies strictly
/// below a branch they hold, and they may move, delete or re-own it.
bool roadmapParentWritable(List<RoadmapNode> all, RoadmapNode node) =>
    all.any((n) => n.id == node.parentId && n.canWrite);
