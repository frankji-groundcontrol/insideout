part of 'client.dart';

/// Sign-in, session and account verbs: login flows, invitations, profile.
mixin _AuthVerbs on _ApiCore {
  Future<UserProfile> register(
    String email,
    String password,
    String username, {
    String inviteCode = '',
  }) async {
    final res = await _dio.post(
      '/auth/register',
      data: {
        'email': email,
        'password': password,
        'username': username,
        if (inviteCode.isNotEmpty) 'inviteCode': inviteCode,
      },
    );
    return _parse(res, (d) => UserProfile.fromJson(d as Map<String, dynamic>));
  }

  Future<UserProfile> login(String email, String password) async {
    final res = await _dio.post(
      '/auth/login',
      data: {'email': email, 'password': password},
    );
    return _parse(res, (d) => UserProfile.fromJson(d as Map<String, dynamic>));
  }

  /// Requests a magic-link email for [email]. Always resolves when the
  /// server accepts the request — the server answers 200 regardless of
  /// whether the account exists.
  Future<void> requestMagicLink(String email) async {
    final res = await _dio.post('/auth/magic-link', data: {'email': email});
    await _parse(res, (d) => d);
  }

  /// Exchanges an emailed magic-link token for a full session (same shape
  /// as login). Throws with status 401 when the link is invalid, expired,
  /// or already used.
  Future<UserProfile> consumeMagicLink(String token) async {
    final res = await _dio.post(
      '/auth/magic-link/consume',
      data: {'token': token},
    );
    return _parse(res, (d) => UserProfile.fromJson(d as Map<String, dynamic>));
  }

  /// Requests a 6-digit sign-in code email for [email]. Always resolves
  /// when the server accepts the request (uniform 200 contract).
  Future<void> requestLoginCode(String email) async {
    final res = await _dio.post('/auth/login-code', data: {'email': email});
    await _parse(res, (d) => d);
  }

  /// Exchanges an emailed sign-in code for a full session (same shape as
  /// login). Throws with status 401 when the code is wrong, expired, or
  /// exhausted by failed attempts.
  Future<UserProfile> verifyLoginCode(String email, String code) async {
    final res = await _dio.post(
      '/auth/login-code/verify',
      data: {'email': email, 'code': code},
    );
    return _parse(res, (d) => UserProfile.fromJson(d as Map<String, dynamic>));
  }

  /// Requests a password-reset email for [email] (uniform 200 contract).
  Future<void> forgotPassword(String email) async {
    final res = await _dio.post(
      '/auth/forgot-password',
      data: {'email': email},
    );
    await _parse(res, (d) => d);
  }

  /// Sets a new password with an emailed reset token. On success every
  /// existing session of that account has been revoked server-side, so the
  /// next step is always a fresh login.
  Future<void> resetPassword(String token, String password) async {
    final res = await _dio.post(
      '/auth/reset-password',
      data: {'token': token, 'password': password},
    );
    await _parse(res, (d) => d);
  }

  /// Public signup policy — the register form needs to know whether an
  /// invitation code field should be shown. Failing open to
  /// invitation-only would be wrong; the caller decides the fallback.
  Future<String> authConfig() async {
    final res = await _dio.get('/auth/config');
    final data = await _parse(res, (d) => d as Map<String, dynamic>);
    return data['signupMode'] as String? ?? 'invitation_only';
  }

  /// Admin: mint an email-bound invitation. Returns the one-time code and
  /// signup URL (the only place the code ever appears).
  Future<Map<String, dynamic>> createInvitation(String email) async {
    final res = await _dio.post('/admin/invitations', data: {'email': email});
    return _parse(res, (d) => d as Map<String, dynamic>);
  }

  /// Admin: invitation ledger without codes.
  Future<List<Map<String, dynamic>>> listInvitations() async {
    final res = await _dio.get('/admin/invitations');
    final data = await _parse(res, (d) => d);
    return ((data as List?) ?? const []).cast<Map<String, dynamic>>();
  }

  /// Admin: revoke a still-usable invitation.
  Future<void> revokeInvitation(String id) async {
    final res = await _dio.post('/admin/invitations/$id/revoke', data: {});
    await _parse(res, (d) => d);
  }

  /// Admin: switch registration policy (invitation_only | open | closed).
  Future<void> setSignupMode(String mode) async {
    final res = await _dio.put('/admin/settings', data: {'signupMode': mode});
    await _parse(res, (d) => d);
  }

  Future<void> logout() async {
    final refresh = await _readRefresh();
    await _dio.post('/auth/logout', data: {'refreshToken': refresh ?? ''});
    await _clearTokens();
  }

  Future<UserProfile> me() async {
    final res = await _dio.get('/me');
    return _parse(res, (d) => UserProfile.fromJson(d as Map<String, dynamic>));
  }

  Future<UserProfile> updateMe({
    required String username,
    required String bio,
    required List<String> keywords,
  }) async {
    final res = await _dio.patch(
      '/me',
      data: {'username': username, 'bio': bio, 'keywords': keywords},
    );
    return _parse(res, (d) => UserProfile.fromJson(d as Map<String, dynamic>));
  }
}
