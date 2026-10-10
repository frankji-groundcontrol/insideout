import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/app.dart';
import 'package:insideout/chrome/loop_focus.dart';
import 'package:insideout/router.dart';
import 'package:insideout/session/appearance.dart';
import 'package:insideout/session/boot.dart';
import 'package:insideout/session/session.dart';
import 'package:provider/provider.dart';

// Neither MaterialApp set a locale, so Material's own text (the deadline date
// picker, dialog buttons, the auth door's Close tooltip) stayed English in the
// zh-CN-first product. The Material locale now follows Appearance.locale, on
// the pre-boot splash and on the routed app, and tracks a live toggle.
Future<void> _pumpApp(
  WidgetTester tester,
  Appearance appearance,
  BootController boot,
) async {
  final session = Session()..ready = true;
  await tester.pumpWidget(
    MultiProvider(
      providers: [
        ChangeNotifierProvider.value(value: session),
        ChangeNotifierProvider.value(value: appearance),
        ChangeNotifierProvider.value(value: boot),
        ChangeNotifierProvider(create: (_) => LoopFocus()),
      ],
      child: InsideOutApp(router: buildRouter(session)),
    ),
  );
  await tester.pump();
}

String _cancelLabel(WidgetTester tester, Finder at) =>
    MaterialLocalizations.of(tester.element(at)).cancelButtonLabel;

void main() {
  testWidgets('the pre-boot splash localizes Material text to the app locale', (
    tester,
  ) async {
    final appearance = Appearance(
      store: MemorySettingsStore(),
      locale: 'zh-CN',
    );
    await _pumpApp(tester, appearance, BootController());

    expect(_cancelLabel(tester, find.byType(BootLoadingPage)), '取消');
  });

  testWidgets('the routed app follows the locale, including a live toggle', (
    tester,
  ) async {
    final appearance = Appearance(
      store: MemorySettingsStore(),
      locale: 'zh-CN',
    );
    final boot = BootController();
    await _pumpApp(tester, appearance, boot);
    boot.complete();
    await tester.pumpAndSettle();

    expect(_cancelLabel(tester, find.byType(Scaffold).first), '取消');

    await appearance.toggleLocale();
    await tester.pumpAndSettle();

    expect(_cancelLabel(tester, find.byType(Scaffold).first), 'Cancel');
  });
}
