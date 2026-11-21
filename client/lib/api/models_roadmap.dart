class RoadmapNode {
  RoadmapNode({
    required this.id,
    required this.projectId,
    required this.title,
    required this.description,
    required this.status,
    required this.position,
    this.deadline,
    this.pressure,
    this.parentId,
    this.creatorName,
    this.editorName,
    this.acceptance = '',
    this.ownerId,
    this.ownerName = '',
    this.validationReason = '',
    this.completionReason = '',
    this.canWrite = false,
    this.rollup = const RoadmapRollup(),
  });

  final String id;
  final String projectId;
  final String? parentId;
  final String title;
  final String description;
  final String status;
  final int position;
  final String? deadline;
  final String? pressure;
  final String? creatorName;
  final String? editorName;

  /// The sentence that says what done means ('' when unset), and the one
  /// human owner of a Now execution item.
  final String acceptance;
  final String? ownerId;
  final String ownerName;

  /// Why a validated node counts as validated ('' when unset) — the
  /// human confirm's own words.
  final String validationReason;

  /// The one-sentence override that marked this node done with no
  /// evidence ('' when unset).
  final String completionReason;

  /// The caller may write this node (the server's authority rule): write
  /// controls show only where they would succeed.
  final bool canWrite;

  /// The leaves at or below this node (a leaf counts itself).
  final RoadmapRollup rollup;

  factory RoadmapNode.fromJson(Map<String, dynamic> j) => RoadmapNode(
    id: j['id'] as String,
    projectId: j['projectId'] as String,
    parentId: j['parentId'] as String?,
    title: j['title'] as String,
    description: (j['description'] as String?) ?? '',
    status: j['status'] as String,
    position: (j['position'] as num?)?.toInt() ?? 0,
    deadline: j['deadline'] as String?,
    pressure: j['pressure'] as String?,
    creatorName: j['creatorName'] as String?,
    editorName: j['editorName'] as String?,
    acceptance: (j['acceptance'] as String?) ?? '',
    ownerId: j['ownerId'] as String?,
    ownerName: (j['ownerName'] as String?) ?? '',
    validationReason: (j['validationReason'] as String?) ?? '',
    completionReason: (j['completionReason'] as String?) ?? '',
    canWrite: (j['canWrite'] as bool?) ?? false,
    rollup: j['rollup'] is Map<String, dynamic>
        ? RoadmapRollup.fromJson(j['rollup'] as Map<String, dynamic>)
        : const RoadmapRollup(),
  );
}

/// Leaf counts under a node; done includes validated, as in Progress.
class RoadmapRollup {
  const RoadmapRollup({
    this.leaves = 0,
    this.done = 0,
    this.validated = 0,
    this.earliestDeadline,
  });

  final int leaves;
  final int done;
  final int validated;

  /// The earliest deadline among the unfinished leaves, if any.
  final String? earliestDeadline;

  factory RoadmapRollup.fromJson(Map<String, dynamic> j) => RoadmapRollup(
    leaves: (j['leaves'] as num?)?.toInt() ?? 0,
    done: (j['done'] as num?)?.toInt() ?? 0,
    validated: (j['validated'] as num?)?.toInt() ?? 0,
    earliestDeadline: j['earliestDeadline'] as String?,
  );
}

class RoadmapNodeRef {
  const RoadmapNodeRef({required this.id, required this.title});

  final String id;
  final String title;

  factory RoadmapNodeRef.fromJson(Map<String, dynamic> j) =>
      RoadmapNodeRef(id: j['id'] as String, title: j['title'] as String);
}

/// GET /roadmap/{nid}: the node, its ancestors root first, and its project.
class RoadmapNodeRead {
  const RoadmapNodeRead({
    required this.node,
    required this.ancestors,
    required this.projectId,
    required this.projectName,
  });

  final RoadmapNode node;
  final List<RoadmapNodeRef> ancestors;
  final String projectId;
  final String projectName;

