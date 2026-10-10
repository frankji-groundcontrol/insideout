import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/api/client.dart';
import 'package:insideout/api/models.dart';
import 'package:insideout/chrome/loop_focus.dart';
import 'package:insideout/features/workspace/ideas_page.dart';
import 'package:insideout/session/appearance.dart';
import 'package:insideout/session/session.dart';
import 'package:insideout/theme/block_button.dart';
import 'package:provider/provider.dart';

// The shipped default locale is zh-CN, yet the ideas wall was only ever
// widget-tested in en-US (ideas_page_test.dart). Pump it in zh-CN at phone
// width and hold it to the strings the zh table actually carries: lookup
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
  Future<List<Idea>> listIdeas(String workspaceId) async => [
    Idea.fromJson({
      'id': 'i1',
      'workspaceId': 'ws1',
      'authorId': 'u1',
      'title': '把导出做成链接',
      'content': '一些内容',
      'status': 'inbox',
      'prdId': null,
      'createdAt': '2026-10-01T00:00:00Z',
      'updatedAt': '2026-10-01T00:00:00Z',
    }),
  ];

  @override
  Future<List<Project>> listProjects(String workspaceId) async => [];
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
      child: const MaterialApp(home: IdeasPage(workspaceId: 'ws1')),
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
  testWidgets('the ideas wall speaks zh-CN and stays on the phone', (
    tester,
  ) async {
    await _pump(tester);
    expect(tester.takeException(), isNull);

    // Distinctive zh strings; an English fallback would fail these.
    expect(find.text('想法'), findsOneWidget); // idea.wallTitle
    expect(
      find.textContaining('仅自己可见——想法默认只有作者能看到'),
      findsOneWidget, // idea.privateHint
    );

    // The capture field's button rides the row's right edge.
    _expectOnScreen(tester.getRect(find.widgetWithText(BlockButton, '记下')));
    expect(find.text('把导出做成链接'), findsOneWidget);
  });
}
