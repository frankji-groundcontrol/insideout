import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/api/client.dart';
import 'package:insideout/api/models.dart';
import 'package:insideout/chrome/loop_focus.dart';
import 'package:insideout/features/prd/versions_page.dart';
import 'package:insideout/session/appearance.dart';
import 'package:insideout/session/session.dart';
import 'package:provider/provider.dart';

/// What a committed version's card shows. PRODUCT.md says a Commit records its
/// name, audience, change summary, unresolved validation items and a Diff, and
/// that a Decision Log goes with it. The first versions page listed the carried
/// items, one line per changed section and the decision note; the Cobalt
/// restyle kept only the diff counts and left the other strings unused. The
/// server stores each carried gap in English as a section, a colon, the reason
/// and "(carried as open question)", so the card shows those like the gaps
/// they came from and anything a caller named as stored.
class _FakeApi extends ApiClient {
  _FakeApi(this.commit)
    : super(
        baseUrl: 'http://127.0.0.1:9/api/v1',
        readAccess: () async => null,
        readRefresh: () async => null,
        writeTokens: (a, r) async {},
        clearTokens: () async {},
      );

  /// Fields of the one commit, laid over a bare "Board review" commit.
  final Map<String, Object?> commit;

  @override
  Future<PrdReadiness> prdReadiness(String id) async => PrdReadiness.fromJson({
    'audiences': {
      'decision': {
        'ready': true,
        'gaps': const <Map<String, String>>[],
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
      ...commit,
    }),
  ];
}

const _carried =
    'background: the why-this, why-now argument a decision maker needs '
    '(carried as open question)';
const _named = 'ask legal about the data retention clause';
const _note = 'Go B2B first; the consumer plan is the alternative not taken';
const _sections = {
  'diff': {
    'counts': {'added': 1, 'changed': 1, 'removed': 0},
    // The server records only the sections that differ, each as
    // `{change: ...}` plus a byte count (prdcommit.Diff).
    'sections': {
      'background': {'change': 'changed', 'bytes_delta': 120},
      'goals': {'change': 'added', 'bytes': 300},
    },
  },
};

Future<void> _pump(
  WidgetTester tester,
  String locale,
  Map<String, Object?> commit,
) async {
  final session = Session()
    ..ready = true
    ..api = _FakeApi(commit);
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
  // The commit loaded, so a failure below means "not shown", not a page error.
  expect(find.text('Board review'), findsWidgets);
}

void main() {
  testWidgets('the commit card lists the open items the version carried', (
    tester,
  ) async {
    await _pump(tester, 'en-US', {
      'unresolved': [_carried, _named],
    });

    expect(find.text('Carried open items'), findsOneWidget);
    expect(
      find.textContaining(
        'Background & Problem Statement — the why-this, why-now argument '
        'a decision maker needs',
      ),
      findsOneWidget,
    );
    // The stored suffix is bookkeeping, not something to read.
    expect(find.textContaining('carried as open question'), findsNothing);
    expect(find.textContaining(_named), findsOneWidget);
  });

  testWidgets('the zh commit card shows carried gaps in Chinese', (
    tester,
  ) async {
    await _pump(tester, 'zh-CN', {
      'unresolved': [_carried, _named],
    });

    expect(find.text('携带的未决事项'), findsOneWidget);
    expect(find.textContaining('决策者需要'), findsOneWidget);
    expect(find.textContaining('decision maker needs'), findsNothing);
    // An item a caller named has no translation: shown as stored.
    expect(find.textContaining(_named), findsOneWidget);
  });

  testWidgets('the commit card names each changed section and the decision', (
    tester,
  ) async {
    await _pump(tester, 'en-US', {..._sections, 'decisionNote': _note});

    expect(
      find.text('· Background & Problem Statement — changed'),
      findsOneWidget,
    );
    expect(find.text('· Goals & Success Metrics — added'), findsOneWidget);
    expect(find.text('Decision note'), findsOneWidget);
    expect(find.text(_note), findsOneWidget);
  });

  testWidgets('the zh commit card names the sections and the note in Chinese', (
    tester,
  ) async {
    await _pump(tester, 'zh-CN', {..._sections, 'decisionNote': _note});

    expect(find.text('· 背景与问题陈述 — 修改'), findsOneWidget);
    expect(find.text('· 目标与成功指标 — 新增'), findsOneWidget);
    expect(find.text('决策备注'), findsOneWidget);
    expect(find.text(_note), findsOneWidget);
  });

  testWidgets('a commit that recorded none of these shows none of them', (
    tester,
  ) async {
    await _pump(tester, 'en-US', const {});

    expect(find.text('Carried open items'), findsNothing);
    expect(find.text('Decision note'), findsNothing);
    expect(find.textContaining(' — changed'), findsNothing);
    expect(find.textContaining(' — added'), findsNothing);
  });
}
