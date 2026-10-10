import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/api/client.dart';
import 'package:insideout/api/models.dart';
import 'package:insideout/chrome/loop_focus.dart';
import 'package:insideout/features/dashboard/dashboard_page.dart';
import 'package:insideout/features/workspace/workspace_page.dart';
import 'package:insideout/session/appearance.dart';
import 'package:insideout/session/session.dart';
import 'package:provider/provider.dart';

// A block form confirmed with a required field blank must say so — the
// modal stays open, the field shows the message, no write goes out —
// instead of closing silently and dropping the tap (r2 follows the
// version-commit dialog's fixed behaviour for every showBlockForm caller).
class _FakeApi extends ApiClient {
  _FakeApi()
    : super(
        baseUrl: 'http://127.0.0.1:9/api/v1',
        readAccess: () async => null,
        readRefresh: () async => null,
        writeTokens: (a, r) async {},
        clearTokens: () async {},
      );

  final createdTitles = <String>[];
  final joinedCodes = <String>[];
  final createdProjects = <String>[];

  @override
  Future<List<Workspace>> listWorkspaces() async => const [];

  @override
  Future<Workspace> createWorkspace(String title, String description) async {
    createdTitles.add(title);
    return _workspace('w-new', title);
  }

  @override
  Future<Workspace> joinWorkspace(String code) async {
    joinedCodes.add(code);
    return _workspace('w-joined', 'Joined WS');
  }

  @override
  Future<Workspace> getWorkspace(String id) async => _workspace('ws1', 'WS');

  @override
  Future<List<Project>> listProjects(String workspaceId) async => const [];

  @override
  Future<Project> createProject(
    String workspaceId,
    String title,
    String description,
  ) async {
    createdProjects.add(title);
    return Project.fromJson({
      'id': 'p-new',
      'workspaceId': workspaceId,
      'title': title,
      'description': description,
      'status': 'planning',
      'repoUrl': '',
      'createdAt': DateTime.now().toUtc().toIso8601String(),
      'flagged': false,
    });
  }
}

Workspace _workspace(String id, String title) => Workspace.fromJson({
  'id': id,
  'title': title,
  'description': '',
  'code': 'C$id',
  'status': 'active',
  'memberCount': 1,
  'myRole': 'admin',
  'createdAt': '2026-10-06T00:00:00Z',
});

Future<_FakeApi> _pump(WidgetTester tester, Widget page) async {
  final session = Session();
  session.ready = true;
  final api = _FakeApi();
  session.api = api;
  await tester.binding.setSurfaceSize(const Size(1400, 900));
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
      child: MaterialApp(home: page),
    ),
  );
  await tester.pumpAndSettle();
  return api;
}

void main() {
  testWidgets('dashboard Create with a blank name keeps the form and says '
      'why', (tester) async {
    final api = await _pump(tester, const DashboardPage());

    await tester.tap(find.text('Create').first);
    await tester.pumpAndSettle();
    await tester.tap(find.text('Create').last);
    await tester.pumpAndSettle();

    expect(find.text('Workspace name'), findsOneWidget);
    expect(find.text('This field is required'), findsOneWidget);
    expect(api.createdTitles, isEmpty);
  });

  testWidgets('a name of only spaces is still blank', (tester) async {
    final api = await _pump(tester, const DashboardPage());

    await tester.tap(find.text('Create').first);
    await tester.pumpAndSettle();
    await tester.enterText(find.byType(TextField), '   ');
    await tester.tap(find.text('Create').last);
    await tester.pumpAndSettle();

    expect(find.text('Workspace name'), findsOneWidget);
    expect(find.text('This field is required'), findsOneWidget);
    expect(api.createdTitles, isEmpty);
  });

  testWidgets('dashboard Join with a blank code keeps the form and says why', (
    tester,
  ) async {
    final api = await _pump(tester, const DashboardPage());

    await tester.tap(find.text('Join'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Join').last);
    await tester.pumpAndSettle();

    expect(find.text('Invite code'), findsOneWidget);
    expect(find.text('This field is required'), findsOneWidget);
    expect(api.joinedCodes, isEmpty);
  });

  testWidgets('workspace New project with a blank name keeps the form', (
    tester,
  ) async {
    final api = await _pump(tester, const WorkspacePage(id: 'ws1'));

    await tester.tap(find.text('New project').first);
    await tester.pumpAndSettle();
    await tester.tap(find.text('Create'));
    await tester.pumpAndSettle();

    expect(find.text('Project name'), findsOneWidget);
    expect(find.text('This field is required'), findsOneWidget);
    expect(api.createdProjects, isEmpty);
  });

  testWidgets('naming the workspace clears the message and makes the call', (
    tester,
  ) async {
    final api = await _pump(tester, const DashboardPage());

    await tester.tap(find.text('Create').first);
    await tester.pumpAndSettle();
    await tester.enterText(find.byType(TextField), 'Fresh WS');
    await tester.tap(find.text('Create').last);
    await tester.pumpAndSettle();

    expect(find.text('Workspace name'), findsNothing);
    expect(api.createdTitles, ['Fresh WS']);
  });
}
