import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/api/errors.dart';
import 'package:insideout/features/auth/reset_password_page.dart';
import 'package:insideout/session/appearance.dart';
import 'package:insideout/session/session.dart';
import 'package:insideout/theme/block_field.dart';
import 'package:provider/provider.dart';

import '_auth_test_helpers.dart';

/// FE-06: the reset page told every failure it was a dead link. A short
/// password, a throttle and a network fault each get their own message; only
/// a refused token reads as an invalid or expired link.
class _FailingApi extends QuietAuthApi {
  _FailingApi(this.failure);

  final Object failure;
  int calls = 0;

  @override
  Future<void> resetPassword(String token, String password) async {
    calls++;
    throw failure;
  }
}

Finder _field(String label) => find.descendant(
  of: find.widgetWithText(BlockField, label),
  matching: find.byType(TextField),
);

Future<_FailingApi> _submit(
  WidgetTester tester,
  Object failure, {
  String password = 'newpass123',
}) async {
  final api = _FailingApi(failure);
  final session = Session()
    ..ready = true
    ..api = api;
  await tester.pumpWidget(
    MultiProvider(
      providers: [
        ChangeNotifierProvider.value(value: session),
        ChangeNotifierProvider(
          create: (_) =>
              Appearance(store: MemorySettingsStore(), locale: 'en-US'),
        ),
      ],
      child: const MaterialApp(home: ResetPasswordPage(token: 'T0KEN')),
    ),
  );
  await tester.pumpAndSettle();
  await tester.tap(find.text('Reset on this page'));
  await tester.pumpAndSettle();
  await tester.enterText(_field('New password'), password);
  await tester.enterText(_field('Confirm new password'), password);
  await tester.tap(find.text('Reset password'));
  await tester.pumpAndSettle();
  return api;
}

void main() {
  testWidgets('a short password is refused before anything is sent', (
    tester,
  ) async {
    final api = await _submit(tester, StateError('unused'), password: 'short');
    expect(api.calls, 0);
    expect(find.text('Use at least 8 characters.'), findsOneWidget);
  });

  testWidgets('a refused token reads as a dead link', (tester) async {
    await _submit(tester, ApiException('invalid', status: 401));
    expect(find.textContaining('invalid or has expired'), findsOneWidget);
  });

  testWidgets('a throttle says wait, not dead link', (tester) async {
    await _submit(tester, ApiException('slow down', status: 429));
    expect(find.textContaining('Too many attempts'), findsOneWidget);
    expect(find.textContaining('invalid or has expired'), findsNothing);
  });

  testWidgets('a network fault says the link still works', (tester) async {
    await _submit(
      tester,
      DioException(
        requestOptions: RequestOptions(path: '/auth/reset-password'),
      ),
    );
    expect(find.textContaining('Could not reach the server'), findsOneWidget);
    expect(find.textContaining('invalid or has expired'), findsNothing);
  });
}
