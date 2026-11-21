import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/api/client.dart';
import 'package:insideout/api/models.dart';
import 'package:insideout/chrome/loop_focus.dart';
import 'package:insideout/features/prd/versions_page.dart';
import 'package:insideout/session/appearance.dart';
import 'package:insideout/session/session.dart';
import 'package:provider/provider.dart';

class _FakeApi extends ApiClient {
  _FakeApi()
    : super(
        baseUrl: 'http://127.0.0.1:9/api/v1',
        readAccess: () async => null,
        readRefresh: () async => null,
        writeTokens: (a, r) async {},
        clearTokens: () async {},
      );

  final committedNames = <String>[];

  @override
  Future<PrdReadiness> prdReadiness(String id) async => PrdReadiness.fromJson({
    'audiences': {
      'decision': {
        'ready': false,
        'gaps': [
          {
            'section': 'background',
            'priority': 'must_clarify_now',
            'reason': 'a gap, so the page has something to list',
          },
        ],
        'carryIntoCommit': const <String>[],
      },
    },
  });

  @override
  Future<List<PrdCommit>> listPrdCommits(String id) async => const [];

  @override
  Future<PrdCommit> commitPrd(
    String id, {
    required String name,
    required String audience,
    String summary = '',
    List<String>? unresolved,
    String note = '',
  }) async {
    committedNames.add(name);
    return PrdCommit.fromJson({
      'id': 'c1',
      'revision': 2,
      'name': name,
      'primaryAudience': audience,
      'createdAt': '2026-08-21T00:00:00Z',
      'unresolved': const <String>[],
      'diff': {
        'counts': {'added': 8, 'changed': 0, 'removed': 0},
        'sections': {},
      },
    });
  }
}

/// Opens the Commit dialog on a fresh Versions page and leaves it untouched.
Future<_FakeApi> _openCommitDialog(
  WidgetTester tester, {
  String locale = 'en-US',
}) async {
  final session = Session();
  session.ready = true;
  final fake = _FakeApi();
  session.api = fake;
  await tester.binding.setSurfaceSize(const Size(1400, 900));
  addTearDown(() => tester.binding.setSurfaceSize(null));

  await tester.pumpWidget(
    MultiProvider(
      providers: [
        ChangeNotifierProvider.value(value: session),
        ChangeNotifierProvider(
          create: (_) =>
              Appearance(store: MemorySettingsStore(), locale: locale),
        ),
        ChangeNotifierProvider(create: (_) => LoopFocus()),
      ],
      child: const MaterialApp(home: VersionsPage(prdId: 'p1')),
    ),
  );
  await tester.pumpAndSettle();

  await tester.tap(find.byKey(const Key('commit-now')));
  await tester.pumpAndSettle();
  return fake;
}

void main() {
  for (final (locale, message) in [
    ('en-US', 'Give this version a name'),
    ('zh-CN', '给这个版本起个名字'),
  ]) {
    testWidgets('committing without a name keeps the dialog and says why '
        '($locale)', (tester) async {
      final fake = await _openCommitDialog(tester, locale: locale);

      await tester.tap(find.byKey(const Key('commit-confirm')));
      await tester.pumpAndSettle();

      // Today the dialog closes and versions_page drops the empty result, so
      // the reader sees nothing happen.
      expect(find.byKey(const Key('commit-confirm')), findsOneWidget);
      expect(find.text(message), findsOneWidget);
      expect(fake.committedNames, isEmpty);
    });
  }

  testWidgets('a name of only spaces is no name', (tester) async {
    final fake = await _openCommitDialog(tester);

    await tester.enterText(find.byType(TextField).first, '   ');
    await tester.tap(find.byKey(const Key('commit-confirm')));
    await tester.pumpAndSettle();

    expect(find.byKey(const Key('commit-confirm')), findsOneWidget);
    expect(find.text('Give this version a name'), findsOneWidget);
    expect(fake.committedNames, isEmpty);
  });

  testWidgets('naming the version clears the message and commits it', (
    tester,
  ) async {
    final fake = await _openCommitDialog(tester);
    await tester.tap(find.byKey(const Key('commit-confirm')));
    await tester.pumpAndSettle();
    expect(find.text('Give this version a name'), findsOneWidget);

    await tester.enterText(find.byType(TextField).first, 'Pricing cut');
    await tester.pump();
    expect(find.text('Give this version a name'), findsNothing);

    await tester.tap(find.byKey(const Key('commit-confirm')));
    await tester.pumpAndSettle();

    expect(find.byKey(const Key('commit-confirm')), findsNothing);
    expect(fake.committedNames, ['Pricing cut']);
  });
}
