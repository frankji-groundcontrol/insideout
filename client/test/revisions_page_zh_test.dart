import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/api/client.dart';
import 'package:insideout/api/models.dart';
import 'package:insideout/features/prd/revisions_page.dart';
import 'package:insideout/session/appearance.dart';
import 'package:insideout/session/session.dart';
import 'package:insideout/theme/block_button.dart';
import 'package:provider/provider.dart';

// The shipped default locale is zh-CN, yet the revisions page was only ever
// widget-tested in en-US (revisions_page_test.dart). Pump it in zh-CN at
// phone width and hold it to the strings the zh table carries: lookup
// silently falls back to English (l10n.dart), so a missing key must fail
// here, not in front of a Chinese reader.
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
  Future<List<PrdRevision>> listRevisions(String prdId) async => [
    PrdRevision(
      id: 'rev-1',
      revision: 1,
      createdAt: '2026-10-01T00:00:00Z',
      note: '第一次快照',
      sections: const {'problem': '文本'},
    ),
  ];
}

// 420 rather than 390 because the Ahem test font is wider; both sit below
// Cobalt.bpNarrow, where the table stacks instead of splitting.
const _phone = Size(420, 844);

Future<void> _pump(WidgetTester tester) async {
  final session = Session()
    ..ready = true
    ..api = _FakeApi();
  // setSurfaceSize leaves MediaQuery at 800x600; resize the view itself.
  tester.view.devicePixelRatio = 1;
  tester.view.physicalSize = _phone;
  addTearDown(tester.view.reset);
  await tester.pumpWidget(
    MultiProvider(
      providers: [
        ChangeNotifierProvider.value(value: session),
        ChangeNotifierProvider(
          create: (_) =>
              Appearance(store: MemorySettingsStore(), locale: 'zh-CN'),
        ),
      ],
      child: const MaterialApp(home: RevisionsPage(prdId: 'prd-1')),
    ),
  );
  await tester.pumpAndSettle();
}

void _expectOnScreen(Rect rect) {
  expect(rect.left, greaterThanOrEqualTo(0), reason: '$rect leaves the left');
  expect(rect.top, greaterThanOrEqualTo(0), reason: '$rect leaves the top');
  expect(rect.right, lessThanOrEqualTo(_phone.width), reason: '$rect: right');
  expect(
    rect.bottom,
    lessThanOrEqualTo(_phone.height),
    reason: '$rect: bottom',
  );
}

void main() {
  testWidgets('the revisions page speaks zh-CN and stays on the phone', (
    tester,
  ) async {
    await _pump(tester);
    expect(tester.takeException(), isNull);

    // Distinctive zh strings; an English fallback would fail these.
    expect(find.text('版本历史'), findsOneWidget); // prd.revisionsTitle
    expect(find.text('第 1 版'), findsOneWidget); // prd.revisionN
    expect(find.text('第一次快照'), findsOneWidget);

    // The snapshot action rides the table's header row.
    final snapshot = find.byKey(const Key('revisions-snapshot'));
    expect(find.widgetWithText(BlockButton, '保存一个版本快照'), findsOneWidget);
    _expectOnScreen(tester.getRect(snapshot));
  });
}
