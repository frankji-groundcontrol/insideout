import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/api/client.dart';
import 'package:insideout/features/prd/export_page.dart';
import 'package:insideout/session/appearance.dart';
import 'package:insideout/session/session.dart';
import 'package:insideout/theme/block_button.dart';
import 'package:provider/provider.dart';

// The shipped default locale is zh-CN, yet the export page had no widget
// test in it at all (export_format_test.dart covers only the query helper).
// Pump it in zh-CN at phone width and hold it to the strings the zh table
// carries: lookup silently falls back to English (l10n.dart), so a missing
// key must fail here, not in front of a Chinese reader.
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
  Future<String> exportPrd(
    String id,
    String format, {
    String? audience,
  }) async => '# 中文 PRD\n\n正文';
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
      ],
      child: const MaterialApp(home: ExportPage(prdId: 'prd-1')),
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
  testWidgets('the export page speaks zh-CN and stays on the phone', (
    tester,
  ) async {
    await _pump(tester);
    expect(tester.takeException(), isNull);

    // Distinctive zh strings; an English fallback would fail these.
    expect(find.text('导出'), findsOneWidget); // prd.exportTitle
    expect(find.text('下载 Markdown'), findsOneWidget); // format option
    expect(find.text('打印 / 另存为 PDF'), findsOneWidget); // prd.exportPrint
    expect(find.text('读者视图（可选）'), findsOneWidget); // prd.exportAudience
    expect(find.text('完整文档'), findsOneWidget); // prd.exportFull

    // The run button is the page's primary action.
    _expectOnScreen(tester.getRect(find.widgetWithText(BlockButton, '生成预览')));

    await tester.tap(find.widgetWithText(BlockButton, '生成预览'));
    await tester.pumpAndSettle();
    expect(find.textContaining('# 中文 PRD'), findsOneWidget);
    expect(find.text('下载'), findsOneWidget); // common.download
    expect(tester.takeException(), isNull);
  });
}
