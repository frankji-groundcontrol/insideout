import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/api/client.dart';
import 'package:insideout/api/models.dart';
import 'package:insideout/chrome/loop_focus.dart';
import 'package:insideout/features/prd/audience_view_page.dart';
import 'package:insideout/session/appearance.dart';
import 'package:insideout/session/session.dart';
import 'package:insideout/theme/block_button.dart';
import 'package:provider/provider.dart';

// The shipped default locale is zh-CN, yet the audience view was only ever
// widget-tested in en-US (audience_view_page_test.dart). Pump it in zh-CN at
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
  Future<Map<String, dynamic>> prdView(String id, String audience) async => {
    'prdId': id,
    'title': '读者视图中文走查',
    'projection': {
      'audience': 'decision',
      'purpose': '为什么是它、为什么是现在',
      'sections': [
        {'section': 'background', 'why': '决策者需要的论证'},
      ],
    },
    'readiness': {
      'gaps': [
        {
          'section': 'risks',
          'priority': 'must_clarify_now',
          'reason': '现在就要做的决定',
        },
      ],
    },
    'latestCommit': {'revision': 3, 'name': '第一次决策稿'},
  };

  @override
  Future<Prd> getPrd(String id) async => Prd(
    id: id,
    workspaceId: 'ws1',
    title: '读者视图中文走查',
    sections: const {'background': '问题是真实的。'},
    status: 'draft',
    currentRevision: 3,
    updatedAt: '2026-08-22T00:00:00Z',
  );
}

// 420 rather than 390 because the Ahem test font is wider; both sit below
// Cobalt.bpNarrow (docs/learning/2026-10-10-test-the-width-you-ship-for.md).
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
        ChangeNotifierProvider(create: (_) => LoopFocus()),
      ],
      child: const MaterialApp(home: AudienceViewPage(prdId: 'p1')),
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
  testWidgets('the audience view speaks zh-CN and stays on the phone', (
    tester,
  ) async {
    await _pump(tester);
    expect(tester.takeException(), isNull);

    // Distinctive zh strings; an English fallback would fail these.
    expect(find.text('读者视图'), findsOneWidget); // view.title
    expect(find.text('背景与问题陈述'), findsOneWidget); // prd.sections.background
    expect(find.textContaining('为什么读这一节'), findsOneWidget); // view.whyYouRead
    expect(find.text('最新版本：v3 · 第一次决策稿'), findsOneWidget);

    // The first section card rides the top of the reading order.
    _expectOnScreen(tester.getRect(find.text('背景与问题陈述')));

    // The gap list and the commit jump sit below the fold on a phone.
    final scrollable = find.descendant(
      of: find.byType(ListView),
      matching: find.byType(Scrollable),
    );
    await tester.scrollUntilVisible(
      find.text('该读者的未决缺口'),
      300,
      scrollable: scrollable,
    );
    await tester.ensureVisible(find.text('该读者的未决缺口'));
    await tester.pumpAndSettle();
    expect(find.text('该读者的未决缺口'), findsOneWidget); // view.gapsTitle

    await tester.scrollUntilVisible(
      find.widgetWithText(BlockButton, '版本'),
      300,
      scrollable: scrollable,
    );
    await tester.ensureVisible(find.widgetWithText(BlockButton, '版本'));
    await tester.pumpAndSettle();
    _expectOnScreen(tester.getRect(find.widgetWithText(BlockButton, '版本')));
    expect(tester.takeException(), isNull);
  });
}
