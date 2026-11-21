import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

import '_dock_test_helpers.dart';

Finder disabledButtons() => find.byWidgetPredicate(
  (w) =>
      w is Semantics &&
      w.properties.button == true &&
      w.properties.enabled == false,
);

// FULL-04: on the ideas wall the Capture/Ideas items and on the versions
// route the Version/Commit items were enabled-but-inert — a tap landed on
// the early return at product_dock.dart and nothing happened. The wall now
// gets an ideasPulse the way the evidence page gets evidencePulse, and the
// versions route renders its two items disabled.
void main() {
  testWidgets(
    'a capture tap on the ideas wall pulses the capture bar, not a dead tap',
    (tester) async {
      await tester.binding.setSurfaceSize(const Size(1400, 900));
      addTearDown(() => tester.binding.setSurfaceSize(null));

      final focus = warmFocus();
      await pumpDock(
        tester,
        focus: focus,
        initialLocation: '/workspace/ws-1/ideas',
      );

      // The destination is on screen: the tap must ask the wall to
      // scroll-highlight its capture bar instead of silently returning.
      await tester.tap(find.text('Capture'));
      await tester.pumpAndSettle();
      expect(focus.ideasPulse, 1);

      // The Ideas item shares the wall as its destination; same affordance.
      await tester.tap(find.text('Ideas'));
      await tester.pumpAndSettle();
      expect(focus.ideasPulse, 2);
      expect(find.text('Ideas ws-1'), findsOneWidget);
    },
  );

  testWidgets(
    'version and commit render disabled while the versions route is on screen',
    (tester) async {
      await tester.binding.setSurfaceSize(const Size(1400, 900));
      addTearDown(() => tester.binding.setSurfaceSize(null));

      final focus = warmFocus();
      await pumpDock(
        tester,
        focus: focus,
        initialLocation: '/prd/prd-1/versions',
      );

      // Their destination is already on screen, so a tap could only be a
      // dead one: both items render disabled instead of enabled-but-inert.
      expect(
        find.descendant(of: disabledButtons(), matching: find.text('Version')),
        findsOneWidget,
      );
      expect(
        find.descendant(of: disabledButtons(), matching: find.text('Commit')),
        findsOneWidget,
      );

      // A disabled tap stays put without crashing...
      await tester.tap(find.text('Version'));
      await tester.pumpAndSettle();
      expect(find.text('Commit prd-1'), findsOneWidget);

      // ...while a step whose destination is elsewhere still navigates.
      await tester.tap(find.text('Roadmap'));
      await tester.pumpAndSettle();
      expect(find.text('Roadmap proj-1'), findsOneWidget);
    },
  );
}
