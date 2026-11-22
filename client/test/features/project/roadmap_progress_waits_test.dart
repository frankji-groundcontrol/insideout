import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/api/models.dart';
import 'package:insideout/features/project/roadmap_progress_view.dart';
import 'package:insideout/session/appearance.dart';
import 'package:provider/provider.dart';

// Progress Next ranks ready work first (2026-11-22 plan, S2); a waiting item
// says what it waits on, from the server's waitsOn.
Future<void> _pump(WidgetTester tester, String locale) async {
  final progress = RoadmapProgress(
    now: const [],
    needsDeadline: const [],
    doneCount: 0,
    next: [
      ProgressItem.fromJson({
        'id': 'me',
        'title': 'Match explanation copy',
        'status': 'pending',
      }),
      ProgressItem.fromJson({
        'id': 'dd',
        'title': 'Daily drop UI',
        'status': 'pending',
        'waitsOn': [
          {'id': 'cs', 'title': 'Compatibility scoring'},
        ],
      }),
    ],
  );
  await tester.pumpWidget(
    ChangeNotifierProvider(
      create: (_) => Appearance(store: MemorySettingsStore(), locale: locale),
      child: MaterialApp(
        home: Scaffold(
          body: RoadmapProgressView(progress: progress, emptyLabel: 'empty'),
        ),
      ),
    ),
  );
  await tester.pumpAndSettle();
}

void main() {
  testWidgets('a waiting Next item names what it waits on', (tester) async {
    await _pump(tester, 'en-US');
    expect(find.text('Waits on: Compatibility scoring'), findsOneWidget);
    expect(find.textContaining('Waits on'), findsOneWidget);
  });

  testWidgets('the waits-on line reads in Chinese', (tester) async {
    await _pump(tester, 'zh-CN');
    expect(find.text('等待：Compatibility scoring'), findsOneWidget);
  });
}