  factory RoadmapNodeRead.fromJson(Map<String, dynamic> j) => RoadmapNodeRead(
    node: RoadmapNode.fromJson(j['node'] as Map<String, dynamic>),
    ancestors: ((j['ancestors'] as List?) ?? const [])
        .map((e) => RoadmapNodeRef.fromJson(e as Map<String, dynamic>))
        .toList(),
    projectId: j['projectId'] as String,
    projectName: (j['projectName'] as String?) ?? '',
  );
}

/// One row of a node's assignment history (delegate, handback, cleared).
class NodeAssignment {
  const NodeAssignment({
    required this.kind,
    required this.createdAt,
    this.fromName,
    this.toName,
    this.assignedByName,
    this.note = '',
  });

  final String kind;
  final String? fromName;
  final String? toName;
  final String? assignedByName;
  final String note;
  final String createdAt;

  factory NodeAssignment.fromJson(Map<String, dynamic> j) => NodeAssignment(
    kind: j['kind'] as String,
    fromName: j['fromName'] as String?,
    toName: j['toName'] as String?,
    assignedByName: j['assignedByName'] as String?,
    note: (j['note'] as String?) ?? '',
    createdAt: j['createdAt'] as String,
  );
}

/// One sub-task the AI drafted for a node (the 200 body of POST
/// /roadmap/{id}/expand). A draft writes nothing: the person reviews these and
/// only confirming sends them back to be created.
class DraftSubtask {
  const DraftSubtask({required this.title, this.description = ''});

  final String title;
  final String description;

  /// The server omits an empty description, so it is optional here.
  factory DraftSubtask.fromJson(Map<String, dynamic> j) => DraftSubtask(
    title: j['title'] as String,
    description: (j['description'] as String?) ?? '',
  );

  /// The shape the confirm route reads back.
  Map<String, dynamic> toJson() => {'title': title, 'description': description};
}

class PresenceEntry {
  PresenceEntry({
    required this.sessionId,
    required this.userId,
    required this.name,
  });

  final String sessionId;
  final String userId;
  final String name;

  factory PresenceEntry.fromJson(Map<String, dynamic> j) => PresenceEntry(
    sessionId: j['sessionId'] as String,
    userId: j['userId'] as String,
    name: j['name'] as String,
  );
}

class ProgressItem {
  ProgressItem({
    required this.id,
    required this.title,
    required this.status,
    this.deadline,
    this.pressure,
    this.acceptance = '',
    this.ownerName = '',
  });

  final String id;
  final String title;
  final String status;
  final String? deadline;
  final String? pressure;

  /// What done means for this item, and who owns it (empty when unset).
  final String acceptance;
  final String ownerName;

  factory ProgressItem.fromJson(Map<String, dynamic> j) => ProgressItem(
    id: j['id'] as String,
    title: j['title'] as String,
    status: j['status'] as String,
    deadline: j['deadline'] as String?,
    pressure: j['pressure'] as String?,
    acceptance: (j['acceptance'] as String?) ?? '',
    ownerName: (j['ownerName'] as String?) ?? '',
  );
}

class RoadmapProgress {
  RoadmapProgress({
    required this.now,
    required this.needsDeadline,
    required this.next,
    required this.doneCount,
    this.doneItems = const [],
  });

  final List<ProgressItem> now;
  final List<ProgressItem> needsDeadline;
  final List<ProgressItem> next;
  final int doneCount;

  /// The finished leaves with their acceptance and owner, so Done answers
  /// what was finished and by whom, not only how many.
  final List<ProgressItem> doneItems;

  factory RoadmapProgress.fromJson(Map<String, dynamic> j) => RoadmapProgress(
    now: ((j['now'] as List?) ?? const [])
        .map((e) => ProgressItem.fromJson(e as Map<String, dynamic>))
        .toList(),
    needsDeadline: ((j['needsDeadline'] as List?) ?? const [])
        .map((e) => ProgressItem.fromJson(e as Map<String, dynamic>))
        .toList(),
    next: ((j['next'] as List?) ?? const [])
        .map((e) => ProgressItem.fromJson(e as Map<String, dynamic>))
        .toList(),
    doneCount: (j['doneCount'] as num?)?.toInt() ?? 0,
    doneItems: ((j['doneItems'] as List?) ?? const [])
        .map((e) => ProgressItem.fromJson(e as Map<String, dynamic>))
        .toList(),
  );
}
