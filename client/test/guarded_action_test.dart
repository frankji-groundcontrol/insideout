import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/api/errors.dart';
import 'package:insideout/session/appearance.dart';
import 'package:insideout/widgets/guarded_action.dart';
import 'package:provider/provider.dart';

// Register 4.1 coverage: a failing write must surface a localized error
// instead of failing silently, and report false so callers keep their
// input; a successful write reports true so callers reload.
Future<void> _pump(
  WidgetTester tester,
  Future<void> Function() write,
  void Function(bool) report,
) async {
  await tester.pumpWidget(
    MultiProvider(
      providers: [
        ChangeNotifierProvider(
          create: (_) =>
              Appearance(store: MemorySettingsStore(), locale: 'en-US'),
        ),
      ],
      child: MaterialApp(
        home: Scaffold(
          body: Builder(
            builder: (context) => FilledButton(
              onPressed: () async => report(await guardWrite(context, write)),
              child: const Text('go'),
            ),
          ),
        ),
      ),
    ),
  );
  await tester.pumpAndSettle();
}

void main() {
  testWidgets('a failing write shows the localized error and reports false', (
    tester,
  ) async {
    bool? outcome;
    await _pump(
      tester,
      () async => throw ApiException('boom', status: 500),
      (ok) => outcome = ok,
    );

    await tester.tap(find.text('go'));
    await tester.pumpAndSettle();

    expect(outcome, isFalse);
    expect(
      find.text('Save failed — Server hiccup — try again'),
      findsOneWidget,
    );
  });

  testWidgets('a successful write reports true and shows no SnackBar', (
    tester,
  ) async {
    bool? outcome;
    await _pump(tester, () async {}, (ok) => outcome = ok);

    await tester.tap(find.text('go'));
    await tester.pumpAndSettle();

    expect(outcome, isTrue);
    expect(find.byType(SnackBar), findsNothing);
  });

  // The deferred roadmap-copy issue: an AI build/expand answered with
  // APP_THROTTLE or CIRCUIT_OPEN must surface the coach's specific copy
  // with its countdown, not guardWrite's generic save-failed snackbar.
  testWidgets('an APP_THROTTLE write shows the coach rate-limit countdown', (
    tester,
  ) async {
    await _pump(
      tester,
      () async =>
          throw ApiException('throttled', status: 429, code: 'APP_THROTTLE'),
      (_) {},
    );

    await tester.tap(find.text('go'));
    await tester.pumpAndSettle();

    expect(find.text('Rate limited. Retry in 60s...'), findsOneWidget);
    expect(find.textContaining('Save failed'), findsNothing);
  });

  testWidgets(
    'a CIRCUIT_OPEN write says the AI service is down, with a timer',
    (tester) async {
      await _pump(
        tester,
        () async => throw ApiException(
          'open',
          status: 503,
          code: 'CIRCUIT_OPEN',
          retryAfterSeconds: 30,
        ),
        (_) {},
      );

      await tester.tap(find.text('go'));
      await tester.pumpAndSettle();

      expect(
        find.text('AI service temporarily unavailable. Retry in 30s...'),
        findsOneWidget,
      );
      expect(find.textContaining('Save failed'), findsNothing);
    },
  );
}
