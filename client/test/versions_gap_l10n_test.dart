import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/api/client.dart';
import 'package:insideout/api/models.dart';
import 'package:insideout/chrome/loop_focus.dart';
import 'package:insideout/features/prd/versions_page.dart';
import 'package:insideout/session/appearance.dart';
import 'package:insideout/session/session.dart';
import 'package:provider/provider.dart';

/// The commit card lists each audience's gaps as "<section> — <reason>". The
/// reason is the server's English sentence, so the zh card read "背景与问题陈述
/// — the why-this, why-now argument a decision maker needs". The reasons the
/// server sends (readiness.go) are a fixed table keyed by audience and
/// section, so the client translates them by that key.
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
        'gaps': [
          {
            'section': 'background',
            'priority': 'must_clarify_now',
            'reason': 'the why-this, why-now argument a decision maker needs',
          },
          {
            'section': 'goals',
            'priority': 'should_clarify_this_version',
            'reason': 'the investment and return being decided on',
          },
          {
            'section': 'stories',
            'priority': 'validate_later',
            'reason': 'useful eventually; not required for this audience',
          },
          // A pair the client has no translation for keeps the server text.
          {
            'section': 'risks',
            'priority': 'must_clarify_now',
            'reason': 'a brand new server-side reason',
          },
        ],
        'carryIntoCommit': const <String>[],
      },
    },
  });

  @override
  Future<List<PrdCommit>> listPrdCommits(String id) async => const [];
}

Future<void> _pump(WidgetTester tester, String locale) async {
  final session = Session()
    ..ready = true
    ..api = _FakeApi();
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
  testWidgets('the zh commit card shows the gap reasons in Chinese', (
    tester,
  ) async {
    await _pump(tester, 'zh-CN');

    expect(find.textContaining('decision maker needs'), findsNothing);
    expect(find.textContaining('being decided on'), findsNothing);
    expect(find.textContaining('useful eventually'), findsNothing);
    expect(find.textContaining('决策者需要'), findsOneWidget);
    expect(find.textContaining('待决策的投入与回报'), findsOneWidget);
    expect(find.textContaining('迟早有用'), findsOneWidget);
    // No translation for it: the server's sentence is better than a key.
    expect(
      find.textContaining('a brand new server-side reason'),
      findsOneWidget,
    );
    expect(find.textContaining('versions.gap.'), findsNothing);
  });

  testWidgets('the en commit card keeps the server reasons', (tester) async {
    await _pump(tester, 'en-US');

    expect(find.textContaining('decision maker needs'), findsOneWidget);
    expect(find.textContaining('being decided on'), findsOneWidget);
    expect(find.textContaining('useful eventually'), findsOneWidget);
    expect(
      find.textContaining('a brand new server-side reason'),
      findsOneWidget,
    );
  });
}
