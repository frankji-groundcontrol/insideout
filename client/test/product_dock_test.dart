import 'package:flutter/gestures.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:insideout/api/models.dart';
import 'package:insideout/app.dart';
import 'package:insideout/chrome/loop_focus.dart';
import 'package:insideout/chrome/product_dock.dart';
import 'package:insideout/session/appearance.dart';
import 'package:insideout/session/session.dart';
import 'package:insideout/widgets/pressable_block.dart';
import 'package:provider/provider.dart';

/// Pumps a signed-in shell with the dock on [initialLocation]. The session
/// remembers nothing by default; tests pass a seeded [LoopFocus] to model
/// warm (pages loaded ids) or cold (fresh dashboard) states.
Future<void> _pump(
  WidgetTester tester, {
  required LoopFocus focus,
  String initialLocation = '/projects/proj-1/roadmap',
}) async {
  final session = Session();
  session.ready = true;

  Widget page(String label) =>
      Scaffold(body: Text(label), bottomNavigationBar: const ProductDock());

  final router = GoRouter(
    initialLocation: initialLocation,
    routes: [
      GoRoute(path: '/dashboard', builder: (_, _) => page('Dashboard')),
      GoRoute(
        path: '/workspace/:id/ideas',
        builder: (_, s) => page('Ideas ${s.pathParameters['id']}'),
      ),
      GoRoute(
        path: '/prd/:id',
        builder: (_, s) => page('Version ${s.pathParameters['id']}'),
      ),
      GoRoute(
        path: '/prd/:id/versions',
        builder: (_, s) => page('Commit ${s.pathParameters['id']}'),
      ),
      GoRoute(
        path: '/projects/:id/roadmap',
        builder: (_, s) => page('Roadmap ${s.pathParameters['id']}'),
      ),
      GoRoute(
        path: '/projects/:id',
        builder: (_, s) => page('Evidence ${s.pathParameters['id']}'),
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
        ChangeNotifierProvider.value(value: focus),
      ],
      child: MaterialApp.router(routerConfig: router),
    ),
  );
  await tester.pump();
  await tester.pump(const Duration(milliseconds: 50));
}

/// Every id the dock can need, so all six items are enabled.
LoopFocus warmFocus({bool committed = true}) => LoopFocus()
  ..rememberWorkspace('ws-1')
  ..rememberProject('proj-1', workspaceId: 'ws-1')
  ..rememberPrd('prd-1', committed: committed);

Finder disabledButtons() => find.byWidgetPredicate(
  (w) =>
      w is Semantics &&
      w.properties.button == true &&
      w.properties.enabled == false,
);

