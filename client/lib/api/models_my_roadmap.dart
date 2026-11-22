import 'models_roadmap.dart';

/// A node across a dependency from a "mine" root: what the root waits on or
/// what waits on it.
class RoadmapLink {
  const RoadmapLink({
    required this.id,
    required this.title,
    this.status = '',
    this.ownerName = '',
  });

  final String id;
  final String title;
  final String status;
  final String ownerName;

  factory RoadmapLink.fromJson(Map<String, dynamic> j) => RoadmapLink(
    id: j['id'] as String,
    title: j['title'] as String,
    status: (j['status'] as String?) ?? '',
    ownerName: (j['ownerName'] as String?) ?? '',
  );
}

/// One root of GET /roadmap/mine: a topmost node the caller is in charge of,
/// with its first glance (Now, waits on, blocking). Fields an older server
/// leaves out read as empty.
class MyRoadmapRoot {
  const MyRoadmapRoot({
    required this.read,
    this.workspaceId = '',
    this.now = const [],
    this.waitsOn = const [],
    this.blocking = const [],
  });

  /// The node, its ancestors and its project, as GET /roadmap/{nid} reads.
  final RoadmapNodeRead read;
  final String workspaceId;
  final List<ProgressItem> now;
  final List<RoadmapLink> waitsOn;
  final List<RoadmapLink> blocking;

  /// Whether any Now item is overdue or at high risk.
  bool get pressed =>
      now.any((i) => i.pressure == 'overdue' || i.pressure == 'high_risk');

  factory MyRoadmapRoot.fromJson(Map<String, dynamic> j) => MyRoadmapRoot(
    read: RoadmapNodeRead.fromJson(j),
    workspaceId: (j['workspaceId'] as String?) ?? '',
    now: [
      for (final e in (j['now'] as List?) ?? const [])
        ProgressItem.fromJson(e as Map<String, dynamic>),
    ],
    waitsOn: _links(j['waitsOn']),
    blocking: _links(j['blocking']),
  );
}

List<RoadmapLink> _links(Object? raw) => [
  for (final e in (raw as List?) ?? const [])
    RoadmapLink.fromJson(e as Map<String, dynamic>),
];

/// Who is in charge of a node (GET /roadmap/{nid}/glance): its owner, else
/// the nearest owned ancestor ([via]), else the project owner.
class RoadmapInCharge {
  const RoadmapInCharge({
    required this.userId,
    required this.name,
    this.via,
    this.projectOwner = false,
  });

  final String userId;
  final String name;

  /// The ancestor the charge comes through; null when the node is owned.
  final RoadmapNodeRef? via;
  final bool projectOwner;

  /// The name's first word, for the "Priya's" view toggle.
  String get firstName => name.trim().split(RegExp(r'\s+')).first;

  factory RoadmapInCharge.fromJson(Map<String, dynamic> j) => RoadmapInCharge(
    userId: (j['userId'] as String?) ?? '',
    name: (j['name'] as String?) ?? '',
    via: j['via'] is Map<String, dynamic>
        ? RoadmapNodeRef.fromJson(j['via'] as Map<String, dynamic>)
        : null,
    projectOwner: (j['projectOwner'] as bool?) ?? false,
  );
}

/// A node as its person in charge first sees it: who that is, and the
/// node's Now, what it waits on and what it blocks. Absent fields read empty.
class RoadmapGlance {
  const RoadmapGlance({
    this.inCharge,
    this.now = const [],
    this.waitsOn = const [],
    this.blocking = const [],
  });

  final RoadmapInCharge? inCharge;
  final List<ProgressItem> now;
  final List<RoadmapLink> waitsOn;
  final List<RoadmapLink> blocking;

  bool get isEmpty => now.isEmpty && waitsOn.isEmpty && blocking.isEmpty;

  factory RoadmapGlance.fromJson(Map<String, dynamic> j) => RoadmapGlance(
    inCharge: j['inCharge'] is Map<String, dynamic>
        ? RoadmapInCharge.fromJson(j['inCharge'] as Map<String, dynamic>)
        : null,
    now: [
      for (final e in (j['now'] as List?) ?? const [])
        ProgressItem.fromJson(e as Map<String, dynamic>),
    ],
    waitsOn: _links(j['waitsOn']),
    blocking: _links(j['blocking']),
  );
}
