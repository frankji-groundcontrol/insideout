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
  );
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
