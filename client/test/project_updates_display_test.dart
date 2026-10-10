import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/api/models.dart';
import 'package:insideout/features/project/project_updates_feed.dart';
import 'package:insideout/session/appearance.dart';
import 'package:insideout/theme/cobalt_chip.dart';
import 'package:provider/provider.dart';

// The Evidence timeline shows an update's kind and age in words, the way the
// board card does, not as the server's enum and raw ISO timestamp.

String _ago(Duration d) => DateTime.now().toUtc().subtract(d).toIso8601String();

final _proposalAt = _ago(const Duration(hours: 3));
final _progressAt = _ago(const Duration(days: 2));

Future<void> _pump(WidgetTester tester, String locale) async {
  await tester.pumpWidget(
    ChangeNotifierProvider(
      // A fresh provider per locale: create only runs once per element.
      key: ValueKey(locale),
      create: (_) => Appearance(store: MemorySettingsStore(), locale: locale),
      child: MaterialApp(
        home: Scaffold(
          body: SingleChildScrollView(
            child: ProjectUpdatesFeed(
              projectId: 'p1',
              onChanged: () async {},
              onEdit: (u, content) async {},
              onRemove: (u) async {},
              updates: [
                ProjectUpdate(
                  id: 'u1',
                  kind: 'agent_proposal',
                  content: 'Split the billing branch',
                  createdAt: _proposalAt,
                  authorId: 'a1',
                ),
                ProjectUpdate(
                  id: 'u2',
                  kind: 'progress',
                  content: 'Shipped the importer',
                  createdAt: _progressAt,
                  authorId: 'u1',
                ),
              ],
            ),
          ),
        ),
      ),
    ),
  );
  await tester.pumpAndSettle();
}

void main() {
  testWidgets('update kinds read as words in the active locale', (
    tester,
  ) async {
    await _pump(tester, 'en-US');
    expect(find.text('agent_proposal'), findsNothing);
    expect(find.widgetWithText(ShadeChip, 'Agent proposal'), findsOneWidget);
    expect(find.widgetWithText(ShadeChip, 'Progress'), findsOneWidget);

    await _pump(tester, 'zh-CN');
    // The zh section heading is also 进展, so match the chip, not the text.
    expect(find.widgetWithText(ShadeChip, '智能体提案'), findsOneWidget);
    expect(find.widgetWithText(ShadeChip, '进展'), findsOneWidget);
  });

  testWidgets('update times read as time-ago, not raw ISO', (tester) async {
    await _pump(tester, 'en-US');
    expect(find.text(_proposalAt), findsNothing);
    expect(find.text(_progressAt), findsNothing);
    expect(find.text('3h ago'), findsOneWidget);
    expect(find.text('2d ago'), findsOneWidget);

    await _pump(tester, 'zh-CN');
    expect(find.text('3 小时前'), findsOneWidget);
    expect(find.text('2 天前'), findsOneWidget);
  });
}
