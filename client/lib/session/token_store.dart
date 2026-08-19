import 'package:flutter_secure_storage/flutter_secure_storage.dart';

import '../api/errors.dart';

class TokenPair {
  const TokenPair(this.access, this.refresh);

  final String access;
  final String refresh;
}

/// Reads top-level tokens from login/register JSON (same shape as [UserProfile]).
TokenPair? tokensFromAuthBody(Map<String, dynamic> body) {
  final access = body['accessToken'] as String?;
  final refresh = body['refreshToken'] as String?;
  if (access == null || access.isEmpty || refresh == null || refresh.isEmpty) {
    return null;
  }
  return TokenPair(access, refresh);
}

/// Reads tokens from POST /auth/refresh JSON `{ok, accessToken, refreshToken}`.
TokenPair tokensFromRefreshBody(Map<String, dynamic> body) {
  final access = body['accessToken'] as String?;
  final refresh = body['refreshToken'] as String?;
  if (access == null || access.isEmpty || refresh == null || refresh.isEmpty) {
    throw ApiException('invalid refresh response');
  }
  return TokenPair(access, refresh);
}

abstract class TokenStore {
  Future<String?> readAccess();
  Future<String?> readRefresh();
  Future<void> write(String access, String refresh);
  Future<void> delete();
}

class MemoryTokenStore implements TokenStore {
  String? access;
  String? refresh;

  @override
  Future<String?> readAccess() async => access;

  @override
  Future<String?> readRefresh() async => refresh;

  @override
  Future<void> write(String access, String refresh) async {
    this.access = access;
    this.refresh = refresh;
  }

  @override
  Future<void> delete() async {
    access = null;
    refresh = null;
  }
}

class SecureTokenStore implements TokenStore {
  SecureTokenStore([FlutterSecureStorage? storage]) : _storage = storage ?? const FlutterSecureStorage();

  static const accessKey = 'insideout_access';
  static const refreshKey = 'insideout_refresh';

  final FlutterSecureStorage _storage;

  @override
  Future<String?> readAccess() => _storage.read(key: accessKey);

  @override
  Future<String?> readRefresh() => _storage.read(key: refreshKey);

  @override
  Future<void> write(String access, String refresh) async {
    await _storage.write(key: accessKey, value: access);
    await _storage.write(key: refreshKey, value: refresh);
  }

  @override
  Future<void> delete() async {
    await _storage.delete(key: accessKey);
    await _storage.delete(key: refreshKey);
  }
}
