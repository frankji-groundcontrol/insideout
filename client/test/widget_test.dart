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
    await tester.pumpWidget(
      MultiProvider(
        providers: [
          ChangeNotifierProvider.value(value: session),
          ChangeNotifierProvider(create: (_) => Appearance(store: MemorySettingsStore(), locale: 'en-US')),
        ],
        child: const MaterialApp(home: LandingPage()),
      ),
    );
    expect(find.text('Start shaping'), findsOneWidget);
    expect(find.text('Log in'), findsOneWidget);
  });
}
