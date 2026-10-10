import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/features/landing/step_peek.dart';
import 'package:insideout/session/appearance.dart';
import 'package:provider/provider.dart';

void main() {
  testWidgets('the shipped peek shows three commit short hashes, all hex', (
    tester,
  ) async {
    await tester.pumpWidget(
      ChangeNotifierProvider(
        create: (_) =>
            Appearance(store: MemorySettingsStore(), locale: 'en-US'),
        child: const MaterialApp(
          home: Scaffold(body: StepPeek(kind: PeekKind.shipped)),
        ),
      ),
    );

    // The hashes are the only monospace text in the peek.
    final hashes = [
      for (final text in tester.widgetList<Text>(find.byType(Text)))
        if (text.style?.fontFamily == 'monospace') text.data!,
    ];
    expect(hashes, hasLength(3));
    for (final hash in hashes) {
      expect(
        hash,
        matches(RegExp(r'^[0-9a-f]{7}$')),
        reason: 'a git short hash is seven hex digits',
      );
    }
  });
}
