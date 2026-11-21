import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/api/client.dart';
import 'package:insideout/api/models.dart';
import 'package:insideout/chrome/loop_focus.dart';
import 'package:insideout/features/workspace/workspace_page.dart';
import 'package:insideout/session/appearance.dart';
import 'package:insideout/session/session.dart';
import 'package:provider/provider.dart';

// rv5-FE-02: the workspace page's refresh re-reads the PRD list with the
// projects and members, so a PRD filed since the page opened shows up.
class _Api extends ApiClient {
  _Api()
    : super(
        baseUrl: 'http://127.0.0.1:9/api/v1',
        readAccess: () async => null,
        readRefresh: () async => null,
        writeTokens: (a, r) async {},
        clearTokens: () async {},
      );

  final List<Project> projects = [];
  final List<WorkspacePrd> prds = [];
  int prdLoads = 0;

  @override
  Future<Workspace> getWorkspace(String id) async => Workspace.fromJson({
    'id': id,
    'title': 'Team',
    'description': '',
    'code': 'CODE12',
    'status': 'active',
    'memberCount': 1,
    'myRole': 'member',
    'createdAt': '2026-01-01T00:00:00Z',
  });

  // The same list each time: a refresh must not depend on a new instance.
  @override
  Future<List<Project>> listProjects(String workspaceId) async => projects;

  @override
  Future<List<WorkspaceMember>> listMembers(String id) async => [];

  @override
  Future<List<WorkspacePrd>> listWorkspacePrds(String workspaceId) async {
    prdLoads++;
    return [...prds];
  }
}

void main() {
  testWidgets('pulling the board re-reads the PRD list', (tester) async {
    await tester.binding.setSurfaceSize(const Size(900, 1500));
    addTearDown(() => tester.binding.setSurfaceSize(null));
    final api = _Api();
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
        child: const MaterialApp(home: WorkspacePage(id: 'ws1')),
      ),
    );
    await tester.pumpAndSettle();
    expect(api.prdLoads, 1);
    expect(find.text('Login revamp'), findsNothing);

    api.prds.add(
      WorkspacePrd.fromJson({
        'id': 'prd-1',
        'title': 'Login revamp',
        'status': 'reviewing',
        'authorId': 'u-dana',
        'authorName': 'dana',
        'projectId': null,
        'updatedAt': '2026-11-20T10:00:00Z',
      }),
    );
    await tester.fling(find.byType(ListView), const Offset(0, 700), 1000);
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 400));
    await tester.pumpAndSettle();

    expect(api.prdLoads, 2);
    expect(find.text('Login revamp'), findsOneWidget);
  });
}
