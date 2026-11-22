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
