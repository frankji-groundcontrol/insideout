import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
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

  @override
  Future<String> authConfig() async => 'invitation_only';
}

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

    router.go('/register?invite=CODE123');
    await tester.pumpAndSettle();

    final inviteField = tester.widget<TextField>(
      find.descendant(
        of: find.widgetWithText(BlockField, 'Invitation code'),
        matching: find.byType(TextField),
      ),
    );
    expect(inviteField.controller?.text, 'CODE123');
  });
}
