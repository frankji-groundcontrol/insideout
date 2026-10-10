import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/api/client.dart';
import 'package:insideout/api/models.dart';
import 'package:insideout/chrome/loop_focus.dart';
import 'package:insideout/features/workspace/workspace_page.dart';
import 'package:insideout/session/appearance.dart';
import 'package:insideout/session/session.dart';
import 'package:insideout/theme/block_chrome.dart';
import 'package:insideout/theme/cobalt.dart';
import 'package:insideout/theme/cobalt_theme.dart';
import 'package:provider/provider.dart';

// Block-craft follow-up: the workspace board's pull-to-refresh painted
// Material's round RefreshProgressIndicator — a spinner — against the
// "loaders are square blocks" decree. The gesture stays (on a phone it is the
// only refresh affordance); what it paints is now the square BlockLoader.
class _FakeApi extends ApiClient {
  _FakeApi()
    : super(
        baseUrl: 'http://127.0.0.1:9/api/v1',
        readAccess: () async => null,
        readRefresh: () async => null,
        writeTokens: (a, r) async {},
        clearTokens: () async {},
      );

  int projectLoads = 0;
  Completer<void>? _gate;

  /// Parks the next project load until [release], so a refresh can be
  /// inspected while it is in flight.
  void hold() => _gate = Completer<void>();

  void release() {
    _gate?.complete();
    _gate = null;
  }

  @override
  Future<Workspace> getWorkspace(String id) async => Workspace.fromJson({
    'id': 'ws1',
    'title': 'Refresh WS',
    'description': '',
    'code': 'CODE12',
    'status': 'active',
    'memberCount': 1,
    'myRole': 'admin',
    'createdAt': '2026-01-01T00:00:00Z',
  });

  @override
  Future<List<Project>> listProjects(String workspaceId) async {
    projectLoads++;
    await _gate?.future;
    return <Project>[];
  }
}

Future<_FakeApi> _pump(WidgetTester tester, {String locale = 'en-US'}) async {
  // Short enough that the board scrolls, so the list can be pulled at all.
  await tester.binding.setSurfaceSize(const Size(900, 420));
  addTearDown(() => tester.binding.setSurfaceSize(null));
  final api = _FakeApi();
  final session = Session()
    ..ready = true
    ..api = api;
  await tester.pumpWidget(
    MultiProvider(
      providers: [
        ChangeNotifierProvider.value(value: session),
        ChangeNotifierProvider(
          create: (_) => Appearance(
            store: MemorySettingsStore(),
            locale: locale,
            dark: false,
          ),
        ),
        ChangeNotifierProvider(create: (_) => LoopFocus()),
      ],
      child: MaterialApp(
        theme: cobaltTheme(Cobalt.light, brightness: Brightness.light),
        home: const WorkspacePage(id: 'ws1'),
      ),
    ),
  );
  await tester.pumpAndSettle();
  return api;
}

/// Any Material progress indicator — circular, linear, or the refresh one.
final _roundOrBar = find.byWidgetPredicate((w) => w is ProgressIndicator);

void main() {
  for (final MapEntry(key: locale, value: label) in const {
    'en-US': 'Refresh',
    'zh-CN': '刷新',
  }.entries) {
    testWidgets('pulling the board refreshes it behind a square loader '
        '($locale)', (tester) async {
      // Released inside the body: the framework verifies handles before
      // tearDowns run.
      final semantics = tester.ensureSemantics();
      try {
        final api = await _pump(tester, locale: locale);
        expect(api.projectLoads, 1);
        expect(find.byType(BlockLoader), findsNothing);

        api.hold();
        await tester.fling(find.byType(ListView), const Offset(0, 400), 1000);
        await tester.pump();
        await tester.pump(const Duration(milliseconds: 400));

        // The pull reached the board's own reload, and while it is in flight
        // the only indicator is the square block — never a spinner.
        expect(api.projectLoads, 2);
        expect(find.byType(BlockLoader), findsOneWidget);
        expect(_roundOrBar, findsNothing);
        // Screen readers keep the localized "Refresh" the stock indicator
        // announced.
        expect(find.bySemanticsLabel(label), findsOneWidget);

        api.release();
        await tester.pump();
        await tester.pumpAndSettle();
        expect(find.byType(BlockLoader), findsNothing);
      } finally {
        semantics.dispose();
      }
    });
  }

  testWidgets('the loader arms only once the pull is long enough to refresh', (
    tester,
  ) async {
    final api = await _pump(tester);
    final gesture = await tester.startGesture(
      tester.getCenter(find.byType(ListView)),
    );

    // A short tug is still just a drag: nothing to show, nothing to refresh.
    await gesture.moveBy(const Offset(0, 24));
    await tester.pump();
    expect(find.byType(BlockLoader), findsNothing);

    // A long pull arms the refresh: the block says "release now" before the
    // finger lifts, and nothing has been fetched yet.
    for (var i = 0; i < 12; i++) {
      await gesture.moveBy(const Offset(0, 30));
      await tester.pump();
    }
    expect(find.byType(BlockLoader), findsOneWidget);
    expect(_roundOrBar, findsNothing);
    expect(api.projectLoads, 1);

    await gesture.up();
    await tester.pump();
    await tester.pumpAndSettle();
    expect(api.projectLoads, 2);
    expect(find.byType(BlockLoader), findsNothing);
  });
}
