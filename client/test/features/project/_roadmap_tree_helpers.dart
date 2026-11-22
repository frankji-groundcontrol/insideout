import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:insideout/api/client.dart';
import 'package:insideout/api/errors.dart';
import 'package:insideout/api/models.dart';
import 'package:insideout/chrome/loop_focus.dart';
import 'package:insideout/features/project/roadmap_node_card.dart';
import 'package:insideout/router.dart';
import 'package:insideout/session/appearance.dart';
import 'package:insideout/session/session.dart';
import 'package:provider/provider.dart';

import '_project_model_helpers.dart';

/// A scripted API over one project tree: Ship > {Payments > Refunds, Search}.
/// Each test reads back what the page or menu sent.
class TreeApi extends ApiClient {
  TreeApi({this.canWritePlan = true, this.canWrite = true})
    : super(
        baseUrl: 'http://127.0.0.1:9/api/v1',
        readAccess: () async => null,
        readRefresh: () async => null,
        writeTokens: (a, r) async {},
        clearTokens: () async {},
      );

  final bool canWritePlan;
  final bool canWrite;
  final _parked = Completer<void>();

  final subtreeRoots = <String>[];
  ({String nodeId, String toUserId, String note})? delegated;
  ({String nodeId, String note})? handedBack;

  /// Every delete asked for, and whether the branch has history: then a
  /// delete without force answers 409 roadmap_delete_has_history.
  final deletes = <({String nodeId, bool force})>[];

  /// Every dependency link asked for or dropped; [dependencyError] refuses
  /// the next link as the server would.
  final dependencyCalls = <({String nodeId, String on, bool remove})>[];
  ApiException? dependencyError;
  bool history = false;
  final _gone = <String>{};

  /// GET /roadmap/{nid}/glance's body; null answers 404, as for a node the
  /// caller cannot see. [glanceCalls] records the nodes asked for.
  Map<String, dynamic>? glance;
  final glanceCalls = <String>[];

  /// The owner of Search, a sibling outside Payments' subtree.
  String? searchOwner;

  RoadmapNode node(String id, String title, String? parent, {String? owner}) =>
      RoadmapNode(
        id: id,
        projectId: 'p1',
        parentId: parent,
        title: title,
        description: id == 'a' ? 'Take money reliably' : '',
        status: 'pending',
        position: 0,
        ownerId: owner,
        ownerName: owner == null ? '' : 'dana',
        canWrite: canWrite,
      );

  List<RoadmapNode> get all => [
    node('r', 'Ship', null),
    node('a', 'Payments', 'r', owner: 'u-dana'),
    node('a1', 'Refunds', 'a'),
    node('b', 'Search', 'r', owner: searchOwner),
  ].where((n) => !_gone.contains(n.id)).toList();

  ApiException get _notFound =>
      ApiException('not found', status: 404, code: 'not_found');

  @override
  Future<void> deleteRoadmapNode(String nodeId, {bool force = false}) async {
    deletes.add((nodeId: nodeId, force: force));
    if (history && !force) {
      throw ApiException(
        'has history',
        status: 409,
        code: 'roadmap_delete_has_history',
        liveCount: 2,
        evidenceCount: 1,
        doneCount: 1,
        validatedCount: 0,
      );
    }
    final doomed = {nodeId};
    for (final n in all) {
      if (doomed.contains(n.parentId)) doomed.add(n.id);
    }
    _gone.addAll(doomed);
  }

  @override
  Future<RoadmapNode> addRoadmapDependency(
    String nodeId,
    String dependsOnId,
  ) async {
    dependencyCalls.add((nodeId: nodeId, on: dependsOnId, remove: false));
    if (dependencyError != null) throw dependencyError!;
    return all.firstWhere((n) => n.id == nodeId);
  }

  @override
  Future<void> removeRoadmapDependency(
    String nodeId,
    String dependsOnId,
  ) async {
    dependencyCalls.add((nodeId: nodeId, on: dependsOnId, remove: true));
  }

  @override
  Future<List<RoadmapNode>> listRoadmap(String projectId) async => all;

  @override
  Future<List<RoadmapNode>> listRoadmapSubtree(
    String projectId,
    String rootId,
  ) async {
    subtreeRoots.add(rootId);
    if (_gone.contains(rootId)) throw _notFound;
    final keep = {rootId};
    for (final n in all) {
      if (keep.contains(n.parentId)) keep.add(n.id);
    }
    return all.where((n) => keep.contains(n.id)).toList();
  }

  @override
  Future<RoadmapNodeRead> readRoadmapNode(String nodeId) async {
    if (_gone.contains(nodeId)) throw _notFound;
    return RoadmapNodeRead(
      node: all.firstWhere((n) => n.id == nodeId),
      ancestors: [
        if (nodeId != 'r') const RoadmapNodeRef(id: 'r', title: 'Ship'),
        if (nodeId == 'a1')
          const RoadmapNodeRef(id: 'a', title: 'Payments', ownerId: 'u-dana'),
      ],
      projectId: 'p1',
      projectName: 'Loop project',
    );
  }

