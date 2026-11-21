import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/api/client.dart';
import 'package:insideout/api/errors.dart';
import 'package:insideout/chrome/loop_focus.dart';
import 'package:insideout/features/prd/export_page.dart';
import 'package:insideout/session/appearance.dart';
import 'package:insideout/session/session.dart';
import 'package:insideout/theme/block_button.dart';
import 'package:insideout/theme/block_chrome.dart';
import 'package:provider/provider.dart';

/// A preview belongs to the option that produced it. These tests pin that
/// changing the format or the audience drops the preview (and any error):
/// otherwise the old text renders under the new option's header, and the
/// Print action would open a stale markdown body as text/html.
class _FakeApi extends ApiClient {
  _FakeApi({this.failWith})
    : super(
        baseUrl: 'http://127.0.0.1:9/api/v1',
        readAccess: () async => null,
        readRefresh: () async => null,
        writeTokens: (a, r) async {},
        clearTokens: () async {},
      );

  /// Set once by a test to make the next export fail, as the server would.
  Object? failWith;

  int get calls => bodies.length;
  final bodies = <String>[];

  @override
  Future<String> exportPrd(String id, String format, {String? audience}) async {
    final failure = failWith;
    if (failure != null) {
      failWith = null;
      throw failure;
    }
    final body = switch ((format, audience)) {
      ('markdown', null) => '# Full plan',
      ('markdown', 'delivery') => '# Delivery view',
      _ => '<h1>Print</h1>',
    };
    bodies.add(body);
    return body;
  }
}

Future<_FakeApi> _pumpExport(WidgetTester tester, {_FakeApi? api}) async {
  final session = Session()
    ..ready = true
    ..api = api ?? _FakeApi();
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
      child: const MaterialApp(home: ExportPage(prdId: 'p1')),
    ),
  );
  await tester.pumpAndSettle();
  return session.api as _FakeApi;
}

/// The page's primary action, found by its icon so these tests do not depend
/// on what the button is called.
Finder _runButton() =>
    find.byWidgetPredicate((w) => w is BlockButton && w.icon == Icons.bolt);

Future<void> _chooseFormat(WidgetTester tester, String label) async {
  await tester.tap(find.widgetWithText(BlockButton, label));
  await tester.pumpAndSettle();
}

Future<void> _chooseAudience(WidgetTester tester, String label) async {
  await tester.tap(find.byType(DropdownButton<String>));
  await tester.pumpAndSettle();
  await tester.tap(find.text(label).last);
  await tester.pumpAndSettle();
}

void main() {
  testWidgets('switching the format drops the stale preview', (tester) async {
    final api = await _pumpExport(tester);
    await tester.tap(_runButton());
    await tester.pumpAndSettle();
    expect(find.text('# Full plan'), findsOneWidget);

    await _chooseFormat(tester, 'Print / Save as PDF');

    expect(find.text('# Full plan'), findsNothing);
    // Merely switching options must not fetch anything either.
    expect(api.calls, 1);
  });

  testWidgets('picking an audience drops the stale preview', (tester) async {
    await _pumpExport(tester);
    await tester.tap(_runButton());
    await tester.pumpAndSettle();
    expect(find.text('# Full plan'), findsOneWidget);

    await _chooseAudience(tester, 'Delivery');

    expect(find.text('# Full plan'), findsNothing);
  });

  testWidgets('switching the format drops a stale error too', (tester) async {
    await _pumpExport(
      tester,
      api: _FakeApi(failWith: ApiException('boom', status: 500)),
    );
    await tester.tap(_runButton());
    await tester.pumpAndSettle();
    expect(find.byType(BlockBanner), findsOneWidget);

    await _chooseFormat(tester, 'Print / Save as PDF');

    expect(find.byType(BlockBanner), findsNothing);
  });
}
