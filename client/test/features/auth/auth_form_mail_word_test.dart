import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/features/auth/auth_door.dart';
import 'package:insideout/features/auth/auth_form.dart';
import 'package:insideout/session/appearance.dart';
import 'package:insideout/session/session.dart';
import 'package:provider/provider.dart';

import '_auth_test_helpers.dart';

// rv5-PARA-05: the server says per send whether the email was handed to a
// provider (accepted) or only written to its log (logged). The sent panels
// say so on logged instead of promising an email.
class _LoggedMailApi extends QuietAuthApi {
  _LoggedMailApi(this.word);

  final String word;

  @override
  Future<String?> requestMagicLink(String email) async => word;

  @override
  Future<String?> requestLoginCode(String email) async => word;
}

const _logged =
    'This server did not send the email: it was only written to the server log. Ask the server operator for it.';

Future<void> _pump(WidgetTester tester, String word) async {
  final session = Session()
    ..ready = true
    ..api = _LoggedMailApi(word);
  await tester.pumpWidget(
    MultiProvider(
      providers: [
        ChangeNotifierProvider.value(value: session),
        ChangeNotifierProvider(
          create: (_) =>
              Appearance(store: MemorySettingsStore(), locale: 'en-US'),
        ),
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
  await tester.pumpAndSettle();
}

Future<void> _send(WidgetTester tester, String mode, String submit) async {
  await tester.tap(find.text(mode));
  await tester.pump();
  await tester.enterText(find.byType(TextField).first, 'me@test.local');
  await tester.tap(find.text(submit));
  await tester.pumpAndSettle();
}

void main() {
  testWidgets('a logged magic link says it was not emailed', (tester) async {
    await _pump(tester, 'logged');
    await _send(tester, 'Email link', 'Send sign-in link');
    expect(find.text(_logged), findsOneWidget);
  });

  testWidgets('a logged sign-in code says it was not emailed', (tester) async {
    await _pump(tester, 'logged');
    await _send(tester, 'Email code', 'Send code');
    expect(find.text(_logged), findsOneWidget);
  });

  testWidgets('an accepted send shows no such notice', (tester) async {
    await _pump(tester, 'accepted');
    await _send(tester, 'Email link', 'Send sign-in link');
    expect(find.text(_logged), findsNothing);
  });
}
