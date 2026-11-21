import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/app.dart';
import 'package:insideout/chrome/loop_focus.dart';
import 'package:insideout/router.dart';
import 'package:insideout/session/appearance.dart';
import 'package:insideout/session/boot.dart';
import 'package:insideout/session/session.dart';
import 'package:provider/provider.dart';

void main() {
  testWidgets('boot loading page shows mark + spinner, no product text', (
    tester,
  ) async {
    await tester.pumpWidget(const MaterialApp(home: BootLoadingPage()));
    await tester.pump();
    expect(find.byKey(const Key('boot-spinner')), findsOneWidget);
    expect(find.byKey(const Key('boot-mark')), findsOneWidget);
    expect(find.byType(Image), findsNothing);
    expect(find.byType(Text), findsNothing);
  });

  testWidgets('app gates on boot: loading page until complete, router after', (
    tester,
  ) async {
    final session = Session();
    session.ready = true;
    final boot = BootController();
    await tester.pumpWidget(
      MultiProvider(
        providers: [
          ChangeNotifierProvider.value(value: session),
          ChangeNotifierProvider(
            create: (_) =>
                Appearance(store: MemorySettingsStore(), locale: 'en-US'),
          ),
          ChangeNotifierProvider.value(value: boot),
          ChangeNotifierProvider(create: (_) => LoopFocus()),
        ],
        child: InsideOutApp(router: buildRouter(session)),
      ),
    );
    await tester.pump();
    expect(find.byKey(const Key('boot-spinner')), findsOneWidget);
    expect(find.text('InsideOut'), findsNothing); // nothing product-ish mounted

    boot.complete();
    await tester.pumpAndSettle();
    expect(find.byKey(const Key('boot-spinner')), findsNothing);
    expect(find.text('InsideOut'), findsWidgets); // landing wordmark via router
  });
}
