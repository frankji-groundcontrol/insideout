import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter/rendering.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:insideout/api/client.dart';
import 'package:insideout/api/models.dart';
import 'package:insideout/chrome/loop_focus.dart';
import 'package:insideout/features/project/roadmap_page.dart';
import 'package:insideout/session/appearance.dart';
import 'package:insideout/session/session.dart';
import 'package:insideout/theme/block_button.dart';
import 'package:insideout/theme/block_chrome.dart';
import 'package:provider/provider.dart';

/// The 390px walk of the web build found the roadmap crowded: the three view
/// toggles rode in the top-right utility cluster, which then ran off the left
/// edge (the Stack clips it, so there was no overflow error to catch), and the
/// "New item" button squeezed the title until "ROADMAP" broke mid-word.
class _FakeApi extends ApiClient {
  _FakeApi()
    : super(
        baseUrl: 'http://127.0.0.1:9/api/v1',
        readAccess: () async => null,
        readRefresh: () async => null,
        writeTokens: (a, r) async {},
        clearTokens: () async {},
      );

  final _parked = Completer<void>();

  @override
  Future<List<RoadmapNode>> listRoadmap(String projectId) async => [];

  @override
  Future<RoadmapProgress> roadmapProgress(String projectId) async =>
      RoadmapProgress.fromJson({
        'now': [
          {
            'id': 'n1',
            'title': 'Payment retry',
            'status': 'in_progress',
            'deadline': '2026-10-20T00:00:00Z',
            'pressure': 'normal',
          },
        ],
        'doneCount': 0,
      });

  @override
  Future<void> presenceStream(
    String projectId,
    String clientId,
    void Function(List<PresenceEntry> entries) onChange, {
    void Function(String sessionId, String name, double x, double y)? onCursor,
    Future<void>? stop,
  }) => _parked.future;

  @override
  Future<void> progressStream(
    String projectId,
    void Function(RoadmapProgress progress) onProgress, {
    Future<void>? stop,
  }) => _parked.future;
}

/// Pumps the signed-in roadmap page (all four utility icons present, the
/// worst case for the cluster) at [size].
Future<void> _pump(
  WidgetTester tester, {
  required String locale,
  required Size size,
}) async {
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
  // setSurfaceSize resizes the layout but leaves MediaQuery at 800x600, which
  // is what the page's breakpoint reads; resize the view itself.
  tester.view.devicePixelRatio = 1;
  tester.view.physicalSize = size;
  addTearDown(tester.view.reset);

  final router = GoRouter(
    initialLocation: '/projects/p1/roadmap',
    routes: [
      GoRoute(
        path: '/projects/:id/roadmap',
        builder: (_, s) => RoadmapPage(projectId: s.pathParameters['id']!),
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
              Appearance(store: MemorySettingsStore(), locale: locale),
        ),
        ChangeNotifierProvider(create: (_) => LoopFocus()),
      ],
      child: MaterialApp.router(routerConfig: router),
    ),
  );
  await tester.pumpAndSettle();
}

const _phone = Size(390, 844);

/// The Stack clips whatever leaves the screen, so a control pushed past an
/// edge raises no overflow error; check the rectangle itself.
void _expectOnScreen(Rect rect) {
  expect(rect.left, greaterThanOrEqualTo(0), reason: '$rect leaves the left');
  expect(rect.top, greaterThanOrEqualTo(0), reason: '$rect leaves the top');
  expect(rect.right, lessThanOrEqualTo(_phone.width), reason: '$rect: right');
  expect(
    rect.bottom,
    lessThanOrEqualTo(_phone.height),
    reason: '$rect: bottom',
  );
}

// Locale, then the strings the page shows in it.
const _locales = <(String, String, String)>[
  ('en-US', 'ROADMAP', 'New item'),
  ('zh-CN', '路线图', '新条目'),
];

void main() {
  for (final (locale, title, newItem) in _locales) {
    testWidgets(
      'at 390px every toggle and utility icon is on screen ($locale)',
      (tester) async {
        await _pump(tester, locale: locale, size: _phone);

        expect(find.byType(BlockChoice), findsNWidgets(3));
        expect(find.byType(BlockIconButton), findsNWidgets(4));
        for (final finder in [
          find.byType(BlockChoice),
          find.byType(BlockIconButton),
        ]) {
          for (var i = 0; i < finder.evaluate().length; i++) {
            _expectOnScreen(tester.getRect(finder.at(i)));
          }
        }
      },
    );

    testWidgets('at 390px the title stays one line ($locale)', (tester) async {
      await _pump(tester, locale: locale, size: _phone);

      // One selection box per laid-out line.
      final lines = tester
          .renderObject<RenderParagraph>(find.text(title))
          .getBoxesForSelection(
            TextSelection(baseOffset: 0, extentOffset: title.length),
          );
      expect(lines, hasLength(1), reason: '"$title" broke across lines');

      _expectOnScreen(
        tester.getRect(find.widgetWithText(BlockButton, newItem)),
      );
    });
  }

  testWidgets('at 390px the view toggles still switch the view', (
    tester,
  ) async {
    await _pump(tester, locale: 'en-US', size: _phone);

    expect(find.text('Payment retry'), findsOneWidget);
    await tester.tap(find.text('Bands'));
    await tester.pumpAndSettle();
    expect(find.text('Payment retry'), findsNothing);
    expect(find.textContaining('No roadmap yet'), findsOneWidget);
  });

  testWidgets('on a wide screen the toggles stay in the utility cluster', (
    tester,
  ) async {
    await _pump(tester, locale: 'en-US', size: const Size(1400, 900));

    final toggle = tester.getRect(find.text('Progress'));
    final language = tester.getRect(find.byIcon(Icons.translate));
    expect(toggle.center.dy, closeTo(language.center.dy, 1));
    expect(toggle.right, lessThan(language.left));
  });
}
