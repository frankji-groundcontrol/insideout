import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/api/client.dart';
import 'package:insideout/api/errors.dart';
import 'package:insideout/api/models.dart';
import 'package:insideout/chrome/loop_focus.dart';
import 'package:insideout/features/prd/versions_page.dart';
import 'package:insideout/session/appearance.dart';
import 'package:insideout/session/session.dart';
import 'package:provider/provider.dart';

class _FakeApi extends ApiClient {
  _FakeApi({this.failCommit = false})
    : super(
        baseUrl: 'http://127.0.0.1:9/api/v1',
        readAccess: () async => null,
        readRefresh: () async => null,
        writeTokens: (a, r) async {},
        clearTokens: () async {},
      );

  final committedNames = <String>[];
  final bool failCommit;

  @override
  Future<PrdReadiness> prdReadiness(String id) async => PrdReadiness.fromJson({
    'audiences': {
      'decision': {
        'ready': false,
        'gaps': [
          {
            'section': 'background',
            'priority': 'must_clarify_now',
            'reason': 'the why-this argument',
          },
          {
            'section': 'goals',
            'priority': 'should_clarify_this_version',
            'reason': 'the investment',
          },
        ],
        'carryIntoCommit': [
          'background: the why-this argument (carried as open question)',
        ],
      },
    },
  });

  @override
  Future<List<PrdCommit>> listPrdCommits(String id) async => committedNames
      .asMap()
      .map(
        (i, n) => MapEntry(
          i,
          PrdCommit.fromJson({
            'id': 'c$i',
            'revision': 2 + i,
            'name': n,
            'primaryAudience': 'decision',
            'createdAt': '2026-08-21T00:00:00Z',
            'unresolved': [
              'background: the why-this argument (carried as open question)',
            ],
            'diff': {
              'counts': {'added': 8, 'changed': 0, 'removed': 0},
              'sections': {},
            },
          }),
        ),
      )
      .values
      .toList();

  @override
  Future<PrdCommit> commitPrd(
    String id, {
    required String name,
    required String audience,
    String summary = '',
    List<String>? unresolved,
    String note = '',
  }) async {
    if (failCommit) throw ApiException('boom', status: 500);
    committedNames.insert(0, name);
    return PrdCommit.fromJson({
      'id': 'c${committedNames.length}',
      'revision': 2,
      'name': name,
      'primaryAudience': audience,
      'createdAt': '2026-08-21T00:00:00Z',
      'unresolved': unresolved ?? const [],
      'diff': {
        'counts': {'added': 8, 'changed': 0, 'removed': 0},
        'sections': {},
      },
    });
  }
}

void main() {
  testWidgets('versions page shows gaps and performs a carrying commit', (
    tester,
  ) async {
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
                Appearance(store: MemorySettingsStore(), locale: 'en-US'),
          ),
          ChangeNotifierProvider(create: (_) => LoopFocus()),
        ],
        child: const MaterialApp(home: VersionsPage(prdId: 'p1')),
      ),
    );
    await tester.pumpAndSettle();

    // Readiness disclosure: chips, gaps with reasons, not-ready wording.
    expect(find.text('Decision'), findsWidgets);
    expect(find.textContaining('Gaps for this audience'), findsOneWidget);
    expect(find.textContaining('why-this argument'), findsWidgets);

    // Empty version list with the form-now affordance.
    expect(find.textContaining('No committed versions yet'), findsOneWidget);

    // Commit: open the dialog, name it, confirm.
    await tester.tap(find.byKey(const Key('commit-now')));
    await tester.pumpAndSettle();
    expect(find.textContaining('open item(s) will be carried'), findsOneWidget);
    await tester.enterText(find.byType(TextField).first, 'First web version');
    await tester.tap(find.byKey(const Key('commit-confirm')));
    await tester.pumpAndSettle();

    expect(fake.committedNames, ['First web version']);
    expect(
      find.text('First web version'),
      findsWidgets,
    ); // list entry + detail headline
  });

  testWidgets(
    'a failed version commit shows the localized error and does not count as success',
    (tester) async {
      final session = Session();
      session.ready = true;
      final fake = _FakeApi(failCommit: true);
      session.api = fake;
      await tester.binding.setSurfaceSize(const Size(1400, 900));
      addTearDown(() => tester.binding.setSurfaceSize(null));

      await tester.pumpWidget(
        MultiProvider(
          providers: [
            ChangeNotifierProvider.value(value: session),
            ChangeNotifierProvider(
              create: (_) =>
                  Appearance(store: MemorySettingsStore(), locale: 'en-US'),
            ),
            ChangeNotifierProvider(create: (_) => LoopFocus()),
          ],
          child: const MaterialApp(home: VersionsPage(prdId: 'p1')),
        ),
      );
      await tester.pumpAndSettle();

      await tester.tap(find.byKey(const Key('commit-now')));
      await tester.pumpAndSettle();
      await tester.enterText(find.byType(TextField).first, 'Broken version');
      await tester.tap(find.byKey(const Key('commit-confirm')));
      await tester.pumpAndSettle();

      expect(
        find.text('Save failed — Server hiccup — try again'),
        findsOneWidget,
      );
      expect(fake.committedNames, isEmpty);
      expect(find.text('Broken version'), findsNothing);
      expect(find.textContaining('No committed versions yet'), findsOneWidget);
    },
  );
}
