import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/api/client.dart';
import 'package:insideout/api/models.dart';
import 'package:insideout/chrome/loop_focus.dart';
import 'package:insideout/features/workspace/workspace_page.dart';
import 'package:insideout/session/appearance.dart';
import 'package:insideout/session/session.dart';
import 'package:provider/provider.dart';

// FEAT-01/FEAT-02 behavioral coverage for the board: quiet badge, the
// latest-update line (kind, one-line content, relative time), and the
// status label — replacing substring-grepping with rendered-output
// assertions (CQ-02 pattern).
class _FakeApi extends ApiClient {
  _FakeApi() : super(
            baseUrl: 'http://127.0.0.1:9/api/v1',
            readAccess: () async => null,
            readRefresh: () async => null,
            writeTokens: (a, r) async {},
            clearTokens: () async {},
          );

  @override
  Future<Workspace> getWorkspace(String id) async => Workspace.fromJson({
        'id': 'ws1',
        'title': 'Board Test WS',
        'description': '',
        'code': 'CODE12',
        'status': 'active',
        'memberCount': 2,
        'myRole': 'admin',
        'createdAt': '2026-01-01T00:00:00Z',
      });

  @override
  Future<List<Project>> listProjects(String workspaceId) async => [
        Project.fromJson({
          'id': 'p1',
          'workspaceId': 'ws1',
          'title': 'Quiet Project',
          'description': '',
          'status': 'active',
          'repoUrl': '',
          'createdAt': '2026-09-01T00:00:00Z',
          'flagged': true,
          'latestUpdateKind': 'progress',
          'latestUpdateContent': 'shipped the invitation flow',
          'latestUpdateAt': DateTime.now().toUtc().subtract(const Duration(days: 3)).toIso8601String(),
        }),
        Project.fromJson({
          'id': 'p2',
          'workspaceId': 'ws1',
          'title': 'Fresh Project',
          'description': '',
          'status': 'planning',
          'repoUrl': '',
          'createdAt': DateTime.now().toUtc().toIso8601String(),
          'flagged': false,
        }),
      ];
}

void main() {
  testWidgets('board cards show the quiet badge and the latest update line', (tester) async {
    final session = Session();
    session.ready = true;
    session.api = _FakeApi();
    await tester.binding.setSurfaceSize(const Size(1400, 900));
    addTearDown(() => tester.binding.setSurfaceSize(null));

    await tester.pumpWidget(
      MultiProvider(
        providers: [
          ChangeNotifierProvider.value(value: session),
          ChangeNotifierProvider(create: (_) => Appearance(store: MemorySettingsStore(), locale: 'en-US')),
          ChangeNotifierProvider(create: (_) => LoopFocus()),
        ],
        child: const MaterialApp(home: WorkspacePage(id: 'ws1')),
      ),
    );
    await tester.pumpAndSettle();

    // Both projects render with their status label.
    expect(find.text('Quiet Project'), findsOneWidget);
    expect(find.text('Fresh Project'), findsOneWidget);
    expect(find.text('Active'), findsOneWidget);

    // The flagged project carries the quiet badge; the fresh one does not.
    expect(find.text('gone quiet'), findsOneWidget);

    // The latest update line: kind label, content, relative time.
    expect(find.text('Progress'), findsOneWidget);
    expect(find.text('shipped the invitation flow'), findsOneWidget);
    expect(find.text('3d ago'), findsOneWidget);

    // A board refresh affordance exists (FEAT-02: nothing refreshed before).
    expect(find.byType(RefreshIndicator), findsOneWidget);
  });
}
