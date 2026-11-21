import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:insideout/api/client.dart';
import 'package:insideout/api/errors.dart';
import 'package:insideout/api/models.dart';
import 'package:insideout/api/requests.dart';
import 'package:insideout/chrome/loop_focus.dart';
import 'package:insideout/features/prd/prd_page.dart';
import 'package:insideout/session/appearance.dart';
import 'package:insideout/session/session.dart';
import 'package:provider/provider.dart';

// rv5-FEAT-09 / FE-09: Build sent the AI draft to confirm unseen. The person
// now reads the drafted tree and applies it. rv5 row 1.5 (web half): a
// replace that would erase evidence or done/validated items says how much,
// and only a yes resends with force. The draft no longer names a project;
// the page follows the projectId confirm returns.
const _tree = {
  'title': 'Ship a thing',
  'children': [
    {
      'title': 'Scope',
      'children': [
        {'title': 'Login flow'},
      ],
    },
    {'title': 'Launch'},
  ],
};

class _BuildApi extends ApiClient {
  _BuildApi({this.buildError, List<Object> confirmErrors = const []})
    : _confirmErrors = [...confirmErrors],
      super(
        baseUrl: 'http://127.0.0.1:9/api/v1',
        readAccess: () async => null,
        readRefresh: () async => null,
        writeTokens: (a, r) async {},
        clearTokens: () async {},
      );

  final calls = <ApiCall>[];
  final Object? buildError;
  final List<Object> _confirmErrors;

  @override
  Future<Prd> getPrd(String id) async => Prd(
    id: id,
    workspaceId: 'ws-1',
    title: 'Ship a thing',
    sections: const {},
    status: 'approved',
    currentRevision: 1,
    updatedAt: '2026-10-05T00:00:00Z',
  );

  @override
  Future<Conversation?> conversationForPrd(String prdId) async => null;

  @override
  Future<Workspace> getWorkspace(String id) async => Workspace.fromJson({
    'id': id,
    'title': 'Build WS',
    'description': '',
    'code': 'CODE12',
    'status': 'active',
    'memberCount': 2,
    'myRole': 'admin',
    'createdAt': '2026-01-01T00:00:00Z',
  });

  // The draft writes nothing and, for an unlinked PRD, names no project.
  @override
  Future<Map<String, dynamic>> buildFromPrd(String id) async {
    calls.add(postPrdBuild(id));
    if (buildError != null) throw buildError!;
    return {'tree': _tree};
  }

  @override
  Future<Map<String, dynamic>> confirmBuildFromPrd(
    String id,
    Map<String, dynamic> tree, {
    int? expectedCount,
    bool force = false,
  }) async {
    calls.add(
      postPrdBuildConfirm(id, tree, expectedCount: expectedCount, force: force),
    );
    if (_confirmErrors.isNotEmpty) throw _confirmErrors.removeAt(0);
    return {'projectId': 'proj-9', 'nodeCount': 4};
  }
}

