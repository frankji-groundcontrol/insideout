import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/features/auth/auth_form.dart';
import 'package:insideout/features/auth/auth_door.dart';
import 'package:insideout/session/appearance.dart';
import 'package:insideout/session/session.dart';
import 'package:provider/provider.dart';

void main() {
  testWidgets('login form switches to magic-link mode and back', (tester) async {
    final session = Session();
    session.ready = true;
    await tester.pumpWidget(
      MultiProvider(
        providers: [
          ChangeNotifierProvider.value(value: session),
          ChangeNotifierProvider(create: (_) => Appearance(store: MemorySettingsStore(), locale: 'en-US')),
        ],
        child: MaterialApp(
          home: AuthDoor(
            greeting: 'Log in',
            onClose: () {},
            child: const AuthForm(prompt: AuthPrompt.login),
          ),
        ),
      ),
    );

    // Password mode: email + password fields, both alternate entries visible.
    expect(find.text('Password'), findsOneWidget);
    expect(find.text('Email me a sign-in link'), findsOneWidget);
    expect(find.text('Forgot password?'), findsOneWidget);

    await tester.tap(find.text('Email me a sign-in link'));
    await tester.pump();

    // Magic mode: password field gone, send button in its place.
    expect(find.text('Password'), findsNothing);
    expect(find.text('Send sign-in link'), findsOneWidget);
    expect(find.text('Forgot password?'), findsNothing);
    expect(find.text('Use password instead'), findsOneWidget);

    await tester.tap(find.text('Use password instead'));
    await tester.pump();
    expect(find.text('Password'), findsOneWidget);
    expect(find.text('Email me a sign-in link'), findsOneWidget);
  });

  testWidgets('login form switches to code mode and back', (tester) async {
    final session = Session();
    session.ready = true;
    await tester.pumpWidget(
      MultiProvider(
        providers: [
          ChangeNotifierProvider.value(value: session),
          ChangeNotifierProvider(create: (_) => Appearance(store: MemorySettingsStore(), locale: 'en-US')),
        ],
        child: MaterialApp(
          home: AuthDoor(
            greeting: 'Log in',
            onClose: () {},
            child: const AuthForm(prompt: AuthPrompt.login),
          ),
        ),
      ),
    );

    // Password mode: the code entry sits next to the magic-link one.
    expect(find.text('Email me a sign-in code'), findsOneWidget);

    await tester.tap(find.text('Email me a sign-in code'));
    await tester.pump();

    // Code mode: password field gone, send-code button in its place.
    expect(find.text('Password'), findsNothing);
    expect(find.text('Send code'), findsOneWidget);
    expect(find.text('Forgot password?'), findsNothing);
    expect(find.text('Use password instead'), findsOneWidget);

    await tester.tap(find.text('Use password instead'));
    await tester.pump();
    expect(find.text('Password'), findsOneWidget);
    expect(find.text('Email me a sign-in code'), findsOneWidget);
  });
}
