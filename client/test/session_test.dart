import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/api/models.dart';
import 'package:insideout/session/session.dart';
import 'package:insideout/session/token_store.dart';

const loginFixture = {
  'id': 'user-1',
  'email': 'ada@example.com',
  'username': 'ada',
  'bio': '',
  'keywords': <String>[],
  'accessToken': 'access-from-login',
  'refreshToken': 'refresh-from-login',
};

const refreshFixture = {
  'ok': true,
  'accessToken': 'access-rotated',
  'refreshToken': 'refresh-rotated',
};

void main() {
  test('login JSON persist stores both tokens via Session', () async {
    final store = MemoryTokenStore();
    final session = Session(store: store);
    final profile = UserProfile.fromJson(Map<String, dynamic>.from(loginFixture));
    await session.applyAuthProfile(profile);
    expect(await store.readAccess(), loginFixture['accessToken']);
    expect(await store.readRefresh(), loginFixture['refreshToken']);
    expect(session.isSignedIn, isTrue);
    expect(session.user!.id, 'user-1');
  });

  test('refresh body replaces the stored refresh token', () async {
    final store = MemoryTokenStore();
    final session = Session(store: store);
    await session.saveTokens('old-access', 'old-refresh');
    await session.applyRefreshBody(Map<String, dynamic>.from(refreshFixture));
    expect(await store.readAccess(), 'access-rotated');
    expect(await store.readRefresh(), 'refresh-rotated');
  });

  test('hydrate with no stored access leaves the user signed out', () async {
    final session = Session(store: MemoryTokenStore());
    await session.hydrate();
    expect(session.ready, isTrue);
    expect(session.isSignedIn, isFalse);
    expect(session.user, isNull);
  });

  test('hydrate uses stored access then me()', () async {
    final store = MemoryTokenStore();
    await store.write('stored-access', 'stored-refresh');
    final meUser = UserProfile.fromJson({
      'id': 'user-1',
      'email': 'ada@example.com',
      'username': 'ada',
      'bio': '',
      'keywords': <String>[],
    });
    var meCalls = 0;
    final session = Session(
      store: store,
      fetchMe: () async {
        meCalls++;
        return meUser;
      },
    );
    await session.hydrate();
    expect(meCalls, 1);
    expect(session.user!.id, meUser.id);
    expect(session.isSignedIn, isTrue);
  });

  test('hydrate refreshes then retries me when access is dead', () async {
    final store = MemoryTokenStore();
    await store.write('dead-access', 'live-refresh');
    var meCalls = 0;
    var refreshed = false;
    final session = Session(
      store: store,
      fetchMe: () async {
        meCalls++;
        if (!refreshed) throw StateError('expired');
        return UserProfile.fromJson({
          'id': 'user-1',
          'email': 'ada@example.com',
          'username': 'ada',
          'bio': '',
          'keywords': <String>[],
        });
      },
      refreshSession: () async {
        refreshed = true;
        await store.write('access-rotated', 'refresh-rotated');
      },
    );
    await session.hydrate();
    expect(refreshed, isTrue);
    expect(meCalls, 2);
    expect(await store.readRefresh(), 'refresh-rotated');
    expect(session.isSignedIn, isTrue);
  });

  test('clear drops stored tokens', () async {
    final store = MemoryTokenStore();
    final session = Session(store: store);
    await session.saveTokens('a', 'r');
    expect(await store.readAccess(), 'a');
    await session.clear();
    expect(await store.readAccess(), isNull);
    expect(await store.readRefresh(), isNull);
  });

  test('tokensFromAuthBody and tokensFromRefreshBody read fixture fields', () {
    final auth = tokensFromAuthBody(Map<String, dynamic>.from(loginFixture));
    expect(auth!.access, 'access-from-login');
    expect(auth.refresh, 'refresh-from-login');
    final rotated = tokensFromRefreshBody(Map<String, dynamic>.from(refreshFixture));
    expect(rotated.access, 'access-rotated');
    expect(rotated.refresh, 'refresh-rotated');
  });
}
