import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/chrome/loop_focus.dart';

import '_dock_test_helpers.dart';

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
      await pumpDock(tester, focus: focus);

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
      await pumpDock(tester, focus: focus, initialLocation: '/dashboard');

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
    await pumpDock(tester, focus: focus);

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
      await pumpDock(
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
    await pumpDock(tester, focus: focus, initialLocation: '/prd/prd-1');

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
      await pumpDock(tester, focus: focus);

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
    'a deep-linked PRD route enables Version and Commit without a seeded focus',
    (tester) async {
      await tester.binding.setSurfaceSize(const Size(1400, 900));
      addTearDown(() => tester.binding.setSurfaceSize(null));

      // A refresh or deep link lands on /prd/<id> before any page has
      // seeded LoopFocus; the dock must still read the PRD id from the
      // route, not leave the highlighted Version item inert.
      final focus = LoopFocus();
      await pumpDock(tester, focus: focus, initialLocation: '/prd/prd-1');

      expect(find.text('Version prd-1'), findsOneWidget);
      expect(disabledButtons(), findsNWidgets(4));

      await tester.tap(find.text('Version'));
      await tester.pumpAndSettle();

      expect(find.text('Commit prd-1'), findsOneWidget);
    },
  );
}
