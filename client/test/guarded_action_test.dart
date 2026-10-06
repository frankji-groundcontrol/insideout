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
}
