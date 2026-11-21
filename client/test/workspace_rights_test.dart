import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/api/client.dart';
import 'package:insideout/api/models.dart';
import 'package:insideout/chrome/loop_focus.dart';
import 'package:insideout/features/workspace/settings_page.dart';
import 'package:insideout/features/workspace/workspace_page.dart';
import 'package:insideout/session/appearance.dart';
import 'package:insideout/session/session.dart';
import 'package:provider/provider.dart';

// rv5-FE-05: the workspace's Edit, Save and Delete follow the canEdit and
// canDelete the server sends: deleting is the creator's alone, so another
// admin edits but sees no Delete, and a member reads the settings.
class _Api extends ApiClient {
  _Api(this.role, {required this.canEdit, required this.canDelete})
    : super(
        baseUrl: 'http://127.0.0.1:9/api/v1',
        readAccess: () async => null,
        readRefresh: () async => null,
        writeTokens: (a, r) async {},
        clearTokens: () async {},
      );

  final String role;
  final bool canEdit;
  final bool canDelete;

  @override
  Future<Workspace> getWorkspace(String id) async => Workspace.fromJson({
    'id': id,
    'title': 'Team',
    'description': 'What we build',
    'code': 'CODE12',
    'status': 'active',
    'memberCount': 1,
    'myRole': role,
    'canEdit': canEdit,
    'canDelete': canDelete,
    'createdAt': '2026-01-01T00:00:00Z',
  });

  @override
  Future<List<Project>> listProjects(String workspaceId) async => [];

  @override
  Future<List<WorkspaceMember>> listMembers(String id) async => [];

  @override
  Future<List<WorkspacePrd>> listWorkspacePrds(String workspaceId) async => [];
}

Future<void> _pump(WidgetTester tester, _Api api, Widget page) async {
  await tester.binding.setSurfaceSize(const Size(1400, 1800));
  addTearDown(() => tester.binding.setSurfaceSize(null));
  await tester.pumpWidget(
    MultiProvider(
      providers: [
        ChangeNotifierProvider.value(
          value: Session()
            ..ready = true
            ..api = api,
        ),
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
}

const _delete = 'Delete this workspace';

void main() {
  testWidgets('the creator edits and deletes the workspace', (tester) async {
    final api = _Api('admin', canEdit: true, canDelete: true);
    await _pump(tester, api, const WorkspacePage(id: 'ws1'));
    expect(find.text('Edit'), findsOneWidget);
    expect(find.text(_delete), findsOneWidget);
    await _pump(tester, api, const SettingsPage(workspaceId: 'ws1'));
    expect(find.text('Save'), findsOneWidget);
    expect(find.text(_delete), findsOneWidget);
  });

  testWidgets('another admin edits but cannot delete', (tester) async {
    final api = _Api('admin', canEdit: true, canDelete: false);
    await _pump(tester, api, const WorkspacePage(id: 'ws1'));
    expect(find.text('Edit'), findsOneWidget);
    expect(find.text(_delete), findsNothing);
    await _pump(tester, api, const SettingsPage(workspaceId: 'ws1'));
    expect(find.text('Save'), findsOneWidget);
    expect(find.byType(TextField), findsNWidgets(2));
    expect(find.text(_delete), findsNothing);
  });

  testWidgets('a member reads the title and description in settings', (
    tester,
  ) async {
    final api = _Api('member', canEdit: false, canDelete: false);
    await _pump(tester, api, const WorkspacePage(id: 'ws1'));
    expect(find.text('Edit'), findsNothing);
    expect(find.text(_delete), findsNothing);
    await _pump(tester, api, const SettingsPage(workspaceId: 'ws1'));
    expect(find.byType(TextField), findsNothing);
    expect(find.text('Team'), findsOneWidget);
    expect(find.text('What we build'), findsOneWidget);
    expect(find.text('Save'), findsNothing);
    expect(find.text(_delete), findsNothing);
  });
}
