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

const _tree = {
  'title': 'Ship a thing',
  'children': [
    {'title': 'Scope'},
  ],
};

/// Records the request each tapped action actually sends. The overrides are
/// the real request builders (`postPrdBuild`, `postPrdBuildConfirm`,
/// `postPrdStatus`); a button wired to the wrong function shows up as the
/// wrong method, path, or body.
class _RecordingApi extends ApiClient {
  _RecordingApi({
    this.saveError,
    this.buildError,
    List<Object> confirmErrors = const [],
  }) : _confirmErrors = [...confirmErrors],
       super(
         baseUrl: 'http://127.0.0.1:9/api/v1',
         readAccess: () async => null,
         readRefresh: () async => null,
         writeTokens: (a, r) async {},
         clearTokens: () async {},
       );

  final calls = <ApiCall>[];
  final Object? saveError;
  final Object? buildError;
  // One entry is thrown per confirm call, oldest first; once empty it succeeds.
  final List<Object> _confirmErrors;

  Prd _prd({String status = 'draft'}) => Prd(
    id: 'prd-1',
    workspaceId: 'ws-1',
    title: 'Ship a thing',
    sections: const {},
    status: status,
    currentRevision: 1,
    updatedAt: '2026-10-05T00:00:00Z',
  );

  @override
  Future<Prd> getPrd(String id) async => _prd();

  @override
  Future<Conversation?> conversationForPrd(String prdId) async => null;

  @override
  Future<Prd> updatePrd(
    String id, {
    String? title,
    Map<String, String>? sections,
  }) async {
    if (saveError != null) throw saveError!;
    return _prd();
  }

  @override
  Future<Prd> updatePrdStatus(String id, String status) async {
    calls.add(postPrdStatus(id, status));
    return _prd(status: 'approved');
  }

  // The server's draft: nothing is written, the tree comes back to be sent on.
  @override
  Future<Map<String, dynamic>> buildFromPrd(String id) async {
    calls.add(postPrdBuild(id));
    if (buildError != null) throw buildError!;
    return {'projectId': 'proj-1', 'tree': _tree};
  }

  @override
  Future<Map<String, dynamic>> confirmBuildFromPrd(
    String id,
    Map<String, dynamic> tree, {
    int? expectedCount,
  }) async {
    calls.add(postPrdBuildConfirm(id, tree, expectedCount: expectedCount));
    if (_confirmErrors.isNotEmpty) throw _confirmErrors.removeAt(0);
    return {'projectId': 'proj-1', 'nodeCount': 2};
  }
}

/// Pumps the PRD page at desktop width in the real providers, with the
/// roadmap route a build navigates to.
Future<GoRouter> _pumpPrd(WidgetTester tester, _RecordingApi api) async {
  final session = Session();
  session.ready = true;
  session.api = api;
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
        path: '/projects/:id',
        builder: (_, _) => const SizedBox.shrink(),
        routes: [
          GoRoute(path: 'roadmap', builder: (_, _) => const SizedBox.shrink()),
        ],
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
  await tester.pump();
  return router;
}

String _path(GoRouter router) => router.routeInformationProvider.value.uri.path;

