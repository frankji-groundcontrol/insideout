import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/api/client.dart';
import 'package:insideout/api/models.dart';
import 'package:insideout/chrome/loop_focus.dart';
import 'package:insideout/features/prd/versions_page.dart';
import 'package:insideout/session/appearance.dart';
import 'package:insideout/session/session.dart';
import 'package:provider/provider.dart';

/// A committed version records the open items it carried (PRODUCT.md,
/// "Commit"), and the first versions page listed them under the commit. The
/// Cobalt restyle dropped the list and left its heading string unused, so after
/// committing nobody could see what had been carried. The server stores each
/// carried gap as `<section>: <reason> (carried as open question)` in English,
/// so the card shows those like the gaps they came from and anything a caller
/// named as stored.
class _FakeApi extends ApiClient {
  _FakeApi(this.unresolved)
    : super(
        baseUrl: 'http://127.0.0.1:9/api/v1',
        readAccess: () async => null,
        readRefresh: () async => null,
        writeTokens: (a, r) async {},
        clearTokens: () async {},
      );

  final List<String> unresolved;

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
      'unresolved': unresolved,
      'diff': {
        'counts': {'added': 1, 'changed': 0, 'removed': 0},
        'sections': {},
      },
    }),
  ];
}

const _carried =
    'background: the why-this, why-now argument a decision maker needs '
    '(carried as open question)';
const _named = 'ask legal about the data retention clause';

Future<void> _pump(
  WidgetTester tester,
  String locale,
  List<String> unresolved,
) async {
  final session = Session()
    ..ready = true
    ..api = _FakeApi(unresolved);
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
}

void main() {
  testWidgets('the commit card lists the open items the version carried', (
    tester,
  ) async {
    await _pump(tester, 'en-US', const [_carried, _named]);

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
    await _pump(tester, 'zh-CN', const [_carried, _named]);

    expect(find.text('携带的未决事项'), findsOneWidget);
    expect(find.textContaining('决策者需要'), findsOneWidget);
    expect(find.textContaining('decision maker needs'), findsNothing);
    // An item a caller named has no translation: shown as stored.
    expect(find.textContaining(_named), findsOneWidget);
  });

  testWidgets('a commit that carried nothing has no open-items heading', (
    tester,
  ) async {
    await _pump(tester, 'en-US', const []);

    expect(find.text('Board review'), findsWidgets);
    expect(find.text('Carried open items'), findsNothing);
  });
}
