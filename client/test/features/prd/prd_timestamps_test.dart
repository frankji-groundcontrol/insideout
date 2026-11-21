import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/api/models.dart';
import 'package:insideout/features/prd/prd_desk.dart';
import 'package:insideout/features/prd/revisions_table.dart';
import 'package:insideout/l10n/l10n.dart';
import 'package:insideout/session/appearance.dart';
import 'package:insideout/theme/cobalt.dart';
import 'package:insideout/theme/cobalt_theme.dart';
import 'package:provider/provider.dart';

// The PRD desk's version line and the snapshot rows printed the API's raw ISO
// string ("2026-10-05T08:12:44.123456Z") where every other page shows the
// app's timeAgo: relative for the last two weeks, a plain date after that,
// localized either way. Rendered output in both locales.
String _ago(Duration d) => DateTime.now().toUtc().subtract(d).toIso8601String();

// Older than timeAgo's two-week window, so it falls back to the date.
const _old = '2026-01-02T03:04:05Z';

Future<void> _pump(WidgetTester tester, Widget child, String locale) async {
  await tester.binding.setSurfaceSize(const Size(1400, 1000));
  addTearDown(() => tester.binding.setSurfaceSize(null));
  await tester.pumpWidget(
    ChangeNotifierProvider(
      create: (_) => Appearance(store: MemorySettingsStore(), locale: locale),
      child: MaterialApp(
        theme: cobaltTheme(Cobalt.light, brightness: Brightness.light),
        home: Scaffold(body: child),
      ),
    ),
  );
  await tester.pumpAndSettle();
}

Widget _desk(Prd prd) {
  final title = TextEditingController(text: prd.title);
  final sections = {
    for (final key in prdSectionKeys) key: TextEditingController(),
  };
  return PrdDeskColumn(
    prdId: prd.id,
    prd: prd,
    titleCtrl: title,
    sectionCtrls: sections,
    onSave: () {},
    onSaveTitle: () {},
    onRequestReview: () {},
    onSnapshot: () {},
    onBuild: () {},
    onStatus: (_) {},
  );
}

Prd _prd(String updatedAt) => Prd(
  id: 'prd-1',
  workspaceId: 'ws-1',
  title: 'Ship a thing',
  sections: const {},
  status: 'draft',
  currentRevision: 4,
  updatedAt: updatedAt,
);

void main() {
  for (final locale in const ['en-US', 'zh-CN']) {
    String t(String key, [Map<String, Object?>? args]) =>
        lookup(locale, key, args);

    testWidgets('the desk version line shows relative time ($locale)', (
      tester,
    ) async {
      final updated = _ago(const Duration(days: 3));
      await _pump(tester, _desk(_prd(updated)), locale);

      expect(find.text('v4 · ${t('time.daysAgo', {'n': 3})}'), findsOneWidget);
      expect(find.textContaining(updated), findsNothing);
    });

    testWidgets('the desk version line falls back to the date, not the raw '
        'ISO string ($locale)', (tester) async {
      await _pump(tester, _desk(_prd(_old)), locale);

      expect(find.text('v4 · 2026-01-02'), findsOneWidget);
      expect(find.textContaining(_old), findsNothing);
    });

    testWidgets('snapshot rows without a note show relative time ($locale)', (
      tester,
    ) async {
      final recent = _ago(const Duration(hours: 5));
      final week = _ago(const Duration(days: 9));
      final revisions = [
        PrdRevision(id: 'r3', revision: 3, createdAt: recent),
        // An empty note is no note.
        PrdRevision(id: 'r2', revision: 2, createdAt: week, note: ''),
        PrdRevision(id: 'r1', revision: 1, createdAt: _old),
        PrdRevision(
          id: 'r0',
          revision: 0,
          createdAt: _ago(const Duration(days: 2)),
          note: 'kept the note',
        ),
      ];
      await _pump(
        tester,
        RevisionsTable(
          revisions: revisions,
          selected: revisions.first,
          onSelect: (_) {},
          onSnapshot: () {},
        ),
        locale,
      );

      expect(find.text(t('time.hoursAgo', {'n': 5})), findsOneWidget);
      expect(find.text(t('time.daysAgo', {'n': 9})), findsOneWidget);
      expect(find.text('2026-01-02'), findsOneWidget);
      // A real note still wins, and its timestamp is not printed beside it.
      expect(find.text('kept the note'), findsOneWidget);
      for (final r in revisions) {
        expect(find.textContaining(r.createdAt), findsNothing);
      }
    });
  }
}
