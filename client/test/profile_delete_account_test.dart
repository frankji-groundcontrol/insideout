import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:insideout/api/client.dart';
import 'package:insideout/api/errors.dart';
import 'package:insideout/api/models.dart';
import 'package:insideout/api/requests.dart';
import 'package:insideout/chrome/loop_focus.dart';
import 'package:insideout/features/profile/profile_page.dart';
import 'package:insideout/session/appearance.dart';
import 'package:insideout/session/session.dart';
import 'package:insideout/session/token_store.dart';
import 'package:provider/provider.dart';

// rv5-FE-07, PARA-03: the profile page deletes the account after asking for
// the current password, clears the session and returns to the landing page.
class _Api extends ApiClient {
  _Api({this.refuse = false})
    : super(
        baseUrl: 'http://127.0.0.1:9/api/v1',
        readAccess: () async => null,
        readRefresh: () async => null,
        writeTokens: (a, r) async {},
        clearTokens: () async {},
      );

  final bool refuse;
  final passwords = <String>[];

  @override
  Future<void> deleteMe(String password) async {
    passwords.add(password);
    if (refuse) {
      throw ApiException('current password does not match', status: 403);
    }
  }
}

Future<Session> _pump(WidgetTester tester, _Api api) async {
  final session = Session(store: MemoryTokenStore())
    ..ready = true
    ..api = api
    ..user = UserProfile(
      id: 'u-me',
      email: 'me@test.local',
      username: 'me',
      bio: '',
      keywords: const [],
    );
  await tester.binding.setSurfaceSize(const Size(1400, 1600));
  addTearDown(() => tester.binding.setSurfaceSize(null));
  await tester.pumpWidget(
    MultiProvider(
      providers: [
        ChangeNotifierProvider.value(value: session),
        ChangeNotifierProvider(
          create: (_) =>
              Appearance(store: MemorySettingsStore(), locale: 'en-US'),
        ),
        ChangeNotifierProvider(create: (_) => LoopFocus()),
      ],
      child: MaterialApp.router(
        routerConfig: GoRouter(
          initialLocation: '/profile',
          routes: [
            GoRoute(path: '/', builder: (_, _) => const Text('Landing')),
            GoRoute(path: '/profile', builder: (_, _) => const ProfilePage()),
          ],
        ),
      ),
    ),
  );
  await tester.pumpAndSettle();
  return session;
}

Future<void> _confirmWith(WidgetTester tester, String password) async {
  await tester.tap(find.text('Delete my account'));
  await tester.pumpAndSettle();
  await tester.enterText(find.byType(TextField).last, password);
  await tester.tap(find.text('Delete account forever'));
  await tester.pumpAndSettle();
}

void main() {
  test('the request carries the current password', () {
    final c = deleteMe('hunter2');
    expect(c.method, 'DELETE');
    expect(c.path, '/me');
    expect(c.body, {'password': 'hunter2'});
  });

  testWidgets('a confirmed deletion signs out and lands on the front page', (
    tester,
  ) async {
    final api = _Api();
    final session = await _pump(tester, api);
    await _confirmWith(tester, 'hunter2');
    expect(api.passwords, ['hunter2']);
    expect(session.user, isNull);
    expect(find.text('Landing'), findsOneWidget);
  });

  testWidgets('a wrong password keeps the account and says why', (
    tester,
  ) async {
    final api = _Api(refuse: true);
    final session = await _pump(tester, api);
    await _confirmWith(tester, 'nope');
    expect(session.user, isNotNull);
    expect(find.text('That is not your current password.'), findsOneWidget);
  });
}
