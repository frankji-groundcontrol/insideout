import 'package:flutter/foundation.dart';

import '../api/client.dart';
import '../api/models.dart';
import 'token_store.dart';

const defaultApiBase = String.fromEnvironment(
  'API_BASE',
  defaultValue: 'http://127.0.0.1:8080/api/v1',
);

class Session extends ChangeNotifier {
  Session({
    TokenStore? store,
    String? apiBase,
    Future<UserProfile> Function()? fetchMe,
    Future<void> Function()? refreshSession,
  }) : _store = store ?? SecureTokenStore() {
    _fetchMe = fetchMe;
    _refreshSession = refreshSession;
    api = ApiClient(
      baseUrl: apiBase ?? defaultApiBase,
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
    } catch (_) {
      user = null;
      await clear();
    } finally {
      ready = true;
      notifyListeners();
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

  Future<void> register(String email, String password, String username) async {
    await applyAuthProfile(await api.register(email, password, username));
  }

  Future<void> login(String email, String password) async {
    await applyAuthProfile(await api.login(email, password));
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
    await _store.delete();
  }
}
