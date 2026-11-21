import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:insideout/api/client.dart';
import 'package:insideout/api/models.dart';
import 'package:insideout/chrome/loop_focus.dart';
import 'package:insideout/features/project/project_page.dart';
import 'package:insideout/features/workspace/settings_page.dart';
import 'package:insideout/features/workspace/workspace_page.dart';
import 'package:insideout/session/appearance.dart';
import 'package:insideout/session/session.dart';
import 'package:provider/provider.dart';
import 'features/project/_project_model_helpers.dart';

// rv5-FE-05: outside the roadmap, write controls show only to the people the
// server lets write: workspace admins delete the workspace and manage members
// and delete projects; the project owner or an admin decides proposals.
class _Api extends ApiClient {
  _Api(this.role, {this.canWritePlan = false})
    : super(
        baseUrl: 'http://127.0.0.1:9/api/v1',
        readAccess: () async => null,
        readRefresh: () async => null,
        writeTokens: (a, r) async {},
        clearTokens: () async {},
      );

  final String role;
  final bool canWritePlan;

  @override
  Future<Workspace> getWorkspace(String id) async => Workspace.fromJson({
    'id': id,
    'title': 'Team',
    'description': '',
    'code': 'CODE12',
    'status': 'active',
    'memberCount': 2,
    'myRole': role,
    'canEdit': role == 'admin',
    'canDelete': role == 'admin',
    'createdAt': '2026-01-01T00:00:00Z',
  });

  @override
  Future<List<Project>> listProjects(String workspaceId) async => [];

  @override
  Future<List<WorkspaceMember>> listMembers(String id) async => [
    WorkspaceMember(
      userId: 'u-me',
      username: 'me',
      email: 'me@test.local',
      role: role,
    ),
    WorkspaceMember(
      userId: 'u-dana',
      username: 'dana',
      email: 'd@test.local',
      role: 'admin',
    ),
  ];

  @override
  Future<ProjectDetail> getProject(String id) async => projectDetail({
    'id': id,
    'workspaceId': 'ws1',
    'title': 'Alpha',
    'description': '',
    'status': 'active',
    'repoUrl': '',
    'ownerId': 'u-dana',
    'canWritePlan': canWritePlan,
    'prdId': null,
    'nextCursor': null,
    'updates': [
      {
        'id': 'up1',
        'kind': 'agent_proposal',
        'content': 'Add login throttling',
        'createdAt': '2026-11-01T00:00:00Z',
        'authorId': 'agent',
      },
    ],
  });

  @override
  Future<List<RoadmapNode>> listRoadmap(String projectId) async => [];
}

Future<void> _pump(WidgetTester tester, _Api api, Widget page) async {
  final session = Session()
    ..ready = true
    ..api = api
    ..user = UserProfile(
      id: 'u-me',
      email: 'me@test.local',
      username: 'me',
      bio: '',
      keywords: const [],
    );
  await tester.binding.setSurfaceSize(const Size(1400, 1800));
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
      child: MaterialApp.router(
        routerConfig: GoRouter(
          routes: [GoRoute(path: '/', builder: (_, _) => page)],
        ),
      ),
    ),
  );
  await tester.pumpAndSettle();
}

void main() {
  testWidgets('a member sees no workspace delete or edit', (tester) async {
    await _pump(tester, _Api('member'), const WorkspacePage(id: 'ws1'));
    expect(find.text('Delete this workspace'), findsNothing);
    expect(find.text('Edit'), findsNothing);
  });

  testWidgets('an admin sees the workspace delete', (tester) async {
    await _pump(tester, _Api('admin'), const WorkspacePage(id: 'ws1'));
    expect(find.text('Delete this workspace'), findsOneWidget);
  });

  testWidgets('a member manages no other member in settings', (tester) async {
    await _pump(tester, _Api('member'), const SettingsPage(workspaceId: 'ws1'));
    // Only their own card keeps a menu, to leave the workspace.
    expect(find.byType(PopupMenuButton<String>), findsOneWidget);
    expect(find.text('Delete this workspace'), findsNothing);
  });

  testWidgets('a member reads a project without its write controls', (
    tester,
  ) async {
    await _pump(tester, _Api('member'), const ProjectPage(id: 'p1'));
    expect(find.text('Delete'), findsNothing);
    expect(find.text('Edit project'), findsNothing);
    expect(find.text('Accept'), findsNothing);
    expect(find.text('Awaiting a decision'), findsOneWidget);
  });

  testWidgets('an admin deletes and decides', (tester) async {
    await _pump(
      tester,
      _Api('admin', canWritePlan: true),
      const ProjectPage(id: 'p1'),
    );
    expect(find.text('Delete'), findsOneWidget);
    expect(find.text('Edit project'), findsOneWidget);
    expect(find.text('Accept'), findsOneWidget);
  });
}
