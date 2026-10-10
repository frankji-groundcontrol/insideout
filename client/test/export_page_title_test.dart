import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/api/client.dart';
import 'package:insideout/features/prd/export_page.dart';
import 'package:insideout/session/appearance.dart';
import 'package:insideout/session/session.dart';
import 'package:provider/provider.dart';

// The export page offers Markdown and Print, so titling it after one option
// ("Download Markdown") mislabels the whole page. The painted title must be
// the neutral Export word; each format button keeps its own label.
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
  }) async => '# Plan';
}

Future<void> _pump(WidgetTester tester, {required String locale}) async {
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
      ],
      child: const MaterialApp(home: ExportPage(prdId: 'prd-1')),
    ),
  );
  await tester.pumpAndSettle();
}

void main() {
  for (final (locale, title, option) in [
    ('en-US', 'Export', 'Download Markdown'),
    ('zh-CN', '导出', '下载 Markdown'),
  ]) {
    testWidgets('the page title names the page, not one option ($locale)', (
      tester,
    ) async {
      await _pump(tester, locale: locale);

      // The AppScaffold title is the page's name; the option label belongs
      // to the format buttons alone, so each appears exactly once.
      expect(find.text(title), findsOneWidget);
      expect(find.text(option), findsOneWidget);
    });
  }
}
