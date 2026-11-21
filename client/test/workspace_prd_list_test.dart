import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:insideout/api/client.dart';
import 'package:insideout/api/models.dart';
import 'package:insideout/chrome/loop_focus.dart';
import 'package:insideout/features/project/project_page.dart';
import 'package:insideout/features/workspace/workspace_page.dart';
import 'package:insideout/session/appearance.dart';
import 'package:insideout/session/session.dart';
import 'package:provider/provider.dart';

// rv5-FE-02: a teammate or the approving admin reaches any PRD of the
// workspace from the workspace page, and a project links to its PRD.
class _Api extends ApiClient {
  _Api()
    : super(
        baseUrl: 'http://127.0.0.1:9/api/v1',
        readAccess: () async => null,
        readRefresh: () async => null,
        writeTokens: (a, r) async {},
        clearTokens: () async {},
      );

  @override
  Future<Workspace> getWorkspace(String id) async => Workspace.fromJson({
    'id': id,
    'title': 'Team',
    'description': '',
    'code': 'CODE12',
    'status': 'active',
    'memberCount': 2,
    'myRole': 'member',
    'createdAt': '2026-01-01T00:00:00Z',
  });

  @override
  Future<List<Project>> listProjects(String workspaceId) async => [
    Project.fromJson({
      'id': 'p1',
      'workspaceId': workspaceId,
      'title': 'Alpha',
      'description': '',
      'status': 'active',
      'repoUrl': '',
      'createdAt': '2026-11-01T00:00:00Z',
      'prdId': 'prd-1',
    }),
  ];

  @override
  Future<List<WorkspacePrd>> listWorkspacePrds(String workspaceId) async => [
    WorkspacePrd.fromJson({
      'id': 'prd-1',
      'title': 'Login revamp',
      'status': 'reviewing',
      'authorId': 'u-dana',
      'authorName': 'dana',
      'projectId': 'p1',
      'updatedAt': '2026-11-20T10:00:00Z',
    }),
  ];

  @override
  Future<Map<String, dynamic>> getProject(String id) async => {
    'id': id,
    'workspaceId': 'ws1',
    'title': 'Alpha',
    'description': '',
    'status': 'active',
    'repoUrl': '',
    'ownerId': 'u-dana',
    'canWritePlan': false,
    'prdId': 'prd-1',
    'updates': const [],
    'nextCursor': null,
  };

  @override
  Future<List<RoadmapNode>> listRoadmap(String projectId) async => [];
}

Future<void> _pump(WidgetTester tester, String start) async {
  final session = Session()
    ..ready = true
    ..api = _Api();
  await tester.binding.setSurfaceSize(const Size(1400, 1600));
  addTearDown(() => tester.binding.setSurfaceSize(null));
  final router = GoRouter(
    initialLocation: start,
    routes: [
      GoRoute(
        path: '/workspace/:id',
        builder: (_, s) => WorkspacePage(id: s.pathParameters['id']!),
      ),
      GoRoute(
        path: '/projects/:id',
        builder: (_, s) => ProjectPage(id: s.pathParameters['id']!),
      ),
      GoRoute(
        path: '/prd/:id',
        builder: (_, s) => Text('PRD page ${s.pathParameters['id']}'),
      ),
    ],
  );
  await tester.pumpWidget(
    MultiProvider(
      providers: [
        ChangeNotifierProvider.value(value: session),
        ChangeNotifierProvider(
          create: (_) =>
              Appearance(store: MemorySettingsStore(), locale: 'en-US'),
        ),
        ChangeNotifierProvider(create: (_) => LoopFocus()),
      ],
      child: MaterialApp.router(routerConfig: router),
    ),
  );
  await tester.pumpAndSettle();
}

void main() {
  testWidgets('the workspace page lists every PRD and opens one', (
    tester,
  ) async {
    await _pump(tester, '/workspace/ws1');
    expect(find.text('PRDS'), findsOneWidget);
    expect(find.text('Login revamp'), findsOneWidget);
    expect(find.text('In review'), findsOneWidget);
    expect(find.textContaining('dana'), findsOneWidget);
    expect(find.text('Project: Alpha'), findsOneWidget);

    await tester.tap(find.text('Login revamp'));
    await tester.pumpAndSettle();
    expect(find.text('PRD page prd-1'), findsOneWidget);
  });

  testWidgets('a project built from a PRD links to it', (tester) async {
    await _pump(tester, '/projects/p1');
    await tester.tap(find.text('Open PRD'));
    await tester.pumpAndSettle();
    expect(find.text('PRD page prd-1'), findsOneWidget);
  });
}
