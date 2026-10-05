import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/api/client.dart';
import 'package:insideout/api/models.dart';
import 'package:insideout/chrome/loop_focus.dart';
import 'package:insideout/features/project/project_page.dart';
import 'package:insideout/features/workspace/workspace_page.dart';
import 'package:insideout/session/appearance.dart';
import 'package:insideout/session/session.dart';
import 'package:provider/provider.dart';

import 'roadmap_page_test.dart' show MemorySettingsStore;

class _BoardApi extends ApiClient {
  _BoardApi({this.nodes = const [], this.evidence = const []})
      : super(
          baseUrl: 'http://127.0.0.1:9/api/v1',
          readAccess: () async => null,
          readRefresh: () async => null,
          writeTokens: (a, r) async {},
          clearTokens: () async {},
        );

  final List<RoadmapNode> nodes;
  final List<Map<String, dynamic>> evidence;

  @override
  Future<Workspace> getWorkspace(String id) async => Workspace(
        id: id,
        title: 'Gap WS',
        description: '',
        code: 'GAP1',
        status: 'active',
        memberCount: 2,
        myRole: 'owner',
        createdAt: '2026-10-05T00:00:00Z',
      );

  @override
  Future<List<Project>> listProjects(String workspaceId) async => [
        Project(
          id: 'p1',
          workspaceId: workspaceId,
          title: 'Board project',
          description: '',
          status: 'active',
          repoUrl: '',
          createdAt: '2026-10-05T00:00:00Z',
          flagged: false,
        ),
      ];

  @override
  Future<RoadmapProgress> roadmapProgress(String projectId) async => RoadmapProgress(
        now: [ProgressItem(id: 'a', title: 'One in motion', status: 'in_progress')],
        needsDeadline: const [],
        next: const [],
        doneCount: 2,
      );

  @override
  Future<Map<String, dynamic>> getProject(String id) async => {
        'id': id,
        'workspaceId': 'ws-1',
        'title': 'Evidence Page Test',
        'description': '',
        'status': 'active',
        'repoUrl': '',
        'ownerId': 'u1',
        'updates': const <Map<String, dynamic>>[],
      };

  @override
  Future<List<RoadmapNode>> listRoadmap(String projectId) async => nodes;

  @override
  Future<List<Map<String, dynamic>>> listNodeEvidence(String nodeId) async => evidence;
}

Future<void> _pump(WidgetTester tester, Widget child, ApiClient api) async {
  final session = Session()
    ..ready = true
    ..api = api;
  await tester.binding.setSurfaceSize(const Size(1400, 900));
  addTearDown(() => tester.binding.setSurfaceSize(null));
  await tester.pumpWidget(
    MultiProvider(
      providers: [
        ChangeNotifierProvider.value(value: session),
        ChangeNotifierProvider(create: (_) => Appearance(store: MemorySettingsStore(), locale: 'en-US')),
        ChangeNotifierProvider(create: (_) => LoopFocus()),
      ],
      child: MaterialApp(home: child),
    ),
  );
  await tester.pumpAndSettle();
}

void main() {
  testWidgets('the workspace board answers where each project stands', (tester) async {
    await _pump(tester, WorkspacePage(id: 'ws-1'), _BoardApi());
    expect(find.text('Board project'), findsOneWidget);
    expect(find.text('Now 1 · Done 2'), findsOneWidget);
  });

  testWidgets('the Evidence screen lists what finished work produced', (tester) async {
    await _pump(
      tester,
      ProjectPage(id: 'p1'),
      _BoardApi(
        nodes: [
          RoadmapNode(id: 'root', projectId: 'p1', parentId: null, title: 'Milestone', description: '', status: 'in_progress', position: 0),
          RoadmapNode(id: 'leaf', projectId: 'p1', parentId: 'root', title: 'Wire the flow', description: '', status: 'done', position: 0),
        ],
        evidence: [
          {'id': 'e1', 'nodeId': 'leaf', 'kind': 'commit', 'detail': 'merged #12 — healthz wiring', 'sourceUrl': '', 'createdAt': '2026-10-05T00:00:00Z'},
        ],
      ),
    );
    expect(find.text('Done, with its evidence'), findsOneWidget);
    expect(find.text('Wire the flow'), findsOneWidget);
    expect(find.textContaining('merged #12 — healthz wiring'), findsOneWidget);
  });
}
