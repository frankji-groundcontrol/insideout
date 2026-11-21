import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/features/prd/coach_column.dart';
import 'package:insideout/session/appearance.dart';
import 'package:provider/provider.dart';

// The strapline under the Coach title is the first line a reader meets on
// the product's core surface, so it must be product copy, not a leftover
// design note ('Large pull-quotes' / '大段引语'). This pins the coaching
// contract in one line, in both catalogs.
Future<void> _pump(WidgetTester tester, {required String locale}) async {
  await tester.pumpWidget(
    MultiProvider(
      providers: [
        ChangeNotifierProvider(
          create: (_) =>
              Appearance(store: MemorySettingsStore(), locale: locale),
        ),
      ],
      child: const MaterialApp(
        home: Scaffold(
          body: CoachHeader(stage: 'clarify', showStage: false, onClose: noop),
        ),
      ),
    ),
  );
  await tester.pumpAndSettle();
}

void noop() {}

void main() {
  for (final (locale, strapline) in [
    ('en-US', 'From fuzzy idea to solid PRD, one conversation at a time.'),
    ('zh-CN', '陪你把模糊的想法，聊成扎实的 PRD。'),
  ]) {
    testWidgets('the masthead strapline speaks to the reader ($locale)', (
      tester,
    ) async {
      await _pump(tester, locale: locale);

      expect(find.text(strapline), findsOneWidget);
    });
  }
}
