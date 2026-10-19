import 'package:flutter/foundation.dart';

import '../api/client.dart';
import '../api/errors.dart';
import '../api/models.dart';
import 'token_store.dart';

const defaultApiBase = String.fromEnvironment(
  'API_BASE',
  defaultValue: 'http://127.0.0.1:8080/api/v1',
);

// Names that mean this device, or the Android emulator's alias for its host.
const _loopbackHosts = {'localhost', '127.0.0.1', '::1', '10.0.2.2'};

/// A release build must not send the bearer and refresh tokens in clear text
/// (rv4-FR-F212, F209, F11). android:usesCleartextTraffic and a network
/// security config govern the platform HTTP stack only; dio runs on dart:io
/// sockets and never reads them, so a manifest alone left
/// `--dart-define=API_BASE=http://<lan host>/...` working. This is the guard:
/// it throws unless [base] is https, a plain path (the web build, same origin
/// behind nginx) or http to this device. Debug and profile builds keep every
/// base for the dev loop. [releaseMode] is injectable because `flutter test`
/// is always a debug build. The message leaves [base] out: it may hold a login.
void checkApiBase(String base, {bool releaseMode = kReleaseMode}) {
  if (!releaseMode) return;
  final uri = Uri.tryParse(base);
  final ok =
      uri != null &&
      switch (uri.scheme) {
        'https' => true,
        'http' => _loopbackHosts.contains(uri.host),
        '' => !uri.hasAuthority,
        _ => false,
      };
  if (ok) return;
  throw StateError(
    'API_BASE must be an https URL in a release build (or http to localhost, '
    '127.0.0.1, ::1 or 10.0.2.2). dart:io ignores android:usesCleartextTraffic, '
    'so nothing else keeps the bearer and refresh tokens off the network in '
    'clear text. Rebuild with --dart-define=API_BASE=https://<server>/api/v1.',
  );
}

class Session extends ChangeNotifier {
  Session({
    TokenStore? store,
    String? apiBase,
    Future<UserProfile> Function()? fetchMe,
    Future<void> Function()? refreshSession,
    bool releaseMode = kReleaseMode,
  }) : _store = store ?? SecureTokenStore() {
    _fetchMe = fetchMe;
    _refreshSession = refreshSession;
    final base = apiBase ?? defaultApiBase;
    checkApiBase(base, releaseMode: releaseMode);
    api = ApiClient(
      baseUrl: base,
      readAccess: _store.readAccess,
      readRefresh: _store.readRefresh,
      writeTokens: saveTokens,
      clearTokens: clear,
    );
  }

  final TokenStore _store;
  // Non-final for widget tests: a page under test can inject a fake.
  late ApiClient api;
  late final Future<UserProfile> Function()? _fetchMe;
  late final Future<void> Function()? _refreshSession;

  UserProfile? user;
  bool ready = false;

  bool get isSignedIn => user != null;

  Future<UserProfile> _me() => (_fetchMe ?? api.me)();

  Future<void> _refresh() => (_refreshSession ?? api.refresh)();

  Future<void> hydrate() async {
    try {
      await _hydrate().timeout(const Duration(seconds: 15));
    } on ApiException catch (e) {
      user = null;
      if (e.status == 401 || e.status == 403) {
        await clear();
      }
    } catch (_) {
      user = null;
    } finally {
      ready = true;
      notifyListeners();
    }
  }

  Future<void> _hydrate() async {
    final token = await _store.readAccess();
    if (token == null || token.isEmpty) {
      user = null;
      return;
    }
    try {
      user = await _me();
    } catch (_) {
      await _refresh();
      user = await _me();
    }
  }

  Future<void> saveTokens(String access, String refresh) async {
    await _store.write(access, refresh);
  }

  Future<void> applyAuthProfile(UserProfile profile) async {
    final pair = tokensFromAuthBody({
      'accessToken': profile.accessToken,
      'refreshToken': profile.refreshToken,
    });
    if (pair != null) {
      await saveTokens(pair.access, pair.refresh);
    }
    user = profile;
    notifyListeners();
  }

  Future<void> applyRefreshBody(Map<String, dynamic> body) async {
    final pair = tokensFromRefreshBody(body);
    await saveTokens(pair.access, pair.refresh);
  }

  Future<void> register(
    String email,
    String password,
    String username, {
    String inviteCode = '',
  }) async {
    await applyAuthProfile(
      await api.register(email, password, username, inviteCode: inviteCode),
    );
  }

  Future<void> login(String email, String password) async {
    await applyAuthProfile(await api.login(email, password));
  }

  /// Signs in from an emailed magic-link token — same session state as a
  /// password login once the server accepts the token.
  Future<void> consumeMagicLink(String token) async {
    await applyAuthProfile(await api.consumeMagicLink(token));
  }

  /// Signs in with an emailed 6-digit code — same session state as a
  /// password login once the server accepts the code.
  Future<void> loginWithCode(String email, String code) async {
    await applyAuthProfile(await api.verifyLoginCode(email, code));
  }

  Future<void> logout() async {
    try {
      await api.logout();
    } finally {
      user = null;
      await clear();
      notifyListeners();
    }
  }

  Future<void> applyUser(UserProfile profile) async {
    user = profile;
    notifyListeners();
  }

  Future<void> clear() async {
    user = null;
    await _store.delete();
    notifyListeners();
  }
}
