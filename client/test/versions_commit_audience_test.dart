import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/api/client.dart';
import 'package:insideout/api/models.dart';
import 'package:insideout/chrome/loop_focus.dart';
import 'package:insideout/features/prd/versions_page.dart';
import 'package:insideout/session/appearance.dart';
import 'package:insideout/session/session.dart';
import 'package:provider/provider.dart';

// Decision carries one open item and Management two, so the count the dialog
// shows and the list the commit sends each say which audience they were worked
// out for.
const _decisionCarry = ['background: decision item (carried as open question)'];
const _managementCarry = [
  'goals: management item one (carried as open question)',
  'risks: management item two (carried as open question)',
];

class _FakeApi extends ApiClient {
  _FakeApi()
    : super(
        baseUrl: 'http://127.0.0.1:9/api/v1',
        readAccess: () async => null,
        readRefresh: () async => null,
        writeTokens: (a, r) async {},
        clearTokens: () async {},
      );

  String? sentAudience;
  List<String>? sentUnresolved;

  Map<String, dynamic> _audience(List<String> carry) => {
    'ready': false,
    'gaps': [
      {
        'section': 'background',
        'priority': 'must_clarify_now',
        'reason': 'a gap, so the page has something to list',
      },
    ],
    'carryIntoCommit': carry,
  };

  @override
  Future<PrdReadiness> prdReadiness(String id) async => PrdReadiness.fromJson({
    'audiences': {
      'decision': _audience(_decisionCarry),
      'management': _audience(_managementCarry),
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
    sentAudience = audience;
    sentUnresolved = unresolved;
    return PrdCommit.fromJson({
      'id': 'c1',
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

/// Opens the Commit dialog on the Decision tab and switches its audience
/// select to Management.
Future<_FakeApi> _openAndChooseManagement(WidgetTester tester) async {
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

  await tester.tap(find.byKey(const Key('commit-now')));
  await tester.pumpAndSettle();
  // The dialog opens on the page's audience, with that audience's count.
  expect(find.textContaining('1 open item(s) will be carried'), findsOneWidget);

  await tester.tap(find.byType(DropdownButton<String>));
  await tester.pumpAndSettle();
  await tester.tap(find.text('Management').last);
  await tester.pumpAndSettle();
  return fake;
}

void main() {
  testWidgets('the carry notice counts the audience chosen in the dialog', (
    tester,
  ) async {
    await _openAndChooseManagement(tester);

    expect(
      find.textContaining('2 open item(s) will be carried'),
      findsOneWidget,
    );
    expect(find.textContaining('1 open item(s) will be carried'), findsNothing);
  });

  testWidgets('the commit sends the open items of the audience it is for', (
    tester,
  ) async {
    final fake = await _openAndChooseManagement(tester);

    await tester.enterText(find.byType(TextField).first, 'For the board');
    await tester.tap(find.byKey(const Key('commit-confirm')));
    await tester.pumpAndSettle();

    expect(fake.sentAudience, 'management');
    // Decision's item belongs to a different commit; the server stores the
    // union of what it is sent and what it derives, so sending it labels the
    // management card with a reason written for another audience.
    expect(fake.sentUnresolved, _managementCarry);
  });
}
