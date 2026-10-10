import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:insideout/api/client.dart';
import 'package:insideout/router.dart';
import 'package:insideout/session/appearance.dart';
import 'package:insideout/session/session.dart';
import 'package:insideout/theme/block_field.dart';
import 'package:provider/provider.dart';

class _FakeApi extends ApiClient {
  _FakeApi()
    : super(
        baseUrl: 'http://127.0.0.1:9/api/v1',
        readAccess: () async => null,
        readRefresh: () async => null,
        writeTokens: (a, r) async {},
        clearTokens: () async {},
      );

  /// What the reset-password page submitted, if it did.
  ({String token, String password})? reset;

  @override
  Future<String> authConfig() async => 'invitation_only';

  @override
  Future<void> resetPassword(String token, String password) async {
    reset = (token: token, password: password);
  }
}

/// Records the magic-link token the page consumes, then fails so the page
/// stays on its dead-link state instead of navigating away.
class _RecordingSession extends Session {
  String? magicToken;

  @override
  Future<void> consumeMagicLink(String token) async {
    magicToken = token;
    throw StateError('stay on the dead-link state');
  }
}

Future<GoRouter> _pumpRouter(WidgetTester tester, Session session) async {
  final router = buildRouter(session);
  await tester.pumpWidget(
    MultiProvider(
      providers: [
        ChangeNotifierProvider.value(value: session),
        ChangeNotifierProvider(
          create: (_) =>
              Appearance(store: MemorySettingsStore(), locale: 'en-US'),
        ),
      ],
      child: MaterialApp.router(routerConfig: router),
    ),
  );
  await tester.pumpAndSettle();
  return router;
}

Finder _field(String label) => find.descendant(
  of: find.widgetWithText(BlockField, label),
  matching: find.byType(TextField),
);

void main() {
  test('productRouteBuilders registers every current surface', () {
    expect(productRouteBuilders.keys.toSet(), {
      '/',
      '/login',
      '/register',
      '/forgot-password',
      '/reset-password',
      '/magic-link',
      '/dashboard',
      '/profile',
      '/workspace/:id',
      '/workspace/:id/ideas',
      '/workspace/:id/settings',
      '/projects/:id',
      '/projects/:id/roadmap',
      '/prd/:id',
      '/prd/:id/revisions',
      '/prd/:id/versions',
      '/prd/:id/view',
      '/prd/:id/export',
    });
  });

  testWidgets('register deep link pre-fills the invitation code', (
    tester,
  ) async {
    final session = Session();
    session.ready = true;
    session.api = _FakeApi();
    final router = await _pumpRouter(tester, session);

    router.go('/register?invite=CODE123');
    await tester.pumpAndSettle();

    final inviteField = tester.widget<TextField>(_field('Invitation code'));
    expect(inviteField.controller?.text, 'CODE123');
  });

  // The emailed links carry their secret as ?token=; a router that drops it
  // (the FS-02 failure class for ?invite=) leaves a page that cannot work.
  testWidgets('magic-link deep link hands its token to the session', (
    tester,
  ) async {
    final session = _RecordingSession();
    session.ready = true;
    final router = await _pumpRouter(tester, session);

    router.go('/magic-link?token=T0KEN');
    await tester.pumpAndSettle();

    expect(session.magicToken, 'T0KEN');
  });

  testWidgets('reset-password deep link submits its token with the password', (
    tester,
  ) async {
    final session = Session();
    session.ready = true;
    final api = _FakeApi();
    session.api = api;
    final router = await _pumpRouter(tester, session);

    router.go('/reset-password?token=T0KEN');
    await tester.pumpAndSettle();
    await tester.enterText(_field('New password'), 'newpass123');
    await tester.enterText(_field('Confirm new password'), 'newpass123');
    await tester.tap(find.text('Reset password'));
    await tester.pumpAndSettle();

    expect(api.reset, (token: 'T0KEN', password: 'newpass123'));
  });
}
