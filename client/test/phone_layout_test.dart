import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:insideout/api/client.dart';
import 'package:insideout/api/models.dart';
import 'package:insideout/chrome/loop_focus.dart';
import 'package:insideout/chrome/product_dock.dart';
import 'package:insideout/features/dashboard/dashboard_page.dart';
import 'package:insideout/features/landing/landing_page.dart';
import 'package:insideout/features/workspace/workspace_page.dart';
import 'package:insideout/session/appearance.dart';
import 'package:insideout/session/session.dart';
import 'package:insideout/theme/block_button.dart';
import 'package:insideout/theme/block_chrome.dart';
import 'package:provider/provider.dart';

import '_auth_test_helpers.dart';

// The reviewer's top five minus the pages that already had narrow tests:
// landing, dashboard and workspace were only ever pumped at desktop sizes
// (1200x5000, 1400x900, 900x560), so a phone regression there passed CI.
// 420 rather than 390 because the Ahem test font is wider (see
// docs/learning/2026-10-10-test-the-width-you-ship-for.md); both sit below
// Cobalt.bpNarrow, the branch the pages actually test against.
const _phone = Size(420, 844);

void _phoneView(WidgetTester tester) {
  // setSurfaceSize resizes the layout but leaves MediaQuery at 800x600,
  // which is what the pages' breakpoints read; resize the view itself.
  tester.view.devicePixelRatio = 1;
  tester.view.physicalSize = _phone;
  addTearDown(tester.view.reset);
}

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

class _FakeApi extends ApiClient {
  _FakeApi()
    : super(
        baseUrl: 'http://127.0.0.1:9/api/v1',
        readAccess: () async => null,
        readRefresh: () async => null,
        writeTokens: (a, r) async {},
        clearTokens: () async {},
      );

  @override
  Future<List<Workspace>> listWorkspaces() async => [
    Workspace.fromJson({
      'id': 'ws1',
      'title': 'First WS',
      'description': '',
      'code': 'CODE12',
      'status': 'active',
      'memberCount': 1,
      'myRole': 'admin',
      'createdAt': '2026-10-06T00:00:00Z',
    }),
  ];

  @override
  Future<Workspace> getWorkspace(String id) async => Workspace.fromJson({
    'id': id,
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
      'workspaceId': workspaceId,
      'title': 'First Project',
      'description': '',
      'status': 'active',
      'repoUrl': '',
      'createdAt': '2026-09-01T00:00:00Z',
      'flagged': false,
    }),
  ];

  @override
  Future<RoadmapProgress> roadmapProgress(String projectId) async =>
      RoadmapProgress.fromJson({'now': const [], 'doneCount': 0});
}

Future<void> _pumpLanding(WidgetTester tester, {required String locale}) async {
  final session = Session()
    ..ready = true
    ..api = QuietAuthApi();
  _phoneView(tester);
  await tester.pumpWidget(
    MultiProvider(
      providers: [
        ChangeNotifierProvider.value(value: session),
        ChangeNotifierProvider(
          create: (_) =>
              Appearance(store: MemorySettingsStore(), locale: locale),
        ),
      ],
      child: const MaterialApp(home: LandingPage()),
    ),
  );
  await tester.pumpAndSettle();
}

/// The dashboard and workspace mount the dock when signed in, and the dock
/// reads the router, so the signed-in pages ride a GoRouter.
Future<void> _pumpSignedIn(
  WidgetTester tester, {
  required String locale,
  required String location,
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
  _phoneView(tester);
  final router = GoRouter(
    initialLocation: location,
    routes: [
      GoRoute(path: '/dashboard', builder: (_, _) => const DashboardPage()),
      GoRoute(
        path: '/workspace/:id',
        builder: (_, s) => WorkspacePage(id: s.pathParameters['id']!),
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

final _scrollable = find.descendant(
  of: find.byType(ListView),
  matching: find.byType(Scrollable),
);

/// Builds (and then fully shows) a card that starts below the fold.
Future<void> _intoView(WidgetTester tester, Finder finder) async {
  await tester.scrollUntilVisible(finder, 300, scrollable: _scrollable);
  await tester.ensureVisible(finder);
  await tester.pumpAndSettle();
}

void _expectDockIconsOnScreen(WidgetTester tester) {
  final dock = find.byType(ProductDock);
  expect(dock, findsOneWidget);
  _expectOnScreen(tester.getRect(dock));
  final icons = find.descendant(of: dock, matching: find.byType(Icon));
  expect(icons, findsNWidgets(6));
  for (var i = 0; i < 6; i++) {
    _expectOnScreen(tester.getRect(icons.at(i)));
  }
}

void _expectRectsOnScreen(WidgetTester tester, Finder finder, int count) {
  expect(finder, findsNWidgets(count));
  for (var i = 0; i < count; i++) {
    _expectOnScreen(tester.getRect(finder.at(i)));
  }
}

// Locale, then the strings each page shows in it.
const _strings = <String, (String, String, String, String, String, String)>{
  'en-US': (
    'Start shaping',
    'Log in',
    'Capture an idea',
    'Create a workspace',
    'Join with a code',
    'New project',
  ),
  'zh-CN': ('开始打磨', '登录', '记下想法', '创建工作区', '用邀请码加入', '新项目'),
};

void main() {
  for (final locale in _strings.keys) {
    final (cta, login, capture, create, join, newProject) = _strings[locale]!;

    testWidgets('landing keeps its hero CTAs on screen ($locale)', (
      tester,
    ) async {
      await _pumpLanding(tester, locale: locale);
      expect(tester.takeException(), isNull);

      // The hero CTA is the first of the two "Start shaping" buttons; the
      // second one sits below the fold in the close section.
      _expectOnScreen(
        tester.getRect(find.widgetWithText(BlockButton, cta).first),
      );
      _expectOnScreen(tester.getRect(find.widgetWithText(BlockButton, login)));
      // Language and theme switches in the app bar.
      _expectRectsOnScreen(tester, find.byType(BlockIconButton), 2);
    });

    testWidgets(
      'dashboard keeps its capture, first card and dock on screen ($locale)',
      (tester) async {
        await _pumpSignedIn(tester, locale: locale, location: '/dashboard');
        expect(tester.takeException(), isNull);

        _expectOnScreen(
          tester.getRect(find.widgetWithText(BlockButton, capture)),
        );
        await _intoView(tester, find.text(create));
        _expectOnScreen(tester.getRect(find.text(create)));
        await _intoView(tester, find.text(join));
        _expectOnScreen(tester.getRect(find.text(join)));
        await _intoView(tester, find.text('First WS'));
        _expectOnScreen(tester.getRect(find.text('First WS')));
        _expectDockIconsOnScreen(tester);
        expect(tester.takeException(), isNull);
      },
    );

    testWidgets(
      'workspace keeps its header actions, FAB, first card and dock on screen '
      '($locale)',
      (tester) async {
        await _pumpSignedIn(tester, locale: locale, location: '/workspace/ws1');
        expect(tester.takeException(), isNull);

        // Language, theme, profile and logout ride the top-right utility
        // pill; the fifth is the invite card's copy button.
        _expectRectsOnScreen(tester, find.byType(BlockIconButton), 5);
        // The quick-capture FAB floats over the dock's top-right corner.
        _expectOnScreen(tester.getRect(find.byTooltip(newProject)));
        await _intoView(tester, find.text('First Project'));
        _expectOnScreen(tester.getRect(find.text('First Project')));
        _expectDockIconsOnScreen(tester);
        expect(tester.takeException(), isNull);
      },
    );
  }
}