Future<GoRouter> _pump(WidgetTester tester, _BuildApi api) async {
  final session = Session()
    ..ready = true
    ..api = api;
  await tester.binding.setSurfaceSize(const Size(1400, 900));
  addTearDown(() => tester.binding.setSurfaceSize(null));
  final router = GoRouter(
    initialLocation: '/prd/prd-1',
    routes: [
      GoRoute(
        path: '/prd/:id',
        builder: (_, s) => PrdPage(id: s.pathParameters['id']!),
      ),
      GoRoute(
        path: '/projects/:id/roadmap',
        builder: (_, _) => const SizedBox.shrink(),
      ),
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
  return router;
}

String _path(GoRouter r) => r.routeInformationProvider.value.uri.path;

Future<void> _draft(WidgetTester tester) async {
  await tester.tap(find.text('Draft the roadmap'));
  await tester.pumpAndSettle();
}

Future<void> _apply(WidgetTester tester) async {
  await tester.tap(find.text('Apply'));
  await tester.pumpAndSettle();
}

ApiException _conflict(Map<String, dynamic> body) => toApiError(409, body);

void main() {
  testWidgets('the drafted tree is shown, nested, and Cancel writes nothing', (
    tester,
  ) async {
    final api = _BuildApi();
    final router = await _pump(tester, api);
    await _draft(tester);

    for (final title in ['Ship a thing', 'Scope', 'Login flow', 'Launch']) {
      expect(
        find.descendant(of: find.byType(Dialog), matching: find.text(title)),
        findsOneWidget,
        reason: title,
      );
    }
    final inDialog = find.byType(Dialog);
    double left(String t) => tester
        .getTopLeft(find.descendant(of: inDialog, matching: find.text(t)))
        .dx;
    expect(left('Login flow'), greaterThan(left('Scope')));
    expect(left('Scope'), greaterThan(left('Ship a thing')));
    expect(left('Launch'), left('Scope'));

    await tester.tap(find.text('Cancel'));
    await tester.pumpAndSettle();
    expect(api.calls.map((c) => c.path), ['/prds/prd-1/build']);
    expect(_path(router), '/prd/prd-1');
  });

  testWidgets('Apply confirms the tree and opens the project confirm names', (
    tester,
  ) async {
    final api = _BuildApi();
    final router = await _pump(tester, api);
    await _draft(tester);
    await _apply(tester);

    expect(api.calls, hasLength(2));
    expect(api.calls[1].path, '/prds/prd-1/build/confirm');
    expect(api.calls[1].body, {'tree': _tree});
    expect(_path(router), '/projects/proj-9/roadmap');
  });

  testWidgets('a replace with history says what is lost; yes forces it', (
    tester,
  ) async {
    final api = _BuildApi(
      confirmErrors: [
        _conflict({
          'error': 'history',
          'code': 'roadmap_replace_has_history',
          'liveCount': 3,
          'evidenceCount': 2,
          'doneCount': 1,
          'validatedCount': 1,
        }),
      ],
    );
    final router = await _pump(tester, api);
    await _draft(tester);
    await _apply(tester);

    expect(
      find.textContaining(
        '3 nodes, 2 evidence records, 1 done and 1 validated items',
      ),
      findsOneWidget,
    );
    expect(api.calls, hasLength(2), reason: 'nothing is forced unasked');

    await tester.tap(find.text('Delete and replace'));
    await tester.pumpAndSettle();
    expect(api.calls, hasLength(3));
    expect(api.calls[2].body, {
      'tree': _tree,
      'expectedCount': 3,
      'force': true,
    });
    expect(_path(router), '/projects/proj-9/roadmap');
  });

  testWidgets('declining the history prompt keeps the roadmap', (tester) async {
    final api = _BuildApi(
      confirmErrors: [
        _conflict({
          'error': 'history',
          'code': 'roadmap_replace_has_history',
          'liveCount': 3,
          'evidenceCount': 2,
          'doneCount': 0,
          'validatedCount': 0,
        }),
      ],
    );
    final router = await _pump(tester, api);
    await _draft(tester);
    await _apply(tester);
    await tester.tap(find.text('Cancel'));
    await tester.pumpAndSettle();

    expect(api.calls, hasLength(2));
    expect(_path(router), '/prd/prd-1');
  });

  testWidgets('the replace prompt counts the evidence it would erase', (
    tester,
  ) async {
    final api = _BuildApi(
      confirmErrors: [
        _conflict({
          'error': 'replace?',
          'code': 'replace_conflict',
          'liveCount': 5,
          'evidenceCount': 2,
        }),
      ],
    );
    await _pump(tester, api);
    await _draft(tester);
    await _apply(tester);

    expect(find.textContaining('2 evidence records'), findsOneWidget);
    await tester.tap(find.text('Confirm'));
    await tester.pumpAndSettle();
    expect(api.calls[2].body, {'tree': _tree, 'expectedCount': 5});
  });

  testWidgets('an unapproved PRD is refused with a clear message', (
    tester,
  ) async {
    final api = _BuildApi(
      buildError: _conflict({
        'error': 'prd not approved',
        'code': 'prd_not_approved',
      }),
    );
    final router = await _pump(tester, api);
    await tester.tap(find.text('Draft the roadmap'));
    await tester.pump();
    await tester.pump();

    expect(
      find.textContaining('must be approved before its roadmap is built'),
      findsOneWidget,
    );
    expect(_path(router), '/prd/prd-1');
  });
}