void main() {
  testWidgets(
    'an uncommitted PRD marks Commit unfinished and a tap reaches the versions route',
    (tester) async {
      await tester.binding.setSurfaceSize(const Size(1400, 900));
      addTearDown(() => tester.binding.setSurfaceSize(null));

      // The page loaded the working version and recorded that nobody has
      // committed it; the dock names Commit and routes to /prd/<id>/versions.
      final focus = LoopFocus()..rememberPrd('prd-1', committed: false);
      await _pump(tester, focus: focus);

      expect(
        find.byKey(const ValueKey('dock-unfinished-commit')),
        findsOneWidget,
      );
      expect(find.text('Unfinished: Commit'), findsOneWidget);
      expect(find.text('Dashboard'), findsNothing);

      // The unfinished indicator is a square amber block (~7:1 on the dock
      // brand), not the old blue-on-blue warn tint on icon/text/border.
      final amber = find.descendant(
        of: find.byKey(const ValueKey('dock-unfinished-commit')),
        matching: find.byType(ColoredBox),
      );
      expect(amber, findsOneWidget);
      expect(tester.widget<ColoredBox>(amber).color, const Color(0xFFFFD166));
      expect(tester.getSize(amber), const Size(8, 8));
      final icon = find.descendant(
        of: find.byKey(const ValueKey('dock-unfinished-commit')),
        matching: find.byIcon(Icons.verified_outlined),
      );
      expect(tester.widget<Icon>(icon).color, const Color(0xFFFFFFFF));

      await tester.tap(find.text('Unfinished: Commit'));
      await tester.pumpAndSettle();

      expect(find.text('Dashboard'), findsNothing);
      expect(find.text('Commit prd-1'), findsOneWidget);
    },
  );

  testWidgets(
    'cold focus disables every id-dependent item without navigating',
    (tester) async {
      await tester.binding.setSurfaceSize(const Size(1400, 900));
      addTearDown(() => tester.binding.setSurfaceSize(null));

      // Fresh dashboard: no remembered workspace/project/PRD. Items that need
      // an id render disabled (dimmed, semantics enabled: false) instead of
      // silently bouncing back to /dashboard.
      final focus = LoopFocus();
      await _pump(tester, focus: focus, initialLocation: '/dashboard');

      expect(find.text('Dashboard'), findsOneWidget);
      expect(disabledButtons(), findsNWidgets(6));

      for (final label in [
        'Capture',
        'Ideas',
        'Version',
        'Commit',
        'Roadmap',
        'Evidence',
      ]) {
        await tester.tap(find.text(label));
        await tester.pumpAndSettle();
      }

      // Nothing navigated, nothing pulsed.
      expect(find.text('Dashboard'), findsOneWidget);
      expect(focus.evidencePulse, 0);
    },
  );

  testWidgets('a commit tap with no PRD id stays put even on a project route', (
    tester,
  ) async {
    await tester.binding.setSurfaceSize(const Size(1400, 900));
    addTearDown(() => tester.binding.setSurfaceSize(null));

    // The route carries a project id, but Commit needs a PRD id the session
    // never learned — the old build fell through to stay(); now it disables.
    final focus = LoopFocus()..prdCommitted = false;
    await _pump(tester, focus: focus);

    expect(find.text('Roadmap proj-1'), findsOneWidget);
    expect(find.text('Unfinished: Commit'), findsOneWidget);
    expect(
      find.descendant(
        of: find.byKey(const ValueKey('dock-unfinished-commit')),
        matching: disabledButtons(),
      ),
      findsOneWidget,
    );

    await tester.tap(find.text('Unfinished: Commit'));
    await tester.pumpAndSettle();

    expect(find.text('Dashboard'), findsNothing);
    expect(find.text('Commit prd-1'), findsNothing);
    expect(find.text('Roadmap proj-1'), findsOneWidget);
  });

  testWidgets(
    'a project tap on a workspace route resolves to that workspace\'s project',
    (tester) async {
      await tester.binding.setSurfaceSize(const Size(1400, 900));
      addTearDown(() => tester.binding.setSurfaceSize(null));

      // ws-2 seeded proj-2, then a direct visit to proj-1 moved the global
      // last-project. From the ws-2 route the dock must still target proj-2.
      final focus = LoopFocus()
        ..rememberProject('proj-2', workspaceId: 'ws-2')
        ..rememberProject('proj-1');
      await _pump(
        tester,
        focus: focus,
        initialLocation: '/workspace/ws-2/ideas',
      );

      expect(find.text('Ideas ws-2'), findsOneWidget);
      await tester.tap(find.text('Roadmap'));
      await tester.pumpAndSettle();

      expect(find.text('Roadmap proj-1'), findsNothing);
      expect(find.text('Roadmap proj-2'), findsOneWidget);
    },
  );

  testWidgets('the version item rides to /prd/<id>/versions, not a self-push', (
    tester,
  ) async {
    await tester.binding.setSurfaceSize(const Size(1400, 900));
    addTearDown(() => tester.binding.setSurfaceSize(null));

    // On the PRD desk the old /prd/<id> target re-pushed the page already
    // open (the register's dead click). Now it goes to the versions route.
    final focus = warmFocus();
    await _pump(tester, focus: focus, initialLocation: '/prd/prd-1');

    expect(find.text('Version prd-1'), findsOneWidget);
    await tester.tap(find.text('Version'));
    await tester.pumpAndSettle();

    expect(find.text('Commit prd-1'), findsOneWidget);
  });

  testWidgets(
    'evidence navigates when elsewhere and pulses a scroll-highlight at its destination',
    (tester) async {
      await tester.binding.setSurfaceSize(const Size(1400, 900));
      addTearDown(() => tester.binding.setSurfaceSize(null));

      final focus = warmFocus();
      await _pump(tester, focus: focus);

      // Not yet on the evidence destination: a plain navigation.
      await tester.tap(find.text('Evidence'));
      await tester.pumpAndSettle();
      expect(find.text('Evidence proj-1'), findsOneWidget);
      expect(focus.evidencePulse, 0);

      // Already on /projects/<id>: no re-push, the page gets the pulse.
      await tester.tap(find.text('Evidence'));
      await tester.pumpAndSettle();
      expect(find.text('Evidence proj-1'), findsOneWidget);
      expect(focus.evidencePulse, 1);
    },
  );

  testWidgets(
    'at 390px the dock collapses to icons: six items fit, no overflow, still tappable',
    (tester) async {
      await tester.binding.setSurfaceSize(const Size(390, 844));
      addTearDown(() => tester.binding.setSurfaceSize(null));

      // The unfinished long label is exactly the case that used to blow the
      // dock out to the viewport edges and shift sibling hit targets.
      final focus = warmFocus(committed: false);
      await _pump(tester, focus: focus);

      expect(tester.takeException(), isNull);

      // The unfinished label must not stretch the pill out to the 390px edges.
      final pill = tester.getRect(
        find.byWidgetPredicate(
          (w) =>
              w is Container &&
              w.decoration is BoxDecoration &&
              (w.decoration! as BoxDecoration).color == const Color(0xFF3B5BFF),
        ),
      );
      expect(pill.width, lessThan(360));

      final itemFinder = find.byType(PressableBlock);
      expect(itemFinder, findsNWidgets(6));

      // Every item is fully on the 390px surface, non-overlapping, and
      // comfortably tappable.
      final rects = <Rect>[
        for (var i = 0; i < 6; i++) tester.getRect(itemFinder.at(i)),
      ]..sort((a, b) => a.left.compareTo(b.left));
      for (var i = 0; i < rects.length; i++) {
        expect(rects[i].left, greaterThanOrEqualTo(0));
        expect(rects[i].right, lessThanOrEqualTo(390));
        expect(rects[i].width, greaterThanOrEqualTo(40));
        if (i > 0) {
          expect(rects[i].left, greaterThanOrEqualTo(rects[i - 1].right));
        }
      }

      // Icons only: no item labels, and the unfinished amber square remains.
      expect(find.text('Commit'), findsNothing);
      expect(find.text('Unfinished: Commit'), findsNothing);
      expect(
        find.byKey(const ValueKey('dock-unfinished-commit')),
        findsOneWidget,
      );

      // Icon-only items still navigate.
      await tester.tap(find.byIcon(Icons.description_outlined));
      await tester.pumpAndSettle();
      expect(find.text('Commit prd-1'), findsOneWidget);
    },
  );

  testWidgets(
    'dock items carry the cast at rest and press flat into the page over 100ms',
    (tester) async {
      await tester.binding.setSurfaceSize(const Size(1400, 900));
      addTearDown(() => tester.binding.setSurfaceSize(null));

      final focus = warmFocus();
      await _pump(tester, focus: focus);

      final item = find.byType(PressableBlock).first;
      final animated = find.descendant(
        of: item,
        matching: find.byType(AnimatedContainer),
      );
      final transform = find.descendant(
        of: item,
        matching: find.byType(Transform),
      );
      double translationX() =>
          tester.widget<Transform>(transform).transform.getTranslation().x;
      BoxDecoration shadowDecoration() =>
          tester
                  .widgetList<DecoratedBox>(
                    find.descendant(
                      of: item,
                      matching: find.byType(DecoratedBox),
                    ),
                  )
                  .firstWhere(
                    (d) =>
                        d.decoration is BoxDecoration &&
                        (d.decoration as BoxDecoration).boxShadow != null,
                  )
                  .decoration
              as BoxDecoration;

      // The press motion animates over the 100ms decree duration (the 180ms
      // sibling is the selection-highlight container).
      final pressContainer = tester
          .widgetList<AnimatedContainer>(animated)
          .firstWhere((c) => c.duration == const Duration(milliseconds: 100));
      expect(pressContainer.duration, const Duration(milliseconds: 100));

      // Rest: full cast shadow (4,4), zero blur, no translation.
      expect(
        shadowDecoration().boxShadow!.single,
        isA<BoxShadow>()
            .having((s) => s.offset, 'offset', const Offset(4, 4))
            .having((s) => s.blurRadius, 'blurRadius', 0),
      );
      expect(translationX(), 0);

      // Press: the block travels to (4,4) and the shadow is gone; halfway
      // through the decree window it sits between rest and pressed. (The first
      // pump applies the press; the tween then runs over the next 100ms.)
      final gesture = await tester.startGesture(
        tester.getRect(item).center,
        kind: PointerDeviceKind.mouse,
      );
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 50));
      expect(translationX(), moreOrLessEquals(2, epsilon: 0.1));
      await tester.pump(const Duration(milliseconds: 60));
      expect(translationX(), 4);
      expect(shadowDecoration().boxShadow, isEmpty);
      await gesture.up();
      await tester.pump();
    },
  );

  testWidgets(
    'a hover shifts the dock item (2,2) and shrinks the cast to (2,2)',
    (tester) async {
      await tester.binding.setSurfaceSize(const Size(1400, 900));
      addTearDown(() => tester.binding.setSurfaceSize(null));

      final focus = warmFocus();
      await _pump(tester, focus: focus);

      final item = find.byType(PressableBlock).first;
      final transform = find.descendant(
        of: item,
        matching: find.byType(Transform),
      );
      double translationX() =>
          tester.widget<Transform>(transform).transform.getTranslation().x;
      BoxDecoration shadowDecoration() =>
          tester
                  .widgetList<DecoratedBox>(
                    find.descendant(
                      of: item,
                      matching: find.byType(DecoratedBox),
                    ),
                  )
                  .firstWhere(
                    (d) =>
                        d.decoration is BoxDecoration &&
                        (d.decoration as BoxDecoration).boxShadow != null,
                  )
                  .decoration
              as BoxDecoration;

      // Park a mouse pointer far from the dock, then glide it over the item:
      // translate (2,2) with the cast shrunk to castSm.
      final mouse = await tester.startGesture(
        const Offset(20, 300),
        kind: PointerDeviceKind.mouse,
      );
      await mouse.up();
      await mouse.moveTo(tester.getRect(item).center);
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 50));
      expect(translationX(), moreOrLessEquals(1, epsilon: 0.1));
      await tester.pump(const Duration(milliseconds: 60));
      expect(translationX(), 2);
      expect(
        shadowDecoration().boxShadow!.single,
        isA<BoxShadow>()
            .having((s) => s.offset, 'offset', const Offset(2, 2))
            .having((s) => s.blurRadius, 'blurRadius', 0),
      );

      // Glide off: the block settles back on the full cast.
      await mouse.moveTo(const Offset(20, 300));
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 120));
      expect(translationX(), 0);
      expect(shadowDecoration().boxShadow!.single.offset, const Offset(4, 4));
      await mouse.removePointer();
    },
  );

  testWidgets(
    'the dock is a Cobalt-Neo block: 0 radius, 2px ink border, cast shadow',
    (tester) async {
      await tester.binding.setSurfaceSize(const Size(1400, 900));
      addTearDown(() => tester.binding.setSurfaceSize(null));

      final focus = warmFocus();
      await _pump(tester, focus: focus);

      final dock = tester.widget<Container>(
        find.byWidgetPredicate(
          (w) =>
              w is Container &&
              w.decoration is BoxDecoration &&
              (w.decoration! as BoxDecoration).color == const Color(0xFF3B5BFF),
        ),
      );
      final decoration = dock.decoration! as BoxDecoration;
      expect(decoration.borderRadius, BorderRadius.zero);
      final border = decoration.border! as Border;
      expect(border.top.width, 2);
      expect(
        decoration.boxShadow!.single,
        isA<BoxShadow>()
            .having((s) => s.offset, 'offset', const Offset(4, 4))
            .having((s) => s.blurRadius, 'blurRadius', 0),
      );
    },
  );

  testWidgets(
    'extendBody scaffold at 1400x900 keeps the dock off the body landmark',
    (tester) async {
      await tester.binding.setSurfaceSize(const Size(1400, 900));
      addTearDown(() => tester.binding.setSurfaceSize(null));

      final session = Session()
        ..ready = true
        ..user = UserProfile(
          id: 'u1',
          email: 'a@b.c',
          username: 'tester',
          bio: '',
          keywords: const [],
        );
      final router = GoRouter(
        initialLocation: '/projects/proj-1',
        routes: [
          GoRoute(
            path: '/projects/:id',
            builder: (_, _) => const AppScaffold(
              body: Center(
                child: Text('landmark', key: ValueKey('body-landmark')),
              ),
            ),
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
            ChangeNotifierProvider.value(value: warmFocus()),
          ],
          child: MaterialApp.router(routerConfig: router),
        ),
      );
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 50));

      expect(tester.widget<Scaffold>(find.byType(Scaffold)).extendBody, isTrue);

      final dockRect = tester.getRect(find.byType(ProductDock));
      final markRect = tester.getRect(
        find.byKey(const ValueKey('body-landmark')),
      );
      expect(dockRect.overlaps(markRect), isFalse);
      expect(dockRect.top, greaterThan(markRect.bottom));
    },
  );
}
