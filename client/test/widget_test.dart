import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/features/landing/landing_page.dart';
import 'package:insideout/session/appearance.dart';
import 'package:insideout/session/session.dart';
import 'package:provider/provider.dart';

void main() {
  testWidgets('landing offers register when signed out', (tester) async {
    final session = Session();
    session.ready = true;
    await tester.binding.setSurfaceSize(const Size(1200, 5000));
    addTearDown(() => tester.binding.setSurfaceSize(null));
    await tester.pumpWidget(
      MultiProvider(
        providers: [
          ChangeNotifierProvider.value(value: session),
          ChangeNotifierProvider(create: (_) => Appearance(store: MemorySettingsStore(), locale: 'en-US')),
        ],
        child: const MaterialApp(home: LandingPage()),
      ),
    );
    expect(find.text('Start shaping'), findsNWidgets(2));
    expect(find.text('Log in'), findsOneWidget);
    expect(find.text('Capture the spark'), findsOneWidget);
    expect(find.text('Ready to press your first seal?'), findsOneWidget);

    await tester.tap(find.widgetWithText(OutlinedButton, 'Log in'));
    await tester.pump();
    expect(find.byType(TextField), findsNWidgets(2));
    expect(find.text('InsideOut'), findsWidgets);
  });
}
