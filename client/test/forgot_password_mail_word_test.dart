import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/features/auth/forgot_password_page.dart';
import 'package:insideout/session/appearance.dart';
import 'package:insideout/session/session.dart';
import 'package:provider/provider.dart';

import '_auth_test_helpers.dart';

// rv5-PARA-05: the forgot-password page reads the server's mail word like the
// sign-in link and code panels, and says when the reset mail was only logged.
class _Api extends QuietAuthApi {
  _Api(this.word);

  final String word;

  @override
  Future<String?> forgotPassword(String email) async => word;
}

const _logged =
    'This server did not send the email: it was only written to the server log. Ask the server operator for it.';

Future<void> _request(WidgetTester tester, String word) async {
  await tester.pumpWidget(
    MultiProvider(
      providers: [
        ChangeNotifierProvider.value(
          value: Session()
            ..ready = true
            ..api = _Api(word),
        ),
        ChangeNotifierProvider(
          create: (_) =>
              Appearance(store: MemorySettingsStore(), locale: 'en-US'),
        ),
      ],
      child: const MaterialApp(home: ForgotPasswordPage()),
    ),
  );
  await tester.pumpAndSettle();
  await tester.enterText(find.byType(TextField), 'me@test.local');
  await tester.tap(find.text('Send reset link'));
  await tester.pumpAndSettle();
}

void main() {
  testWidgets('a logged reset mail says it was not emailed', (tester) async {
    await _request(tester, 'logged');
    expect(find.text(_logged), findsOneWidget);
  });

  testWidgets('an accepted reset mail shows no such notice', (tester) async {
    await _request(tester, 'accepted');
    expect(find.text(_logged), findsNothing);
    expect(find.text('Back to login'), findsOneWidget);
  });
}