  @override
  Future<RoadmapGlance> roadmapGlance(String nodeId) async {
    glanceCalls.add(nodeId);
    if (glance == null) throw _notFound;
    return RoadmapGlance.fromJson(glance!);
  }

  @override
  Future<RoadmapProgress> roadmapProgress(String projectId) async =>
      RoadmapProgress(now: [], needsDeadline: [], next: [], doneCount: 0);

  @override
  Future<RoadmapProgress> roadmapSubtreeProgress(
    String projectId,
    String rootId,
  ) => roadmapProgress(projectId);

  @override
  Future<ProjectDetail> getProject(String id) async => projectDetail({
    'id': id,
    'workspaceId': 'ws-1',
    'title': 'Loop project',
    'canWritePlan': canWritePlan,
  });

  @override
  Future<List<WorkspaceMember>> listMembers(String id) async => [
    WorkspaceMember(
      userId: 'u-dana',
      username: 'dana',
      email: 'dana@test.local',
      role: 'member',
    ),
    WorkspaceMember(
      userId: 'u-sam',
      username: 'sam',
      email: 'sam@test.local',
      role: 'member',
    ),
  ];

  @override
  Future<RoadmapNode> delegateRoadmapNode(
    String nodeId,
    String toUserId, {
    String note = '',
  }) async {
    delegated = (nodeId: nodeId, toUserId: toUserId, note: note);
    return all.firstWhere((n) => n.id == nodeId);
  }

  @override
  Future<RoadmapNode> handBackRoadmapNode(
    String nodeId, {
    String note = '',
  }) async {
    handedBack = (nodeId: nodeId, note: note);
    return all.firstWhere((n) => n.id == nodeId);
  }

  @override
  Future<List<NodeEvidence>> listNodeEvidence(String nodeId) async => [
    evidenceRow({'kind': 'commit', 'detail': 'retry the charge once'}),
  ];

  @override
  Future<List<NodeAssignment>> listNodeAssignments(String nodeId) async => [
    NodeAssignment.fromJson({
      'id': 'as1',
      'kind': 'delegate',
      'fromUserId': null,
      'fromName': null,
      'toUserId': 'u-dana',
      'toName': 'dana',
      'assignedBy': 'u-me',
      'assignedByName': 'mira',
      'note': 'yours for the sprint',
      'createdAt': '2026-11-20T09:00:00Z',
    }),
  ];

  @override
  Future<void> presenceStream(
    String projectId,
    String clientId,
    void Function(List<PresenceEntry> entries) onChange, {
    void Function(String sessionId, String name, double x, double y)? onCursor,
    Future<void>? stop,
  }) => _parked.future;

  @override
  Future<void> progressStream(
    String projectId,
    void Function(RoadmapProgress progress) onProgress, {
    Future<void>? stop,
  }) => _parked.future;
}

Session treeSession(ApiClient api, {String userId = 'u-me'}) => Session()
  ..ready = true
  ..api = api
  ..user = UserProfile(
    id: userId,
    email: '$userId@test.local',
    username: userId,
    bio: '',
    keywords: const [],
  );

Widget _providers(Session session, Widget child) => MultiProvider(
  providers: [
    ChangeNotifierProvider.value(value: session),
    ChangeNotifierProvider(
      create: (_) => Appearance(store: MemorySettingsStore(), locale: 'en-US'),
    ),
    ChangeNotifierProvider(create: (_) => LoopFocus()),
  ],
  child: child,
);

/// The real router, so paths, breadcrumbs and deep links are the shipped ones.
Future<GoRouter> pumpTreeRouter(WidgetTester tester, Session session) async {
  await tester.binding.setSurfaceSize(const Size(1400, 900));
  addTearDown(() => tester.binding.setSurfaceSize(null));
  final router = buildRouter(session);
  await tester.pumpWidget(
    _providers(session, MaterialApp.router(routerConfig: router)),
  );
  await tester.pumpAndSettle();
  return router;
}

Future<void> pumpTreeCard(
  WidgetTester tester,
  Session session,
  RoadmapNode node, {
  bool planWriter = true,
  int descendants = 0,
}) async {
  await tester.pumpWidget(
    _providers(
      session,
      MaterialApp(
        home: Scaffold(
          body: RoadmapNodeCard(
            projectId: 'p1',
            node: node,
            onReload: () async {},
            planWriter: planWriter,
            descendants: descendants,
          ),
        ),
      ),
    ),
  );
  await tester.pumpAndSettle();
}

Future<void> openNodeMenu(WidgetTester tester) async {
  await tester.tap(find.byType(PopupMenuButton<String>));
  await tester.pumpAndSettle();
}
