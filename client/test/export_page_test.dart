import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/api/client.dart';
import 'package:insideout/api/errors.dart';
import 'package:insideout/chrome/loop_focus.dart';
import 'package:insideout/features/prd/export_page.dart';
import 'package:insideout/session/appearance.dart';
import 'package:insideout/session/session.dart';
import 'package:insideout/theme/block_button.dart';
import 'package:insideout/theme/block_chrome.dart';
import 'package:insideout/theme/block_field.dart';
import 'package:provider/provider.dart';

/// Answers the way the Go export handler does: an audience projection exports
/// as markdown only, so print plus an audience is a 400.
class _FakeApi extends ApiClient {
  _FakeApi()
    : super(
        baseUrl: 'http://127.0.0.1:9/api/v1',
        readAccess: () async => null,
        readRefresh: () async => null,
        writeTokens: (a, r) async {},
        clearTokens: () async {},
      );

  /// (format, audience) of every export request, in order.
  final calls = <(String, String?)>[];

  @override
  Future<String> exportPrd(String id, String format, {String? audience}) async {
    calls.add((format, audience));
    if (format != 'markdown' && audience != null) {
      throw ApiException(
        'audience projections export as markdown (format=markdown) for now',
        status: 400,
      );
    }
    return format == 'markdown' ? '# Plan' : '<h1>Plan</h1>';
  }
}

Future<_FakeApi> _pumpExport(
  WidgetTester tester, {
  String locale = 'en-US',
}) async {
  final session = Session();
  session.ready = true;
  final api = _FakeApi();
  session.api = api;
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
      child: const MaterialApp(home: ExportPage(prdId: 'p1')),
    ),
  );
  await tester.pumpAndSettle();
  return api;
}

/// The page's primary action, found by its icon so these tests do not depend
/// on what the button is called.
Finder _runButton() =>
    find.byWidgetPredicate((w) => w is BlockButton && w.icon == Icons.bolt);

Future<void> _chooseAudience(WidgetTester tester, String label) async {
  await tester.tap(find.byType(DropdownButton<String>));
  await tester.pumpAndSettle();
  await tester.tap(find.text(label).last);
  await tester.pumpAndSettle();
}

Future<void> _chooseFormat(WidgetTester tester, String label) async {
  await tester.tap(find.widgetWithText(BlockButton, label));
  await tester.pumpAndSettle();
}

/// The only small button on the page: the Download / Print action on the
/// preview card.
Finder _cardAction() => find.byWidgetPredicate(
  (w) => w is BlockButton && w.size == BlockButtonSize.sm,
);

/// Records what the page puts on the clipboard.
List<String> _captureClipboard(WidgetTester tester) {
  final copied = <String>[];
  final messenger = tester.binding.defaultBinaryMessenger;
  messenger.setMockMethodCallHandler(SystemChannels.platform, (call) async {
    if (call.method == 'Clipboard.setData') {
      copied.add((call.arguments as Map)['text'] as String);
    }
    return null;
  });
  addTearDown(
    () => messenger.setMockMethodCallHandler(SystemChannels.platform, null),
  );
  return copied;
}

void main() {
  testWidgets('Print after picking an audience exports the full document', (
    tester,
  ) async {
    final api = await _pumpExport(tester);

    await _chooseAudience(tester, 'Delivery');
    await _chooseFormat(tester, 'Print / Save as PDF');
    await tester.tap(_runButton());
    await tester.pumpAndSettle();

    // The server answers print plus an audience with a 400; the page must not
    // send the combination at all.
    expect(api.calls, [('print', null)]);
    expect(find.byType(BlockBanner), findsNothing);
    expect(find.text('<h1>Plan</h1>'), findsOneWidget);
  });

  testWidgets('the audience select is off in Print and on again in Markdown', (
    tester,
  ) async {
    await _pumpExport(tester);
    BlockSelect<String> select() =>
        tester.widget(find.byType(BlockSelect<String>));
    expect(select().enabled, isTrue);

    await _chooseFormat(tester, 'Print / Save as PDF');
    expect(select().enabled, isFalse);

    await _chooseFormat(tester, 'Download Markdown');
    expect(select().enabled, isTrue);
  });

  testWidgets('a picked audience does not survive a trip through Print', (
    tester,
  ) async {
    await _pumpExport(tester);
    String shown() => tester
        .widget<BlockSelect<String>>(find.byType(BlockSelect<String>))
        .value!;

    await _chooseAudience(tester, 'Delivery');
    expect(shown(), 'delivery');

    // Print is always the whole document, and the select says so.
    await _chooseFormat(tester, 'Print / Save as PDF');
    expect(shown(), '');
    await _chooseFormat(tester, 'Download Markdown');
    expect(shown(), '');
  });

  testWidgets('Markdown with an audience still asks for that projection', (
    tester,
  ) async {
    final api = await _pumpExport(tester);

    await _chooseAudience(tester, 'Delivery');
    await tester.tap(_runButton());
    await tester.pumpAndSettle();

    expect(api.calls, [('markdown', 'delivery')]);
    expect(find.byType(BlockBanner), findsNothing);
  });

  for (final (locale, label, retired) in [
    ('en-US', 'Generate preview', 'Submit'),
    ('zh-CN', '生成预览', '提交'),
  ]) {
    testWidgets('the primary action is named for what it does ($locale)', (
      tester,
    ) async {
      final api = await _pumpExport(tester, locale: locale);

      // The button only fetches the export and previews it. "Submit" is the
      // generic form verb, and 提交 is this product's word for Commit.
      expect(find.text(retired), findsNothing);
      await tester.tap(find.widgetWithText(BlockButton, label));
      await tester.pumpAndSettle();

      expect(api.calls, [('markdown', null)]);
      expect(find.text('# Plan'), findsOneWidget);
    });
  }

  // flutter test runs off the web, so download_stub.dart is the helper in play:
  // downloadTextFile and openTextInNewTab both report false, as on iOS/Android.
  testWidgets('Download copies the markdown where no file can be saved', (
    tester,
  ) async {
    final copied = _captureClipboard(tester);
    await _pumpExport(tester);
    await tester.tap(_runButton());
    await tester.pumpAndSettle();

    await tester.tap(_cardAction());
    await tester.pumpAndSettle();

    expect(copied, ['# Plan']);
    expect(
      find.text('Copied to the clipboard — paste it into a file'),
      findsOneWidget,
    );
  });

  testWidgets('Print copies the HTML where no tab and no file are possible', (
    tester,
  ) async {
    final copied = _captureClipboard(tester);
    await _pumpExport(tester);
    await _chooseFormat(tester, 'Print / Save as PDF');
    await tester.tap(_runButton());
    await tester.pumpAndSettle();

    await tester.tap(_cardAction());
    await tester.pumpAndSettle();

    expect(copied, ['<h1>Plan</h1>']);
    expect(
      find.text('Copied to the clipboard — paste it into a file'),
      findsOneWidget,
    );
  });
}
