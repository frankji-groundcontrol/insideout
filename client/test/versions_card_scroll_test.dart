import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/api/client.dart';
import 'package:insideout/api/models.dart';
import 'package:insideout/chrome/loop_focus.dart';
import 'package:insideout/features/prd/versions_page.dart';
import 'package:insideout/session/appearance.dart';
import 'package:insideout/session/session.dart';
import 'package:provider/provider.dart';

/// Pins the in-card scroll of the fanned commit stack. The front card sits in
/// a pinned 580px Stack, so a long readiness gap list must scroll inside the
/// card instead of overflowing it or pushing the Commit action out of reach —
/// a later split of versions_stack.dart that drops the scroll must fail here,
/// not in a web walk (docs/learning/2026-10-10-test-the-width-you-ship-for.md:
/// assert the offset or the position, not just "no exception").
class _FakeApi extends ApiClient {
  _FakeApi()
    : super(
        baseUrl: 'http://127.0.0.1:9/api/v1',
        readAccess: () async => null,
        readRefresh: () async => null,
        writeTokens: (a, r) async {},
        clearTokens: () async {},
      );

  @override
  Future<PrdReadiness> prdReadiness(String id) async => PrdReadiness.fromJson({
    'audiences': {
      'decision': {
        'ready': false,
        // Far more gaps than the fixed-height card can show at once. The
        // section is unknown to the l10n tables on purpose: gapReason then
        // keeps the server's text, so every row carries its own marker.
        'gaps': [
          for (var i = 1; i <= 12; i++)
            {
              'section': 'appendix',
              'priority': 'must_clarify_now',
              'reason':
                  'decision gap $i of 12: the why-this, why-now argument a '
                  'decision maker needs before the investment can be decided '
                  'on, with room to spare',
            },
        ],
        'carryIntoCommit': const <String>[],
      },
    },
  });

  @override
  Future<List<PrdCommit>> listPrdCommits(String id) async => [
    PrdCommit.fromJson({
      'id': 'c1',
      'revision': 3,
      'name': 'Board review',
      'primaryAudience': 'decision',
      'createdAt': '2026-10-05T00:00:00Z',
      'diff': {
        'counts': {'added': 1, 'changed': 1, 'removed': 0},
        'sections': {},
      },
    }),
  ];
}

void main() {
  testWidgets('a long gap list scrolls inside the 580px front card', (
    tester,
  ) async {
    tester.view.physicalSize = const Size(1440, 900);
    tester.view.devicePixelRatio = 1.0;
    addTearDown(tester.view.reset);
    final session = Session()
      ..ready = true
      ..api = _FakeApi();
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

    // The card holds the gaps without overflowing its 580px box.
    expect(tester.takeException(), isNull);
    final scrollable = find.descendant(
      of: find.byType(SingleChildScrollView),
      matching: find.byType(Scrollable),
    );
    final position = tester.state<ScrollableState>(scrollable).position;
    expect(position.maxScrollExtent, greaterThan(0));

    // The last gap starts below the card's fold; dragging reaches it.
    final lastGap = find.textContaining('decision gap 12 of 12');
    await tester.drag(scrollable, const Offset(0, -4000));
    await tester.pumpAndSettle();
    expect(position.pixels, position.maxScrollExtent);
    expect(
      tester.getBottomLeft(lastGap).dy,
      lessThanOrEqualTo(tester.getBottomLeft(scrollable).dy),
    );

    // Commit stays reachable whatever the gap list did to the card.
    await tester.tap(find.byKey(const Key('commit-now')));
    await tester.pumpAndSettle();
    expect(find.text('Commit this version'), findsOneWidget);
  });
}
