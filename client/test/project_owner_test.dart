import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:insideout/api/client.dart';
import 'package:insideout/api/models.dart';
import 'package:insideout/api/requests.dart';
import 'package:insideout/chrome/loop_focus.dart';
import 'package:insideout/features/project/project_page.dart';
import 'package:insideout/features/workspace/workspace_page.dart';
import 'package:insideout/session/appearance.dart';
import 'package:insideout/session/session.dart';
import 'package:provider/provider.dart';

// rv5-FE-04, PARB-08: the project owner is shown on the project and its board
// card, an admin hands the project over from the edit dialog, and ownerId is
// sent only when it changed.
class _Api extends ApiClient {
  _Api()
    : super(
        baseUrl: 'http://127.0.0.1:9/api/v1',
        readAccess: () async => null,
        readRefresh: () async => null,
        writeTokens: (a, r) async {},
        clearTokens: () async {},
      );

  final saves = <String?>[];

  static final _project = {
    'id': 'p1',
    'workspaceId': 'ws1',
    'title': 'Alpha',
    'description': '',
    'status': 'active',
    'repoUrl': '',
    'createdAt': '2026-11-01T00:00:00Z',
    'ownerId': 'u-dana',
    'canWritePlan': true,
    'prdId': null,
  };

  @override
  Future<Map<String, dynamic>> getProject(String id) async => {
    ..._project,
    'updates': const [],
    'nextCursor': null,
  };

  @override
  Future<List<RoadmapNode>> listRoadmap(String projectId) async => [];

  @override
  Future<List<WorkspaceMember>> listMembers(String id) async => [
    WorkspaceMember(
      userId: 'u-me',
      username: 'frank',
      email: 'f@test.local',
      role: 'admin',
    ),
    WorkspaceMember(
      userId: 'u-dana',
      username: 'dana',
      email: 'd@test.local',
      role: 'member',
    ),
  ];

  @override
  Future<Workspace> getWorkspace(String id) async => Workspace.fromJson({
    'id': id,
    'title': 'Team',
    'description': '',
    'code': 'CODE12',
    'status': 'active',
    'memberCount': 2,
    'myRole': 'admin',
    'createdAt': '2026-01-01T00:00:00Z',
  });

  @override
  Future<List<Project>> listProjects(String workspaceId) async => [
    Project.fromJson(_project),
  ];

  @override
  Future<Project> updateProject(
    String id, {
    required String title,
    required String description,
    required String status,
    String? ownerId,
  }) async {
    saves.add(ownerId);
    return Project.fromJson(_project);
  }
}

Future<_Api> _pump(WidgetTester tester, Widget page) async {
  final api = _Api();
  final session = Session()
    ..ready = true
    ..api = api
    ..user = UserProfile(
      id: 'u-me',
      email: 'f@test.local',
      username: 'frank',
      bio: '',
      keywords: const [],
    );
  await tester.binding.setSurfaceSize(const Size(1400, 1600));
  addTearDown(() => tester.binding.setSurfaceSize(null));
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
      // A signed-in session shows the dock, which reads the router.
      child: MaterialApp.router(
        routerConfig: GoRouter(
          routes: [GoRoute(path: '/', builder: (_, _) => page)],
        ),
      ),
    ),
  );
  await tester.pumpAndSettle();
  return api;
}

Future<void> _openEdit(WidgetTester tester) async {
  await tester.tap(find.text('Edit project'));
  await tester.pumpAndSettle();
}

void main() {
  test('a project save without an owner change leaves ownerId out', () {
    final c = patchProject('p1', title: 'T', description: '', status: 'active');
    expect(c.body, {'title': 'T', 'description': '', 'status': 'active'});
  });

  testWidgets('the project page names its owner', (tester) async {
    await _pump(tester, const ProjectPage(id: 'p1'));
    expect(find.text('Owner: dana'), findsOneWidget);
  });

  testWidgets('a save that keeps the owner does not send it', (tester) async {
    final api = await _pump(tester, const ProjectPage(id: 'p1'));
    await _openEdit(tester);
    await tester.tap(find.text('Save').last);
    await tester.pumpAndSettle();
    expect(api.saves, [null]);
  });

  testWidgets('an admin hands the project to another member', (tester) async {
    final api = await _pump(tester, const ProjectPage(id: 'p1'));
    await _openEdit(tester);
    await tester.tap(find.byKey(const Key('project-owner-picker')));
    await tester.pumpAndSettle();
    await tester.tap(find.text('frank').last);
    await tester.pumpAndSettle();
    await tester.tap(find.text('Save').last);
    await tester.pumpAndSettle();
    expect(api.saves, ['u-me']);
  });

  testWidgets('the board card names the owner', (tester) async {
    await _pump(tester, const WorkspacePage(id: 'ws1'));
    expect(find.text('Owner: dana'), findsOneWidget);
  });
}