void main() {
  testWidgets('tapping build and request review sends those requests', (
    tester,
  ) async {
    final api = _RecordingApi();
    final router = await _pumpPrd(tester, api);

    expect(find.text('Next: Request review'), findsOneWidget);
    await tester.tap(find.text('Next: Request review'));
    await tester.pump();

    expect(find.text('Draft the roadmap'), findsOneWidget);
    await tester.tap(find.text('Draft the roadmap'));
    await tester.pump();
    await tester.pump();

    // /build only drafts; the roadmap exists once /build/confirm has run.
    expect(api.calls, hasLength(3));
    expect(api.calls[0].method, 'POST');
    expect(api.calls[0].path, '/prds/prd-1/status');
    expect(api.calls[0].body, {'status': 'reviewing'});
    expect(api.calls[1].method, 'POST');
    expect(api.calls[1].path, '/prds/prd-1/build');
    expect(api.calls[1].body, isNull);
    expect(api.calls[2].method, 'POST');
    expect(api.calls[2].path, '/prds/prd-1/build/confirm');
    expect(api.calls[2].body, {'tree': _tree});
    expect(_path(router), '/projects/proj-1/roadmap');
  });

  testWidgets('building over an existing roadmap asks, then replaces it', (
    tester,
  ) async {
    final api = _RecordingApi(
      confirmErrors: [
        ApiException(
          'replace?',
          status: 409,
          code: 'replace_conflict',
          liveCount: 5,
        ),
      ],
    );
    final router = await _pumpPrd(tester, api);

    await tester.tap(find.text('Next: Request review'));
    await tester.pump();
    await tester.tap(find.text('Draft the roadmap'));
    await tester.pumpAndSettle();

    expect(find.textContaining('with 5 nodes'), findsOneWidget);
    expect(api.calls, hasLength(3), reason: 'nothing is replaced unasked');
    expect(_path(router), '/prd/prd-1');

    await tester.tap(find.text('Confirm'));
    await tester.pumpAndSettle();

    expect(api.calls, hasLength(4));
    expect(api.calls[3].path, '/prds/prd-1/build/confirm');
    expect(api.calls[3].body, {'tree': _tree, 'expectedCount': 5});
    expect(_path(router), '/projects/proj-1/roadmap');
  });

  testWidgets('replacing a roadmap of one node says "1 node"', (tester) async {
    final api = _RecordingApi(
      confirmErrors: [
        ApiException(
          'replace?',
          status: 409,
          code: 'replace_conflict',
          liveCount: 1,
        ),
      ],
    );
    await _pumpPrd(tester, api);

    await tester.tap(find.text('Next: Request review'));
    await tester.pump();
    await tester.tap(find.text('Draft the roadmap'));
    await tester.pumpAndSettle();

    expect(find.textContaining('with 1 node.'), findsOneWidget);
    expect(find.textContaining('1 nodes'), findsNothing);
  });

  testWidgets('declining the replace keeps the roadmap and the PRD page', (
    tester,
  ) async {
    final api = _RecordingApi(
      confirmErrors: [
        ApiException(
          'replace?',
          status: 409,
          code: 'replace_conflict',
          liveCount: 5,
        ),
      ],
    );
    final router = await _pumpPrd(tester, api);

    await tester.tap(find.text('Next: Request review'));
    await tester.pump();
    await tester.tap(find.text('Draft the roadmap'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Cancel'));
    await tester.pumpAndSettle();

    expect(api.calls, hasLength(3), reason: 'no second confirm');
    expect(_path(router), '/prd/prd-1');
  });

  // Register 4.1: a failing build must surface the localized error and
  // keep the user on the PRD instead of navigating nowhere.
  testWidgets('failing build shows the error SnackBar and stays on the PRD', (
    tester,
  ) async {
    final api = _RecordingApi(buildError: ApiException('boom', status: 500));
    final router = await _pumpPrd(tester, api);

    await tester.tap(find.text('Next: Request review'));
    await tester.pump();
    await tester.tap(find.text('Draft the roadmap'));
    await tester.pump();
    await tester.pump();

    expect(
      find.text('Save failed — Server hiccup — try again'),
      findsOneWidget,
    );
    expect(_path(router), '/prd/prd-1');
    expect(api.calls.any((c) => c.path.endsWith('/confirm')), isFalse);
  });

  testWidgets('failing confirm shows the error SnackBar and stays on the PRD', (
    tester,
  ) async {
    final api = _RecordingApi(
      confirmErrors: [ApiException('boom', status: 500)],
    );
    final router = await _pumpPrd(tester, api);

    await tester.tap(find.text('Next: Request review'));
    await tester.pump();
    await tester.tap(find.text('Draft the roadmap'));
    await tester.pump();
    await tester.pump();

    expect(
      find.text('Save failed — Server hiccup — try again'),
      findsOneWidget,
    );
    expect(_path(router), '/prd/prd-1');
  });

  // Register 4.1: a failing save must surface the localized error and keep
  // the edited title in the field.
  testWidgets('failing save shows the error SnackBar and keeps the input', (
    tester,
  ) async {
    final api = _RecordingApi(saveError: ApiException('boom', status: 500));
    await _pumpPrd(tester, api);

    await tester.enterText(find.byType(TextField).first, 'Ship it now');
    await tester.tap(find.text('Save'));
    await tester.pump();
    await tester.pump();

    expect(
      find.text('Save failed — Server hiccup — try again'),
      findsOneWidget,
    );
    expect(
      tester.widget<TextField>(find.byType(TextField).first).controller!.text,
      'Ship it now',
    );
  });
}
