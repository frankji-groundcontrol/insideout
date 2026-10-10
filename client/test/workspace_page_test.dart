import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:insideout/api/client.dart';
import 'package:insideout/api/errors.dart';
import 'package:insideout/api/models.dart';
import 'package:insideout/chrome/loop_focus.dart';
import 'package:insideout/features/workspace/workspace_page.dart';
import 'package:insideout/session/appearance.dart';
import 'package:insideout/session/session.dart';
import 'package:insideout/theme/field_card.dart';
import 'package:provider/provider.dart';

// FEAT-01/FEAT-02 behavioral coverage for the board: quiet badge, the
// latest-update line (kind, one-line content, relative time), and the
// status label — replacing substring-grepping with rendered-output
// assertions (CQ-02 pattern).
class _FakeApi extends ApiClient {
  _FakeApi({this.onCreateProject})
    : super(
        baseUrl: 'http://127.0.0.1:9/api/v1',
        readAccess: () async => null,
        readRefresh: () async => null,
        writeTokens: (a, r) async {},
        clearTokens: () async {},
      );

  void Function(String title)? onCreateProject;

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
  Future<Project> createProject(
    String workspaceId,
    String title,
    String description,
  ) async {
    onCreateProject?.call(title);
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
      'latestUpdateAt': DateTime.now()
          .toUtc()
          .subtract(const Duration(days: 3))
          .toIso8601String(),
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
  testWidgets('board cards show the quiet badge and the latest update line', (
    tester,
  ) async {
    final session = Session();
    session.ready = true;
    session.api = _FakeApi();
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

    // The latest update line: kind tag (uppercase micro tag), content,
    // relative time.
    expect(find.text('PROGRESS'), findsOneWidget);
    expect(find.text('shipped the invitation flow'), findsOneWidget);
    expect(find.text('3d ago'), findsOneWidget);

    // A board refresh affordance exists (FEAT-02: nothing refreshed before).
    expect(find.byType(RefreshIndicator), findsOneWidget);
  });

  // 4.1: a failing create must surface the localized error and leave the
  // board untouched instead of navigating or silently closing.
  testWidgets('failed project create shows the error SnackBar', (tester) async {
    final session = Session();
    session.ready = true;
    session.api = _FakeApi(
      onCreateProject: (_) => throw ApiException('boom', status: 500),
    );
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
        child: const MaterialApp(home: WorkspacePage(id: 'ws1')),
      ),
    );
    await tester.pumpAndSettle();

    await tester.tap(find.byTooltip('New project'));
    await tester.pumpAndSettle();
    await tester.enterText(find.byType(TextField), 'Failing Project');
    await tester.tap(find.text('Create'));
    await tester.pumpAndSettle();

    expect(
      find.text('Save failed — Server hiccup — try again'),
      findsOneWidget,
    );
    expect(find.text('Failing Project'), findsNothing);
  });

  // N1: with the dock mounted (signed-in chrome) the quick-capture FAB
  // floats over the list's bottom-right; the reserved bottom inset must
  // keep the last project card fully clear of it at max scroll.
  testWidgets('last project card clears the quick-capture FAB at max scroll', (
    tester,
  ) async {
    final session = Session()
      ..ready = true
      ..api = _FakeApi()
      ..user = UserProfile(
        id: 'u1',
        email: 'a@b.c',
        username: 'tester',
        bio: '',
        keywords: const [],
      );
    await tester.binding.setSurfaceSize(const Size(900, 560));
    addTearDown(() => tester.binding.setSurfaceSize(null));

    final router = GoRouter(
      initialLocation: '/workspace/ws1',
      routes: [
        GoRoute(
          path: '/workspace/:id',
          builder: (_, s) => WorkspacePage(id: s.pathParameters['id']!),
        ),
        GoRoute(path: '/dashboard', builder: (_, _) => const Scaffold()),
      ],
    );
    addTearDown(router.dispose);

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

    // Scroll to the absolute bottom of the board.
    await tester.drag(find.byType(ListView), const Offset(0, -4000));
    await tester.pumpAndSettle();

    final cardRect = tester.getRect(
      find.ancestor(
        of: find.text('Fresh Project'),
        matching: find.byType(FieldCard),
      ),
    );
    final fabRect = tester.getRect(find.byTooltip('New project'));
    expect(
      cardRect.bottom,
      lessThanOrEqualTo(fabRect.top),
      reason: 'the last card must never slide under the quick-capture FAB',
    );
  });
}
